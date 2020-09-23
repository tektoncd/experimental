// Copyright 2020 The Tekton Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pipelinerun

import (
	"os"
	"regexp"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

func TestGetAuthSecretWithExistingToken(t *testing.T) {
	logf.SetLogger(logf.ZapLogger(true))

	secret := makeSecret(defaultSecretName,
		map[string][]byte{"token": []byte(testToken)})
	objs := []runtime.Object{
		secret,
	}

	cl := fake.NewFakeClient(objs...)
	sec, err := getAuthSecret(cl, testNamespace)
	if err != nil {
		t.Fatal(err)
	}
	if sec != testToken {
		t.Fatalf("got %s, want %s", sec, testToken)
	}
}

func TestGetAuthSecretWithNoSecret(t *testing.T) {
	logf.SetLogger(logf.ZapLogger(true))
	objs := []runtime.Object{}

	cl := fake.NewFakeClient(objs...)
	_, err := getAuthSecret(cl, testNamespace)

	wantErr := "error getting secret 'commit-status-tracker-git-secret' in namespace 'test-namespace':.* not found"
	if !matchError(t, wantErr, err) {
		t.Fatalf("failed to match error when no secret: got %s, want %s", err, wantErr)
	}
}

func TestGetAuthSecretWithNoToken(t *testing.T) {
	logf.SetLogger(logf.ZapLogger(true))
	secret := makeSecret(
		defaultSecretName,
		map[string][]byte{})
	objs := []runtime.Object{
		secret,
	}

	cl := fake.NewFakeClient(objs...)
	_, err := getAuthSecret(cl, testNamespace)

	wantErr := "secret .* does not have a 'token' key"
	if !matchError(t, wantErr, err) {
		t.Fatalf("failed to match error when no secret: got %s, want %s", err, wantErr)
	}
}

func TestGetAuthSecretWithNameInEnvironment(t *testing.T) {
	customSecretName := "testing-secret-name"
	old := os.Getenv(secretNameEnvVar)
	defer func() {
		os.Setenv(secretNameEnvVar, old)
	}()
	os.Setenv(secretNameEnvVar, customSecretName)
	logf.SetLogger(logf.ZapLogger(true))

	secret := makeSecret(customSecretName,
		map[string][]byte{"token": []byte(testToken)})
	objs := []runtime.Object{
		secret,
	}

	cl := fake.NewFakeClient(objs...)
	sec, err := getAuthSecret(cl, testNamespace)
	if err != nil {
		t.Fatal(err)
	}
	if sec != testToken {
		t.Fatalf("got %s, want %s", sec, testToken)
	}
}

func makeSecret(secretName string, data map[string][]byte) *corev1.Secret {
	return &corev1.Secret{
		Type: corev1.SecretTypeOpaque,
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: testNamespace,
		},
		Data: data,
	}
}

func matchError(t *testing.T, s string, e error) bool {
	t.Helper()
	if s == "" && e == nil {
		return true
	}
	if s != "" && e == nil {
		return false
	}
	match, err := regexp.MatchString(s, e.Error())
	if err != nil {
		t.Fatal(err)
	}
	return match
}
