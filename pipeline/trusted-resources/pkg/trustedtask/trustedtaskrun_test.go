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
	"path/filepath"
	"testing"

	"github.com/google/go-containerregistry/pkg/authn"
	imgname "github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	typesv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	cosignsignature "github.com/sigstore/cosign/pkg/signature"
	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/tektoncd/pipeline/pkg/apis/config"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	faketekton "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/fake"
	remotetest "github.com/tektoncd/pipeline/test"
	"go.uber.org/zap/zaptest"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	fakek8s "k8s.io/client-go/kubernetes/fake"
	"knative.dev/pkg/logging"
	logtesting "knative.dev/pkg/logging/testing"
	"knative.dev/pkg/system"
)

const (
	nameSpace      = "trusted-task"
	serviceAccount = "tekton-verify-task-webhook"
	password       = "hello"
)

var (
	// tasks for testing
	taskSpecTest = &v1beta1.TaskSpec{
		Steps: []v1beta1.Step{{
			Container: corev1.Container{
				Image: "ubuntu",
				Name:  "echo",
			},
		}},
	}

	trTypeMeta = metav1.TypeMeta{
		Kind:       pipeline.TaskRunControllerName,
		APIVersion: "tekton.dev/v1beta1"}

	trObjectMeta = metav1.ObjectMeta{
		Name:        "tr",
		Namespace:   nameSpace,
		Annotations: map[string]string{},
	}

	// service account
	sa = &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nameSpace,
			Name:      serviceAccount,
		},
	}
)

func init() {
	os.Setenv("SYSTEM_NAMESPACE", nameSpace)
	os.Setenv("WEBHOOK_SERVICEACCOUNT_NAME", serviceAccount)
}

func TestVerifyTaskRun_TaskRun(t *testing.T) {
	ctx := logging.WithLogger(context.Background(), zaptest.NewLogger(t).Sugar())
	k8sclient := fakek8s.NewSimpleClientset()
	tektonClient := faketekton.NewSimpleClientset()

	unsignedTask := getUnsignedTask()

	tr := v1beta1.TaskRun{
		TypeMeta:   trTypeMeta,
		ObjectMeta: trObjectMeta,
		Spec: v1beta1.TaskRunSpec{
			TaskSpec: &unsignedTask.Spec,
		},
	}

	unsigned := &TrustedTaskRun{TaskRun: tr}

	if err := unsigned.verifyTaskRun(ctx, k8sclient, tektonClient); err != nil {
		t.Errorf("verifyTaskRun() get err %v", err)
	}

}

func TestVerifyTaskRun_OCIBundle(t *testing.T) {
	ctx := logging.WithLogger(context.Background(), zaptest.NewLogger(t).Sugar())

	// this is needed to enable oci bundle
	cfg := config.NewStore(logtesting.TestLogger(t))
	cfg.OnConfigChanged(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: config.GetFeatureFlagsConfigName()},
		Data: map[string]string{
			"enable-tekton-oci-bundles": "true",
		},
	})
	ctx = cfg.ToContext(ctx)

	k8sclient := fakek8s.NewSimpleClientset(sa)
	tektonClient := faketekton.NewSimpleClientset()

	// Get Signer
	signer, err := getSignerFromFile(t, ctx, k8sclient)
	if err != nil {
		t.Fatal(err)
	}

	unsignedTask := getUnsignedTask()

	signedTask, err := getSignedTask(unsignedTask, signer)
	if err != nil {
		t.Fatal("fail to sign task", err)
	}

	tamperedTask := signedTask.DeepCopy()
	tamperedTask.Name = "tampered"

	// Create registry server
	s := httptest.NewServer(registry.New())
	defer s.Close()
	u, _ := url.Parse(s.URL)

	// Push OCI bundle
	if _, err = pushOCIImage(t, u, unsignedTask); err != nil {
		t.Fatal(err)
	}

	if _, err = pushOCIImage(t, u, signedTask); err != nil {
		t.Fatal(err)
	}

	if _, err := pushOCIImage(t, u, tamperedTask); err != nil {
		t.Fatal(err)
	}

	// OCI taskruns
	otr := v1beta1.TaskRun{
		TypeMeta:   trTypeMeta,
		ObjectMeta: trObjectMeta,
		Spec: v1beta1.TaskRunSpec{
			TaskRef: &v1beta1.TaskRef{
				Name:   unsignedTask.Name,
				Bundle: u.Host + "/task/" + unsignedTask.Name,
			},
		},
	}

	unsigned := &TrustedTaskRun{TaskRun: otr}

	signed := unsigned.DeepCopy()
	signed.Spec.TaskRef.Name = signedTask.Name
	signed.Spec.TaskRef.Bundle = u.Host + "/task/" + signedTask.Name

	tampered := unsigned.DeepCopy()
	tampered.Spec.TaskRef.Name = tamperedTask.Name
	tampered.Spec.TaskRef.Bundle = u.Host + "/task/" + tamperedTask.Name

	tcs := []struct {
		name    string
		taskRun *TrustedTaskRun
		wantErr bool
	}{{
		name:    "OCI Bundle Pass Verification",
		taskRun: signed,
		wantErr: false,
	}, {
		name:    "OCI Bundle Fail Verification without tampered content",
		taskRun: tampered,
		wantErr: true,
	}, {
		name:    "OCI Bundle Fail Verification without signature",
		taskRun: unsigned,
		wantErr: true,
	},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			//cp := copyTrustedTaskRun(tc.taskRun)
			if err := tc.taskRun.verifyTaskRun(ctx, k8sclient, tektonClient); (err != nil) != tc.wantErr {
				t.Errorf("verifyTaskRun() get err: %v, wantErr: %t", err, tc.wantErr)
			}
		})
	}

}

