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

func TestVerifyInterface_Task(t *testing.T) {
	ctx := logging.WithLogger(context.Background(), zaptest.NewLogger(t).Sugar())

	// get signerverifer
	sv, err := GetSignerVerifier(password)
	if err != nil {
		t.Fatalf("Failed to get SignerVerifier %v", err)
	}

	unsignedTask := getUnsignedTask()

	signedTask, err := getSignedTask(unsignedTask, sv)
	if err != nil {
		t.Fatalf("Failed to get signed task %v", err)
	}

	tamperedTask := signedTask.DeepCopy()
	tamperedTask.Name = "tampered"

	tcs := []struct {
		name    string
		task    *v1beta1.Task
		wantErr bool
	}{{
		name:    "Signed Task Pass Verification",
		task:    signedTask,
		wantErr: false,
	}, {
		name:    "Unsigned Task Fail Verification",
		task:    unsignedTask,
		wantErr: true,
	}, {
		name:    "task Fail Verification with empty task",
		task:    nil,
		wantErr: true,
	}, {
		name:    "Tampered task Fail Verification",
		task:    tamperedTask,
		wantErr: true,
	},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			signature := []byte{}

			if tc.task != nil {
				if sig, ok := tc.task.Annotations[SignatureAnnotation]; ok {
					delete(tc.task.Annotations, SignatureAnnotation)
					signature, err = base64.StdEncoding.DecodeString(sig)
					if err != nil {
						t.Fatal(err)
					}
				}
			}

			errs := VerifyInterface(ctx, tc.task, sv, signature)
			if (errs != nil) != tc.wantErr {
				t.Fatalf("VerifyInterface() get err %v, wantErr %t", err, tc.wantErr)
			}
		})
	}

}
