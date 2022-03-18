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
	taskSpecTestTampered = &v1beta1.TaskSpec{
		Steps: []v1beta1.Step{{
			Container: corev1.Container{
				Image: "ubuntu",
				Name:  "hello",
			},
		}},
	}

	ts = &v1beta1.Task{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "tekton.dev/v1beta1",
			Kind:       "Task"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-task",
			Namespace: nameSpace,
		},
		Spec: *taskSpecTest,
	}
	tsTampered = &v1beta1.Task{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "tekton.dev/v1beta1",
			Kind:       "Task"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-task-tampered",
			Namespace: nameSpace,
		},
		Spec: *taskSpecTestTampered,
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
	tektonClient := faketekton.NewSimpleClientset(ts, tsTampered)

	// Get Signer
	signer, err := getSignerFromFile(t, ctx, k8sclient)
	if err != nil {
		t.Fatal(err)
	}

	tr := v1beta1.TaskRun{
		TypeMeta:   trTypeMeta,
		ObjectMeta: trObjectMeta,
		Spec: v1beta1.TaskRunSpec{
			TaskSpec: &ts.Spec,
		},
	}

	unsigned := &TrustedTaskRun{TaskRun: tr}

	signed := unsigned.DeepCopy()
	signed.Annotations[SignatureAnnotation], err = SignInterface(signer, tr)
	if err != nil {
		t.Fatal(err)
	}

	tampered := signed.DeepCopy()
	tampered.Spec.TaskSpec = &tsTampered.Spec

	tcs := []struct {
		name    string
		taskRun *TrustedTaskRun
		wantErr bool
	}{{
		name:    "API Task Pass Verification",
		taskRun: signed,
		wantErr: false,
	}, {
		name:    "API Task Fail Verification with tampered content",
		taskRun: tampered,
		wantErr: true,
	}, {
		name:    "API Task Fail Verification without signature",
		taskRun: unsigned,
		wantErr: true,
	},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.taskRun.verifyTaskRun(ctx, k8sclient, tektonClient); (err != nil) != tc.wantErr {
				t.Errorf("verifyResources() get err %v, wantErr %t", err, tc.wantErr)
			}
		})
	}

}

func TestVerifyTaskRun_OCIBundle(t *testing.T) {
	ctx := logging.WithLogger(context.Background(), zaptest.NewLogger(t).Sugar())

	k8sclient := fakek8s.NewSimpleClientset(sa)
	tektonClient := faketekton.NewSimpleClientset(ts, tsTampered)

	// Get Signer
	signer, err := getSignerFromFile(t, ctx, k8sclient)
	if err != nil {
		t.Fatal(err)
	}

	// Create registry server
	s := httptest.NewServer(registry.New())
	defer s.Close()
	u, _ := url.Parse(s.URL)

	// Push OCI bundle
	if _, err = pushOCIImage(t, u, ts); err != nil {
		t.Fatal(err)
	}

	if _, err := pushOCIImage(t, u, tsTampered); err != nil {
		t.Fatal(err)
	}

	// OCI taskruns
	otr := v1beta1.TaskRun{
		TypeMeta:   trTypeMeta,
		ObjectMeta: trObjectMeta,
		Spec: v1beta1.TaskRunSpec{
			TaskRef: &v1beta1.TaskRef{
				Name:   "test-task",
				Bundle: u.Host + "/task/" + ts.Name,
			},
		},
	}

	unsigned := &TrustedTaskRun{TaskRun: otr}

	signed := unsigned.DeepCopy()
	signed.Annotations[SignatureAnnotation], err = SignInterface(signer, otr)

	if err != nil {
		t.Fatal(err)
	}

	tampered := signed.DeepCopy()
	tampered.Spec.TaskRef.Bundle = u.Host + "/task/" + tsTampered.Name

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
			if err := tc.taskRun.verifyTaskRun(ctx, k8sclient, tektonClient); (err != nil) != tc.wantErr {
				t.Errorf("verifyResources() get err %v, wantErr %t", err, tc.wantErr)
			}
		})
	}

}

func TestVerifyTaskRun_TaskRef(t *testing.T) {
	ctx := logging.WithLogger(context.Background(), zaptest.NewLogger(t).Sugar())

	k8sclient := fakek8s.NewSimpleClientset()
	tektonClient := faketekton.NewSimpleClientset(ts, tsTampered)

	// Get Signer
	signer, err := getSignerFromFile(t, ctx, k8sclient)
	if err != nil {
		t.Fatal(err)
	}

	// Local taskref taskruns
	ltr := v1beta1.TaskRun{
		TypeMeta:   trTypeMeta,
		ObjectMeta: trObjectMeta,
		Spec: v1beta1.TaskRunSpec{
			TaskRef: &v1beta1.TaskRef{
				Name: "test-task",
			},
		},
	}

	unsigned := &TrustedTaskRun{TaskRun: ltr}

	signed := unsigned.DeepCopy()

	signed.Annotations[SignatureAnnotation], err = SignInterface(signer, ltr)

	if err != nil {
		t.Fatalf("Unexpected err %v", err)
	}

	tampered := signed.DeepCopy()
	tampered.Spec.TaskRef.Name = tsTampered.Name

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
		wantErr: true,
	}, {
		name:    "Local taskRef Fail Verification without signature",
		taskRun: unsigned,
		wantErr: true,
	},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.taskRun.verifyTaskRun(ctx, k8sclient, tektonClient)
			if (err != nil) != tc.wantErr {
				t.Fatalf("verifyResources() get err %v, wantErr %t", err, tc.wantErr)
			}
		})
	}

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