func TestVerifyTaskRun_TaskRef(t *testing.T) {
	ctx := logging.WithLogger(context.Background(), zaptest.NewLogger(t).Sugar())

	k8sclient := fakek8s.NewSimpleClientset()

	// Get Signer
	signer, err := getSignerFromFile(t, ctx, k8sclient)
	if err != nil {
		t.Fatal(err)
	}

	unsignedTask := getUnsignedTask()

	// Local taskref taskruns
	ltr := v1beta1.TaskRun{
		TypeMeta:   trTypeMeta,
		ObjectMeta: trObjectMeta,
		Spec: v1beta1.TaskRunSpec{
			TaskRef: &v1beta1.TaskRef{
				Name: unsignedTask.Name,
			},
		},
	}

	unsigned := &TrustedTaskRun{TaskRun: ltr}

	signedTask, err := getSignedTask(unsignedTask, signer)
	if err != nil {
		t.Fatal("fail to sign task", err)
	}

	signed := unsigned.DeepCopy()
	signed.TaskRun.Spec.TaskRef.Name = signedTask.Name

	tamperedTask := signedTask.DeepCopy()
	tamperedTask.Name = "tampered"

	tampered := signed.DeepCopy()
	tampered.Spec.TaskRef.Name = tamperedTask.Name

	tektonClient := faketekton.NewSimpleClientset(unsignedTask, signedTask, tamperedTask)

	tcs := []struct {
		name    string
		taskRun *TrustedTaskRun
		wantErr bool
	}{{
		name:    "Local taskRef Pass Verification",
		taskRun: signed,
		wantErr: false,
	}, {
		name:    "Local taskRef Fail Verification with tampered content",
		taskRun: tampered,
		wantErr: false,
	}, {
		name:    "Local taskRef Fail Verification without signature",
		taskRun: unsigned,
		wantErr: false,
	},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.taskRun.verifyTaskRun(ctx, k8sclient, tektonClient)
			if (err != nil) != tc.wantErr {
				t.Fatalf("verifyTaskRun() get err %v, wantErr %t", err, tc.wantErr)
			}
		})
	}

}

func getUnsignedTask() *v1beta1.Task {
	return &v1beta1.Task{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "tekton.dev/v1beta1",
			Kind:       "Task"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-task",
			Namespace: nameSpace,
		},
		Spec: *taskSpecTest,
	}
}

func getSignedTask(unsigned *v1beta1.Task, signer signature.Signer) (*v1beta1.Task, error) {
	signedTask := unsigned.DeepCopy()
	signedTask.Name = "signed"
	if signedTask.Annotations == nil {
		signedTask.Annotations = map[string]string{}
	}
	signature, err := SignInterface(signer, signedTask)
	if err != nil {
		return nil, err
	}
	signedTask.Annotations[SignatureAnnotation] = signature
	return signedTask, nil
}

// Generate key files to tmpdir, set configMap and return signer
func getSignerFromFile(t *testing.T, ctx context.Context, k8sclient kubernetes.Interface) (signature.Signer, error) {
	t.Helper()
	tmpDir := t.TempDir()
	privateKeyPath, _, err := GenerateKeyFile(tmpDir, pass(password))
	if err != nil {
		t.Fatal(err)
	}
	signer, err := cosignsignature.SignerFromKeyRef(ctx, privateKeyPath, pass(password))
	if err != nil {
		t.Fatal(err)
	}
	cfg := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nameSpace,
			Name:      signingConfigMap,
		},
		Data: map[string]string{"signing-secret-path": filepath.Join(tmpDir, "cosign.pub")},
	}
	if _, err := k8sclient.CoreV1().ConfigMaps(system.Namespace()).Create(ctx, cfg, metav1.CreateOptions{}); err != nil && !errors.IsAlreadyExists(err) {
		t.Fatal(err)
	}

	return signer, nil
}

func pushOCIImage(t *testing.T, u *url.URL, task *v1beta1.Task) (typesv1.Hash, error) {
	t.Helper()
	ref, err := remotetest.CreateImage(u.Host+"/task/"+task.Name, task)
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
