/*
Copyright 2022 The Tekton Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package trustedtask

import (
	"context"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/google/go-containerregistry/pkg/authn"
	imgname "github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	typesv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	faketekton "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/fake"
	remotetest "github.com/tektoncd/pipeline/test"
	"go.uber.org/zap/zaptest"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakek8s "k8s.io/client-go/kubernetes/fake"
	"knative.dev/pkg/logging"
)

var (
	// pipeline for testing
	prTypeMeta = metav1.TypeMeta{
		Kind:       pipeline.PipelineRunControllerName,
		APIVersion: "tekton.dev/v1beta1"}

	prObjectMeta = metav1.ObjectMeta{
		Name:        "pr",
		Namespace:   nameSpace,
		Annotations: map[string]string{},
	}
)

func init() {
	os.Setenv("SYSTEM_NAMESPACE", nameSpace)
	os.Setenv("WEBHOOK_SERVICEACCOUNT_NAME", serviceAccount)
}

func TestVerifyPipelineRun_APIPipeline(t *testing.T) {
	ctx := logging.WithLogger(context.Background(), zaptest.NewLogger(t).Sugar())
	k8sclient := fakek8s.NewSimpleClientset()

	// Get Signer
	signer, secretpath, err := getSignerFromFile(t, ctx, k8sclient)
	if err != nil {
		t.Fatal(err)
	}
	ctx = setupContext(ctx, k8sclient, secretpath)

	prWithoutTasks := &TrustedPipelineRun{
		PipelineRun: v1beta1.PipelineRun{
			TypeMeta:   prTypeMeta,
			ObjectMeta: prObjectMeta,
			Spec: v1beta1.PipelineRunSpec{
				PipelineSpec: &getUnsignedPipeline("unsigned").Spec,
			},
		}}

	prWithUnsignedTasks := prWithoutTasks.DeepCopy()
	unsignedTask := getUnsignedTask()
	prWithUnsignedTasks.Spec.PipelineSpec.Tasks = []v1beta1.PipelineTask{
		{
			TaskRef: &v1beta1.TaskRef{
				Name: unsignedTask.Name,
				Kind: "Task",
			},
		}}

	prWithSignedTasks := prWithoutTasks.DeepCopy()
	signedTask, err := getSignedTask(unsignedTask, signer)
	if err != nil {
		t.Fatal("fail to sign task", err)
	}
	prWithSignedTasks.Spec.PipelineSpec.Tasks = []v1beta1.PipelineTask{
		{TaskRef: &v1beta1.TaskRef{
			Name: signedTask.Name,
			Kind: "Task",
		}},
	}

	tektonClient := faketekton.NewSimpleClientset(unsignedTask, signedTask)

	tcs := []struct {
		name        string
		pipelineRun *TrustedPipelineRun
		wantErr     bool
	}{{
		name:        "Pipeline Run without tasks Pass Verification",
		pipelineRun: prWithoutTasks,
		wantErr:     false,
	}, {
		name:        "Pipeline Run Fail Verification with unsigned tasks",
		pipelineRun: prWithUnsignedTasks,
		wantErr:     true,
	}, {
		name:        "Pipeline Run Pass Verification with signed tasks",
		pipelineRun: prWithSignedTasks,
		wantErr:     false,
	},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.pipelineRun.verifyPipelineRun(ctx, k8sclient, tektonClient)
			if (err != nil) != tc.wantErr {
				t.Fatalf("verifyPipelineRun() get err %v, wantErr %t", err, tc.wantErr)
			}
		})
	}
}

func TestVerifyPipelineRun_PipelineOCIBundle(t *testing.T) {
	ctx := logging.WithLogger(context.Background(), zaptest.NewLogger(t).Sugar())
	k8sclient := fakek8s.NewSimpleClientset(sa)
	tektonClient := faketekton.NewSimpleClientset()

	// Get Signer
	signer, secretpath, err := getSignerFromFile(t, ctx, k8sclient)
	if err != nil {
		t.Fatal(err)
	}
	ctx = setupContext(ctx, k8sclient, secretpath)

	// Create registry server
	s := httptest.NewServer(registry.New())
	defer s.Close()
	u, _ := url.Parse(s.URL)

	unsignedPipeline := getUnsignedPipeline("unsigned")
	if _, err = pushPipelineImage(t, u, unsignedPipeline); err != nil {
		t.Fatal(err)
	}

	signedPipeline, err := getSignedPipeline(getUnsignedPipeline("signed"), signer)
	if err != nil {
		t.Fatal("fail to sign pipeline", err)
	}
	if _, err = pushPipelineImage(t, u, signedPipeline); err != nil {
		t.Fatal(err)
	}

	tamperedPipeline := signedPipeline.DeepCopy()
	tamperedPipeline.Name = "tampered"
	if _, err := pushPipelineImage(t, u, tamperedPipeline); err != nil {
		t.Fatal(err)
	}

	// OCI pipelineruns
	opr := v1beta1.PipelineRun{
		TypeMeta:   trTypeMeta,
		ObjectMeta: trObjectMeta,
		Spec: v1beta1.PipelineRunSpec{
			ServiceAccountName: sa.Name,
			PipelineRef: &v1beta1.PipelineRef{
				Name:   unsignedPipeline.Name,
				Bundle: u.Host + "/pipeline/" + unsignedPipeline.Name,
			},
		},
	}

	unsigned := &TrustedPipelineRun{PipelineRun: opr}

	signed := unsigned.DeepCopy()
	signed.Spec.PipelineRef.Name = signedPipeline.Name
	signed.Spec.PipelineRef.Bundle = u.Host + "/pipeline/" + signedPipeline.Name

	tampered := unsigned.DeepCopy()
	tampered.Spec.PipelineRef.Name = tamperedPipeline.Name
	tampered.Spec.PipelineRef.Bundle = u.Host + "/pipeline/" + tamperedPipeline.Name

	tcs := []struct {
		name        string
		pipelineRun *TrustedPipelineRun
		wantErr     bool
	}{{
		name:        "OCI Bundle Pass Verification",
		pipelineRun: signed,
		wantErr:     false,
	}, {
		name:        "OCI Bundle Fail Verification with tampered content",
		pipelineRun: tampered,
		wantErr:     true,
	}, {
		name:        "OCI Bundle Fail Verification without signature",
		pipelineRun: unsigned,
		wantErr:     true,
	},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.pipelineRun.verifyPipelineRun(ctx, k8sclient, tektonClient); (err != nil) != tc.wantErr {
				t.Errorf("verifyPipelineRun() get err: %v, wantErr: %t", err, tc.wantErr)
			}
		})
	}

}

func TestVerifyPipelineRun_TaskOCIBundle(t *testing.T) {
	ctx := logging.WithLogger(context.Background(), zaptest.NewLogger(t).Sugar())
	k8sclient := fakek8s.NewSimpleClientset(sa)
	tektonClient := faketekton.NewSimpleClientset()

	// Get Signer
	signer, secretpath, err := getSignerFromFile(t, ctx, k8sclient)
	if err != nil {
		t.Fatal(err)
	}
	ctx = setupContext(ctx, k8sclient, secretpath)

	// Create registry server
	s := httptest.NewServer(registry.New())
	defer s.Close()
	u, _ := url.Parse(s.URL)

	// signed pipeline bundle with unsigned task bundle
	pipelineWithUnsignedTask := getUnsignedPipeline("unsigned")
	unsignedTask := getUnsignedTask()
	pipelineWithUnsignedTask.Spec.Tasks = []v1beta1.PipelineTask{
		{TaskRef: &v1beta1.TaskRef{
			Name:   unsignedTask.Name,
			Kind:   "Task",
			Bundle: u.Host + "/task/" + unsignedTask.Name,
		}}}
	pipelineWithUnsignedTask, err = getSignedPipeline(pipelineWithUnsignedTask, signer)
	if err != nil {
		t.Fatal("fail to sign pipeline", err)
	}
	unsigned := &TrustedPipelineRun{
		v1beta1.PipelineRun{
			TypeMeta:   prTypeMeta,
			ObjectMeta: prObjectMeta,
			Spec: v1beta1.PipelineRunSpec{
				ServiceAccountName: sa.Name,
				PipelineRef: &v1beta1.PipelineRef{
					Name:   pipelineWithUnsignedTask.Name,
					Bundle: u.Host + "/pipeline/" + pipelineWithUnsignedTask.Name,
				},
			},
		},
	}
	// Push OCI bundle
	if _, err = pushPipelineImage(t, u, pipelineWithUnsignedTask); err != nil {
		t.Fatal(err)
	}
	if _, err = pushOCIImage(t, u, unsignedTask); err != nil {
		t.Fatal(err)
	}

	// signed pipelineref with signed taskref
	pipelineWithSignedTask := getUnsignedPipeline("signed")
	signedTask, err := getSignedTask(unsignedTask, signer)
	if err != nil {
		t.Fatal("fail to sign task", err)
	}
	pipelineWithSignedTask.Spec.Tasks = []v1beta1.PipelineTask{
		{TaskRef: &v1beta1.TaskRef{
			Name:   signedTask.Name,
			Kind:   "Task",
			Bundle: u.Host + "/task/" + signedTask.Name,
		}}}
	pipelineWithSignedTask, err = getSignedPipeline(pipelineWithSignedTask, signer)
	if err != nil {
		t.Fatal("fail to sign pipeline", err)
	}
	signed := unsigned.DeepCopy()
	signed.Spec.PipelineRef = &v1beta1.PipelineRef{
		Name:   pipelineWithSignedTask.Name,
		Bundle: u.Host + "/pipeline/" + pipelineWithSignedTask.Name,
	}
	// Push OCI bundle
	if _, err = pushPipelineImage(t, u, pipelineWithSignedTask); err != nil {
		t.Fatal(err)
	}
	if _, err = pushOCIImage(t, u, signedTask); err != nil {
		t.Fatal(err)
	}

	// signed pipelineref with tampered taskref
	pipelineWithTamperedTask := getUnsignedPipeline("tampered")
	tamperedTask := signedTask.DeepCopy()
	tamperedTask.Name = "tampered"
	pipelineWithTamperedTask.Spec.Tasks = []v1beta1.PipelineTask{
		{TaskRef: &v1beta1.TaskRef{
			Name:   tamperedTask.Name,
			Kind:   "Task",
			Bundle: u.Host + "/pipeline/" + tamperedTask.Name,
		}}}
	pipelineWithTamperedTask, err = getSignedPipeline(pipelineWithTamperedTask, signer)
	if err != nil {
		t.Fatal("fail to sign pipeline", err)
	}
	tampered := signed.DeepCopy()
	tampered.Spec.PipelineRef = &v1beta1.PipelineRef{
		Name:   pipelineWithTamperedTask.Name,
		Bundle: u.Host + "/pipeline/" + pipelineWithTamperedTask.Name,
	}
	// Push OCI bundle
	if _, err = pushPipelineImage(t, u, pipelineWithTamperedTask); err != nil {
		t.Fatal(err)
	}
	if _, err = pushOCIImage(t, u, tamperedTask); err != nil {
		t.Fatal(err)
	}

	tcs := []struct {
		name        string
		pipelineRun *TrustedPipelineRun
		wantErr     bool
	}{{
		name:        "OCI Bundle Pass Verification",
		pipelineRun: signed,
		wantErr:     false,
	}, {
		name:        "OCI Bundle Fail Verification with tampered content",
		pipelineRun: tampered,
		wantErr:     true,
	}, {
		name:        "OCI Bundle Fail Verification without signature",
		pipelineRun: unsigned,
		wantErr:     true,
	},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.pipelineRun.verifyPipelineRun(ctx, k8sclient, tektonClient); (err != nil) != tc.wantErr {
				t.Errorf("verifyPipelineRun() get err: %v, wantErr: %t", err, tc.wantErr)
			}
		})
	}

}

func TestVerifyPipelineRun_PipelineRef(t *testing.T) {
	ctx := logging.WithLogger(context.Background(), zaptest.NewLogger(t).Sugar())
	k8sclient := fakek8s.NewSimpleClientset()

	// Get Signer
	signer, secretpath, err := getSignerFromFile(t, ctx, k8sclient)
	if err != nil {
		t.Fatal(err)
	}
	ctx = setupContext(ctx, k8sclient, secretpath)

	unsignedPipeline := getUnsignedPipeline("unsigned")
	unsigned := &TrustedPipelineRun{v1beta1.PipelineRun{
		TypeMeta:   prTypeMeta,
		ObjectMeta: prObjectMeta,
		Spec: v1beta1.PipelineRunSpec{
			PipelineRef: &v1beta1.PipelineRef{
				Name: unsignedPipeline.Name,
			},
		},
	}}

	signedPipeline, err := getSignedPipeline(getUnsignedPipeline("signed"), signer)
	if err != nil {
		t.Fatal("fail to sign pipeline", err)
	}
	signed := unsigned.DeepCopy()
	signed.PipelineRun.Spec.PipelineRef.Name = signedPipeline.Name

	tamperedPipeline := signedPipeline.DeepCopy()
	tamperedPipeline.Name = "tampered"
	tampered := signed.DeepCopy()
	tampered.Spec.PipelineRef.Name = tamperedPipeline.Name

	tektonClient := faketekton.NewSimpleClientset(unsignedPipeline, signedPipeline, tamperedPipeline)

	tcs := []struct {
		name        string
		pipelineRun *TrustedPipelineRun
		wantErr     bool
	}{{
		name:        "Local PipelineRef Pass Verification",
		pipelineRun: signed,
		wantErr:     false,
	}, {
		name:        "Local PipelineRef Fail Verification with tampered content",
		pipelineRun: tampered,
		wantErr:     true,
	}, {
		name:        "Local PipelineRef Fail Verification without signature",
		pipelineRun: unsigned,
		wantErr:     true,
	},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.pipelineRun.verifyPipelineRun(ctx, k8sclient, tektonClient)
			if (err != nil) != tc.wantErr {
				t.Fatalf("verifyPipelineRun() get err %v, wantErr %t", err, tc.wantErr)
			}
		})
	}

}

func TestVerifyPipelineRun_TaskRef(t *testing.T) {
	ctx := logging.WithLogger(context.Background(), zaptest.NewLogger(t).Sugar())

	k8sclient := fakek8s.NewSimpleClientset()

	// Get Signer
	signer, secretpath, err := getSignerFromFile(t, ctx, k8sclient)
	if err != nil {
		t.Fatal(err)
	}
	ctx = setupContext(ctx, k8sclient, secretpath)

	// signed pipelineref with unsigned taskref
	pipelineWithUnsignedTask := getUnsignedPipeline("unsigned")
	unsignedTask := getUnsignedTask()
	pipelineWithUnsignedTask.Spec.Tasks = []v1beta1.PipelineTask{
		{TaskRef: &v1beta1.TaskRef{
			Name: unsignedTask.Name,
			Kind: "Task",
		}}}
	pipelineWithUnsignedTask, err = getSignedPipeline(pipelineWithUnsignedTask, signer)
	if err != nil {
		t.Fatal("fail to sign pipeline", err)
	}
	unsigned := &TrustedPipelineRun{
		v1beta1.PipelineRun{
			TypeMeta:   prTypeMeta,
			ObjectMeta: prObjectMeta,
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{
					Name: pipelineWithUnsignedTask.Name,
				},
			},
		},
	}

	// signed pipelineref with signed taskref
	pipelineWithSignedTask := getUnsignedPipeline("signed")
	signedTask, err := getSignedTask(unsignedTask, signer)
	if err != nil {
		t.Fatal("fail to sign task", err)
	}
	pipelineWithSignedTask.Spec.Tasks = []v1beta1.PipelineTask{{
		TaskRef: &v1beta1.TaskRef{
			Name: signedTask.Name,
			Kind: "Task",
		}}}
	pipelineWithSignedTask, err = getSignedPipeline(pipelineWithSignedTask, signer)
	if err != nil {
		t.Fatal("fail to sign pipeline", err)
	}
	signed := unsigned.DeepCopy()
	signed.Spec.PipelineRef = &v1beta1.PipelineRef{
		Name: pipelineWithSignedTask.Name,
	}

	// signed pipelineref with tampered taskref
	pipelineWithTamperedTask := getUnsignedPipeline("tampered")
	tamperedTask := signedTask.DeepCopy()
	tamperedTask.Name = "tampered"
	pipelineWithTamperedTask.Spec.Tasks = []v1beta1.PipelineTask{{TaskRef: &v1beta1.TaskRef{Name: tamperedTask.Name, Kind: "Task"}}}
	pipelineWithTamperedTask, err = getSignedPipeline(pipelineWithTamperedTask, signer)
	if err != nil {
		t.Fatal("fail to sign pipeline", err)
	}
	tampered := signed.DeepCopy()
	tampered.Spec.PipelineRef = &v1beta1.PipelineRef{
		Name: pipelineWithTamperedTask.Name,
	}

	tektonClient := faketekton.NewSimpleClientset(pipelineWithUnsignedTask, pipelineWithSignedTask, pipelineWithTamperedTask, unsignedTask, signedTask, tamperedTask)

	tcs := []struct {
		name        string
		pipelineRun *TrustedPipelineRun
		wantErr     bool
	}{{
		name:        "Local TaskRef Pass Verification",
		pipelineRun: signed,
		wantErr:     false,
	}, {
		name:        "Local TaskRef Fail Verification with tampered content",
		pipelineRun: tampered,
		wantErr:     true,
	}, {
		name:        "Local TaskRef Fail Verification without signature",
		pipelineRun: unsigned,
		wantErr:     true,
	},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.pipelineRun.verifyPipelineRun(ctx, k8sclient, tektonClient)
			if (err != nil) != tc.wantErr {
				t.Fatalf("verifyPipelineRun() get err %v, wantErr %t", err, tc.wantErr)
			}
		})
	}

}

func getUnsignedPipeline(name string) *v1beta1.Pipeline {
	return &v1beta1.Pipeline{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "tekton.dev/v1beta1",
			Kind:       "Pipeline"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: nameSpace,
		},
		Spec: v1beta1.PipelineSpec{
			Tasks: []v1beta1.PipelineTask{},
		},
	}
}

func getSignedPipeline(unsigned *v1beta1.Pipeline, signer signature.Signer) (*v1beta1.Pipeline, error) {
	signedPipeline := unsigned.DeepCopy()
	if signedPipeline.Annotations == nil {
		signedPipeline.Annotations = map[string]string{}
	}
	signature, err := SignInterface(signer, signedPipeline)
	if err != nil {
		return nil, err
	}
	signedPipeline.Annotations[SignatureAnnotation] = signature
	return signedPipeline, nil
}

func pushPipelineImage(t *testing.T, u *url.URL, pipeline *v1beta1.Pipeline) (typesv1.Hash, error) {
	t.Helper()
	ref, err := remotetest.CreateImage(u.Host+"/pipeline/"+pipeline.Name, pipeline)
	if err != nil {
		t.Fatalf("uploading image failed unexpectedly with an error: %v", err)
	}

	imgRef, err := imgname.ParseReference(ref)
	if err != nil {
		t.Fatalf("digest %s is not a valid reference: %v", ref, err)
	}

	img, err := remote.Image(imgRef, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	if err != nil {
		t.Fatalf("could not fetch created image: %v", err)
	}

	dig, err := img.Digest()
	if err != nil {
		t.Fatalf("failed to fetch img manifest: %v", err)
	}
	return dig, nil
}
