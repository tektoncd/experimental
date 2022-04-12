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

	"github.com/tektoncd/experimental/pipelines/trusted-resources/pkg/config"

	cosignsignature "github.com/sigstore/cosign/pkg/signature"
	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/sigstore/sigstore/pkg/signature/kms"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/webhook/json"
)

const (
	signingConfigMap = "config-trusted-resources"
	KMSAnnotation    = "tekton.dev/kms"
)

func verifier(ctx context.Context) (signature.Verifier, error) {
	// Check if kms is configured, if not check if cosign key configured.
	// TODO: Configuration of multiple keys will be discussed and can be changed soon.
	cfg := config.FromContextOrDefaults(ctx)
	if cfg.KMSKey != "" {
		// Fetch key from kms.
		return kms.Get(ctx, cfg.KMSKey, crypto.SHA256)
	} else {
		return cosignsignature.LoadPublicKey(ctx, cfg.CosignKey)
	}
}

// VerifyInterface get the checksum of json marshalled object and verify it.
func VerifyInterface(
	ctx context.Context,
	obj interface{},
	verifier signature.Verifier,
	signature []byte,
) (errs *apis.FieldError) {
	ts, err := json.Marshal(obj)
	if err != nil {
		return apis.ErrGeneric(err.Error(), "verify")
	}

	h := sha256.New()
	h.Write(ts)

	if err := verifier.VerifySignature(bytes.NewReader(signature), bytes.NewReader(h.Sum(nil))); err != nil {
		return apis.ErrGeneric(err.Error(), "verify")
	}

	return nil
}
