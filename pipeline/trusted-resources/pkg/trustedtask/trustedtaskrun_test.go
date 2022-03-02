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
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/authn/k8schain"
	imgname "github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	v1types "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/sigstore/cosign/pkg/cosign"
	cosignsignature "github.com/sigstore/cosign/pkg/signature"
	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	faketekton "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/fake"
	remotetest "github.com/tektoncd/pipeline/test"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakek8s "k8s.io/client-go/kubernetes/fake"
	"knative.dev/pkg/system"
	"knative.dev/pkg/webhook/json"
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

func TestVerifyResources_TaskSpec(t *testing.T) {
	ctx := context.Background()

	k8sclient := fakek8s.NewSimpleClientset(sa)
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
	signed.Annotations["tekton.dev/signature"] = signTaskSpec(t, signer, tr.Spec.TaskSpec)

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
			err := tc.taskRun.verifyTask(ctx, k8sclient, tektonClient)
			if (err != nil) != tc.wantErr {
				t.Errorf("verifyResources() get err %v, wantErr %t", err, tc.wantErr)
			}
		})
	}

}

func TestVerifyResources_OCIBundle(t *testing.T) {
	ctx := context.Background()
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
	dig, err := pushOCIImage(t, u, ts)
	if err != nil {
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
				Name:   "ts",
				Bundle: u.Host + "/task/" + ts.Name,
			},
		},
	}

	unsigned := &TrustedTaskRun{TaskRun: otr}

	signed := unsigned.DeepCopy()
	signed.Annotations["tekton.dev/signature"] = signOCIBundle(t, signer, dig)

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
			err := tc.taskRun.verifyTask(ctx, k8sclient, tektonClient)
			if (err != nil) != tc.wantErr {
				t.Errorf("verifyResources() get err %v, wantErr %t", err, tc.wantErr)
			}
		})
	}

}

func TestVerifyResources_TaskRef(t *testing.T) {
	ctx := context.Background()

	k8sclient := fakek8s.NewSimpleClientset(sa)
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
	ts, err := tektonClient.TektonV1beta1().Tasks(unsigned.Namespace).Get(ctx, unsigned.Spec.TaskRef.Name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Unexpected err %v", err)
	}
	signed.Annotations["tekton.dev/signature"] = signTaskSpec(t, signer, &ts.Spec)

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
			err := tc.taskRun.verifyTask(ctx, k8sclient, tektonClient)
			if (err != nil) != tc.wantErr {
				t.Errorf("verifyResources() get err %v, wantErr %t", err, tc.wantErr)
			}
		})
	}

}

func TestVerifyTaskSpec(t *testing.T) {
	ctx := context.Background()

	// get keys
	sv, err := getSignerVerifier(t, pass(password))
	if err != nil {
		t.Fatalf("Unexpected err %v", err)
	}

	tcs := []struct {
		name      string
		taskSpec  *v1beta1.TaskSpec
		signature string
		wantErr   bool
	}{
		{
			name:      "taskSpec Pass Verification",
			taskSpec:  taskSpecTest,
			signature: signTaskSpec(t, sv, taskSpecTest),
			wantErr:   false,
		},
		{
			name:      "taskSpec Fail Verification with empty signature",
			taskSpec:  taskSpecTest,
			signature: base64.StdEncoding.EncodeToString(nil),
			wantErr:   true,
		},
		{
			name:      "taskSpec Fail Verification with empty taskSpec",
			taskSpec:  nil,
			signature: signTaskSpec(t, sv, taskSpecTest),
			wantErr:   true,
		},
		{
			name:      "taskSpec Fail Verification with tampered taskSpec",
			taskSpec:  taskSpecTestTampered,
			signature: signTaskSpec(t, sv, taskSpecTest),
			wantErr:   true,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			signature, err := base64.StdEncoding.DecodeString(tc.signature)
			if err != nil {
				t.Fatal(err)
			}
			errs := verifyTaskSpec(ctx, tc.taskSpec, sv, signature)
			if (errs != nil) != tc.wantErr {
				t.Errorf("verifyTaskSpec() get err %v, wantErr %t", err, tc.wantErr)
			}
		})
	}

}

func TestVerifyTaskOCIBundle(t *testing.T) {
	ctx := context.Background()

	k8sclient := fakek8s.NewSimpleClientset(sa)

	// Create registry server
	s := httptest.NewServer(registry.New())
	defer s.Close()
	u, _ := url.Parse(s.URL)

	// Push OCI bundle
	dig, err := pushOCIImage(t, u, ts)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := pushOCIImage(t, u, tsTampered); err != nil {
		t.Fatal(err)
	}

	// Get signer
	sv, err := getSignerVerifier(t, pass(password))
	if err != nil {
		t.Fatalf("Unexpected err %v", err)
	}

	tcs := []struct {
		name      string
		bundle    string
		signature string
		wantErr   bool
	}{
		{
			name:      "OCIBundle Pass Verification",
			bundle:    u.Host + "/task/" + ts.Name,
			signature: signOCIBundle(t, sv, dig),
			wantErr:   false,
		},
		{
			name:      "OCIBundle Fail Verification with empty signature",
			bundle:    u.Host + "/task/" + ts.Name,
			signature: "",
			wantErr:   true,
		},
		{
			name:      "OCIBundle Fail Verification with empty Bundle",
			bundle:    "",
			signature: signOCIBundle(t, sv, dig),
			wantErr:   true,
		},
		{
			name:      "OCIBundle Fail Verification with tampered OCIBundle",
			bundle:    u.Host + "/task/" + tsTampered.Name,
			signature: signOCIBundle(t, sv, dig),
			wantErr:   true,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			signature, err := base64.StdEncoding.DecodeString(tc.signature)
			if err != nil {
				t.Fatal(err)
			}
			errs := verifyTaskOCIBundle(ctx, tc.bundle, sv, signature, k8sclient)
			if (errs != nil) != tc.wantErr {
				t.Errorf("verifyTaskOCIBundle() get err %v, wantErr %t", err, tc.wantErr)
			}
		})
	}

}

