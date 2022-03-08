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

package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/google/go-containerregistry/pkg/authn"
	imgname "github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	typesv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/tektoncd/experimental/pipelines/trusted-resources/pkg/trustedtask"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	remotetest "github.com/tektoncd/pipeline/test"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakek8s "k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/yaml"
)

const (
	password       = "hello"
	namespace      = "test"
	serviceAccount = "tekton-verify-task-webhook"
)

func init() {
	os.Setenv("SYSTEM_NAMESPACE", namespace)
	os.Setenv("WEBHOOK_SERVICEACCOUNT_NAME", serviceAccount)
}

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

	ts = &v1beta1.Task{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "tekton.dev/v1beta1",
			Kind:       "Task"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-task",
			Namespace: namespace,
		},
		Spec: *taskSpecTest,
	}

	trTypeMeta = metav1.TypeMeta{
		Kind:       pipeline.TaskRunControllerName,
		APIVersion: "tekton.dev/v1beta1"}

	trObjectMeta = metav1.ObjectMeta{
		Name:        "tr",
		Namespace:   namespace,
		Annotations: map[string]string{},
	}

	sa = &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      serviceAccount,
		},
	}
)

func TestSign_Taskrun(t *testing.T) {
	ctx := context.Background()

	sv, err := trustedtask.GetSignerVerifier(password)
	if err != nil {
		t.Fatalf("error get signerverifier: %v", err)
	}

	tr := &v1beta1.TaskRun{
		TypeMeta:   trTypeMeta,
		ObjectMeta: trObjectMeta,
		Spec: v1beta1.TaskRunSpec{
			TaskSpec: &ts.Spec,
		},
	}

	var writer bytes.Buffer

	if err := Sign(ctx, tr, sv, &writer); err != nil {
		t.Fatalf("Sign() get err %v", err)
	}

	signed := writer.Bytes()
	tr, signature := unmarshal(t, signed)

	if err := trustedtask.VerifyInterface(ctx, tr.Spec.TaskSpec, sv, signature); err != nil {
		t.Fatalf("VerifyTaskOCIBundle get error: %v", err)
	}

}

func TestSign_OCIBundle(t *testing.T) {
	ctx := context.Background()

	sv, err := trustedtask.GetSignerVerifier(password)
	if err != nil {
		t.Fatalf("error get signerverifier: %v", err)
	}

	k8sclient := fakek8s.NewSimpleClientset(sa)

	// Create registry server
	s := httptest.NewServer(registry.New())
	defer s.Close()
	u, err := url.Parse(s.URL)
	if err != nil {
		t.Fatalf("error parsing url: %v", err)
	}

	// Push OCI bundle
	pushOCIImage(t, u, ts)

	tr := &v1beta1.TaskRun{
		TypeMeta:   trTypeMeta,
		ObjectMeta: trObjectMeta,
		Spec: v1beta1.TaskRunSpec{
			TaskRef: &v1beta1.TaskRef{
				Name:   "ts",
				Bundle: u.Host + "/task/" + ts.Name,
			},
		},
	}

	var writer bytes.Buffer
	if err := Sign(ctx, tr, sv, &writer); err != nil {
		t.Fatalf("Sign() get err %v", err)
	}

	signed := writer.Bytes()
	tr, signature := unmarshal(t, signed)

	if err := trustedtask.VerifyTaskOCIBundle(ctx, tr.Spec.TaskRef.Bundle, sv, signature, k8sclient); err != nil {
		t.Fatalf("VerifyTaskOCIBundle get error: %v", err)
	}

}

func unmarshal(t *testing.T, buf []byte) (*v1beta1.TaskRun, []byte) {
	tr := &v1beta1.TaskRun{}
	if err := yaml.Unmarshal(buf, &tr); err != nil {
		t.Fatalf("error unmarshalling buffer: %v", err)
	}

	signature, err := base64.StdEncoding.DecodeString(tr.Annotations[trustedtask.SignatureAnnotation])
	if err != nil {
		t.Fatalf("error decoding signature: %v", err)
	}
	return tr, signature
}

func pushOCIImage(t *testing.T, u *url.URL, task *v1beta1.Task) typesv1.Hash {
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
	return dig
}
