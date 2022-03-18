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
	"crypto"
	"crypto/sha256"

	cosignsignature "github.com/sigstore/cosign/pkg/signature"
	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/sigstore/sigstore/pkg/signature/kms"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/system"
	"knative.dev/pkg/webhook/json"
)

func verifier(
	ctx context.Context,
	annotations map[string]string,
	k8sclient kubernetes.Interface,
) (signature.Verifier, error) {
	if annotations[kmsAnnotation] != "" {
		// Fetch key from kms.
		return kms.Get(ctx, annotations[kmsAnnotation], crypto.SHA256)
	} else {
		cosignPublicKeypath := secretPath
		// Overwrite the path if set in configmap.
		cm, err := k8sclient.CoreV1().ConfigMaps(system.Namespace()).Get(ctx, signingConfigMap, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		if cm.Data["signing-secret-path"] != "" {
			cosignPublicKeypath = cm.Data["signing-secret-path"]
		}
		return cosignsignature.LoadPublicKey(ctx, cosignPublicKeypath)
	}
}

func VerifyInterface(
	ctx context.Context,
	taskspec interface{},
	verifier signature.Verifier,
	signature []byte,
) (errs *apis.FieldError) {
	ts, err := json.Marshal(taskspec)
	if err != nil {
		return apis.ErrGeneric(err.Error(), "taskSpec")
	}

	h := sha256.New()
	h.Write(ts)

	if err := verifier.VerifySignature(bytes.NewReader(signature), bytes.NewReader(h.Sum(nil))); err != nil {
		return apis.ErrGeneric(err.Error(), "taskSpec")
	}

	return nil
}