func TestDigest(t *testing.T) {
	ctx := context.Background()

	k8sclient := fakek8s.NewSimpleClientset(sa)
	// Create registry server
	s := httptest.NewServer(registry.New())
	defer s.Close()
	u, _ := url.Parse(s.URL)

	// Push OCI bundle
	dig, err := pushOCIImage(t, u, ts)
	t.Logf("Digest: %v", dig.String())
	if err != nil {
		t.Fatal(err)
	}

	kc, err := k8schain.New(ctx, k8sclient, k8schain.Options{
		Namespace:          nameSpace,
		ServiceAccountName: serviceAccount,
	})
	if err != nil {
		t.Fatal(err)
	}

	tcs := []struct {
		name     string
		imageRef string
		wantErr  bool
	}{
		{
			name:     "OCIBundle Pass Verification",
			imageRef: u.Host + "/task/" + ts.Name,
			wantErr:  false,
		},
		{
			name:     "OCIBundle Fail Verification with empty signature",
			imageRef: u.Host + "/task/" + tsTampered.Name,
			wantErr:  true,
		},
		{
			name:     "OCIBundle Fail Verification with empty Bundle",
			imageRef: "",
			wantErr:  true,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			_, err = digest(ctx, tc.imageRef, kc)
			if (err != nil) != tc.wantErr {
				t.Errorf("digest() get err %v, wantErr %t", err, tc.wantErr)
			}
		})
	}

}

// Generate key files to tmpdir, set configMap and return signer
func getSignerFromFile(t *testing.T, ctx context.Context, k8sclient *fakek8s.Clientset) (signature.Signer, error) {
	t.Helper()
	tmpDir := t.TempDir()
	privateKeyPath, _ := generateKeyFile(t, tmpDir, pass(password))
	signer, err := cosignsignature.SignerFromKeyRef(ctx, privateKeyPath, pass(password))
	if err != nil {
		t.Fatal(err)
	}
	signgingConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nameSpace,
			Name:      signgingConfigMap,
		},
		Data: map[string]string{"path": tmpDir + "/cosign.pub"},
	}

	_, err = k8sclient.CoreV1().ConfigMaps(system.Namespace()).Create(ctx, signgingConfigMap, metav1.CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}

	return signer, nil
}

func pushOCIImage(t *testing.T, u *url.URL, task *v1beta1.Task) (v1types.Hash, error) {
	t.Helper()
	ref, err := remotetest.CreateImage(u.Host+"/task/"+task.Name, task)
	if err != nil {
		t.Errorf("uploading image failed unexpectedly with an error: %v", err)
	}

	imgRef, err := imgname.ParseReference(ref)
	if err != nil {
		t.Errorf("digest %s is not a valid reference: %v", ref, err)
	}

	img, err := remote.Image(imgRef, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	if err != nil {
		t.Errorf("could not fetch created image: %v", err)
	}

	dig, err := img.Digest()
	if err != nil {
		t.Errorf("failed to fetch img manifest: %v", err)
	}
	return dig, nil
}

func getSignerVerifier(t *testing.T, pf cosign.PassFunc) (signature.SignerVerifier, error) {
	t.Helper()
	keys, err := cosign.GenerateKeyPair(pf)
	if err != nil {
		t.Fatal(err)
	}
	sv, err := cosign.LoadPrivateKey(keys.PrivateBytes, []byte(password))
	if err != nil {
		t.Fatal(err)
	}
	return sv, nil
}

func generateKeyFile(t *testing.T, tmpDir string, pf cosign.PassFunc) (privFile, pubFile string) {
	t.Helper()
	keys, err := cosign.GenerateKeyPair(pf)
	if err != nil {
		t.Fatalf("failed to generate keypair: %v", err)
	}

	tmpPrivFile := filepath.Join(tmpDir, "cosign.key")
	err = os.WriteFile(tmpPrivFile, keys.PrivateBytes, 0666)
	if err != nil {
		t.Fatal(err)
	}

	tmpPubFile := filepath.Join(tmpDir, "cosign.pub")
	err = os.WriteFile(tmpPubFile, keys.PublicBytes, 0666)
	if err != nil {
		t.Fatal(err)
	}

	return tmpPrivFile, tmpPubFile
}

func pass(s string) cosign.PassFunc {
	return func(_ bool) ([]byte, error) {
		return []byte(s), nil
	}
}

func signTaskSpec(t *testing.T, signer signature.Signer, taskspec *v1beta1.TaskSpec) string {
	t.Helper()
	tsSpec, err := json.Marshal(taskspec)
	if err != nil {
		t.Fatalf("Unexpected err %v", err)
	}

	h := sha256.New()
	h.Write(tsSpec)

	return signRawPayload(t, signer, h.Sum(nil))
}

func signOCIBundle(t *testing.T, signer signature.Signer, digest v1types.Hash) string {
	t.Helper()
	return signRawPayload(t, signer, []byte(digest.String()))
}

func signRawPayload(t *testing.T, signer signature.Signer, rawPayload []byte) string {
	t.Helper()
	signature, err := signer.SignMessage(bytes.NewReader(rawPayload))
	if err != nil {
		t.Fatalf("Unexpected err %v", err)
	}
	signatureEncoding := base64.StdEncoding.EncodeToString(signature)
	return signatureEncoding
}
