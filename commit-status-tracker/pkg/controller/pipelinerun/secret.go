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
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// TODO: what should these be called?
	secretName = "commit-status-tracker-git-secret"
	secretID   = "token"
)

func getAuthSecret(c client.Client, ns string) (string, error) {
	secret := &corev1.Secret{}
	err := c.Get(context.TODO(), getNamespaceSecretName(ns), secret)
	if err != nil {
		return "", fmt.Errorf("failed to getAuthSecret, error getting secret '%s' in namespace '%s': '%q'", secretName, ns, err)
	}

	tokenData, ok := secret.Data[secretID]
	if !ok {
		return "", fmt.Errorf("failed to getAuthSecret, secret %s does not have a 'token' key", ns)
	}
	return string(tokenData), nil
}

func getNamespaceSecretName(s string) types.NamespacedName {
	return types.NamespacedName{
		Namespace: s,
		Name:      secretName,
	}

}
