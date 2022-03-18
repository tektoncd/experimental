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
	"encoding/base64"
	"testing"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"go.uber.org/zap/zaptest"
	"knative.dev/pkg/logging"
)

func TestVerifyInterface(t *testing.T) {
	ctx := logging.WithLogger(context.Background(), zaptest.NewLogger(t).Sugar())

	// get keys
	sv, err := GetSignerVerifier(password)
	if err != nil {
		t.Fatalf("Unexpected err %v", err)
	}

	tcs := []struct {
		name         string
		taskSpec     *v1beta1.TaskSpec
		hasSignature bool
		wantErr      bool
	}{{
		name:         "taskSpec Pass Verification",
		taskSpec:     taskSpecTest,
		hasSignature: true,
		wantErr:      false,
	}, {
		name:         "taskSpec Fail Verification with empty signature",
		taskSpec:     taskSpecTest,
		hasSignature: false,
		wantErr:      true,
	}, {
		name:         "taskSpec Fail Verification with empty taskSpec",
		taskSpec:     nil,
		hasSignature: true,
		wantErr:      true,
	}, {
		name:         "taskSpec Fail Verification with tampered taskSpec",
		taskSpec:     taskSpecTestTampered,
		hasSignature: true,
		wantErr:      true,
	},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			sig := ""
			if tc.hasSignature {
				sig, err = SignInterface(sv, taskSpecTest)
				if err != nil {
					t.Fatal(err)
				}
			}

			signature, err := base64.StdEncoding.DecodeString(sig)
			if err != nil {
				t.Fatal(err)
			}
			errs := VerifyInterface(ctx, tc.taskSpec, sv, signature)
			if (errs != nil) != tc.wantErr {
				t.Fatalf("VerifyInterface() get err %v, wantErr %t", err, tc.wantErr)
			}
		})
	}

}
