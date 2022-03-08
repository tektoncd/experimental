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
	"encoding/base64"
	"io"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-containerregistry/pkg/authn/k8schain"
	"github.com/google/go-containerregistry/pkg/registry"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/tektoncd/pipeline/test/diff"
)

func TestSignInterface(t *testing.T) {
	ctx := context.Background()
	sv, err := GetSignerVerifier(password)
	if err != nil {
		t.Fatalf("failed to get signerverifier %v", err)
	}

	var mocksigner mockSigner

	tcs := []struct {
		name     string
		signer   signature.SignerVerifier
		target   interface{}
		expected string
		wantErr  bool
	}{{
		name:    "Sign TaskSpec",
		signer:  sv,
		target:  taskSpecTest,
		wantErr: false,
	}, {
		name:    "Sign String with cosign signer",
		signer:  sv,
		target:  "Hello world",
		wantErr: false,
	}, {
		name:    "Empty TaskSpec",
		signer:  sv,
		target:  nil,
		wantErr: false,
	}, {
		name:    "Empty Signer",
		signer:  nil,
		target:  taskSpecTest,
		wantErr: true,
	}, {
		name:     "Sign String with mock signer",
		signer:   mocksigner,
		target:   "Hello world",
		expected: "tY805zV53PtwDarK3VD6dQPx5MbIgctNcg/oSle+MG0=",
		wantErr:  false,
	},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			sig, err := SignInterface(tc.signer, tc.target)
			if (err != nil) != tc.wantErr {
				t.Fatalf("SignInterface() get err %v, wantErr %t", err, tc.wantErr)
			}

			if tc.expected != "" {
				if d := cmp.Diff(sig, tc.expected); d != "" {
					t.Fatalf("Diff:\n%s", diff.PrintWantGot(d))
				}
				return
			}

			if tc.wantErr {
				return
			}
			signature, err := base64.StdEncoding.DecodeString(sig)
			if err != nil {
				t.Fatalf("error decoding signature: %v", err)
			}
			if err := VerifyInterface(ctx, tc.target, tc.signer, signature); err != nil {
				t.Fatalf("SignInterface() generate wrong signature: %v", err)
			}

		})
	}
}

func TestSignRawPayload(t *testing.T) {
	sv, err := GetSignerVerifier(password)
	if err != nil {
		t.Fatalf("failed to get signerverifier %v", err)
	}

	var mocksigner mockSigner

	tcs := []struct {
		name     string
		signer   signature.SignerVerifier
		payload  []byte
		expected string
		wantErr  bool
	}{{
		name:    "Sign raw payload with cosign signer",
		signer:  sv,
		payload: []byte("payload"),
		wantErr: false,
	}, {
		name:    "Empty payload",
		signer:  sv,
		payload: nil,
		wantErr: false,
	}, {
		name:    "Empty Signer",
		signer:  nil,
		payload: []byte("payload"),
		wantErr: true,
	}, {
		name:     "Sign raw payload with mock signer",
		signer:   mocksigner,
		payload:  []byte("payload"),
		expected: "cGF5bG9hZA==",
		wantErr:  false,
	},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			sig, err := SignRawPayload(tc.signer, tc.payload)
			if (err != nil) != tc.wantErr {
				t.Fatalf("SignRawPayload() get err %v, wantErr %t", err, tc.wantErr)
			}

			if tc.expected != "" {
				if d := cmp.Diff(sig, tc.expected); d != "" {
					t.Fatalf("Diff:\n%s", diff.PrintWantGot(d))
				}
				return
			}

			if tc.wantErr {
				return
			}
			signature, err := base64.StdEncoding.DecodeString(sig)
			if err != nil {
				t.Fatal("failed to decode signature")
			}
			if err := sv.VerifySignature((bytes.NewReader(signature)), bytes.NewReader(tc.payload)); err != nil {
				t.Fatalf("SignRawPayload() get wrong signature %v:", err)
			}

		})
	}
}

func TestDigest(t *testing.T) {
	ctx := context.Background()

	// Create registry server
	s := httptest.NewServer(registry.New())
	defer s.Close()
	u, _ := url.Parse(s.URL)

	// Push OCI bundle
	if _, err := pushOCIImage(t, u, ts); err != nil {
		t.Fatal(err)
	}

	kc, err := k8schain.NewNoClient(ctx)
	if err != nil {
		t.Fatal(err)
	}

	tcs := []struct {
		name     string
		imageRef string
		wantErr  bool
	}{{
		name:     "OCIBundle Pass Verification",
		imageRef: u.Host + "/task/" + ts.Name,
		wantErr:  false,
	}, {
		name:     "OCIBundle Fail Verification with empty signature",
		imageRef: u.Host + "/task/" + tsTampered.Name,
		wantErr:  true,
	}, {
		name:     "OCIBundle Fail Verification with empty Bundle",
		imageRef: "",
		wantErr:  true,
	},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			if _, err = Digest(ctx, tc.imageRef, remote.WithAuthFromKeychain(kc)); (err != nil) != tc.wantErr {
				t.Fatalf("Digest() get err %v, wantErr %t", err, tc.wantErr)
			}
		})
	}
}

func TestGenerateKeyFile(t *testing.T) {
	tmpDir := t.TempDir()

	tcs := []struct {
		name     string
		dir      string
		password string
		wantErr  bool
	}{{
		name:     "Generate key files",
		dir:      tmpDir,
		password: password,
		wantErr:  false,
	}, {
		name:     "Empty directory",
		dir:      "",
		password: password,
		wantErr:  false,
	}, {
		name:     "Empty password",
		dir:      tmpDir,
		password: "",
		wantErr:  false,
	}}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, err := GenerateKeyFile(tmpDir, pass(password)); (err != nil) != tc.wantErr {
				t.Fatalf("GenerateKeyFile() get err %v, wantErr %t", err, tc.wantErr)
			}
		})
	}
}

type mockSigner struct {
	signature.SignerVerifier
}

func (mockSigner) SignMessage(message io.Reader, opts ...signature.SignOption) ([]byte, error) {
	return io.ReadAll(message)
}
