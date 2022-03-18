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
	"testing"

	faketekton "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/fake"
	"go.uber.org/zap/zaptest"
	fakek8s "k8s.io/client-go/kubernetes/fake"
	"knative.dev/pkg/logging"
)

func TestVerifyTask_TaskRun(t *testing.T) {
	ctx := logging.WithLogger(context.Background(), zaptest.NewLogger(t).Sugar())
	k8sclient := fakek8s.NewSimpleClientset()
	tektonClient := faketekton.NewSimpleClientset(ts, tsTampered)

	// Get Signer
	signer, err := getSignerFromFile(t, ctx, k8sclient)
	if err != nil {
		t.Fatal(err)
	}

	unsigned := &TrustedTask{Task: *ts}

	signed := unsigned.DeepCopy()

	if signed.Annotations == nil {
		signed.Annotations = map[string]string{}
	}

	signed.Annotations[SignatureAnnotation], err = SignInterface(signer, ts)
	if err != nil {
		t.Fatal(err)
	}

	tampered := signed.DeepCopy()
	tampered.Spec = tsTampered.Spec

	tcs := []struct {
		name    string
		task    *TrustedTask
		wantErr bool
	}{{
		name:    "API Task Pass Verification",
		task:    signed,
		wantErr: false,
	}, {
		name:    "API Task Fail Verification with tampered content",
		task:    tampered,
		wantErr: true,
	}, {
		name:    "API Task Fail Verification without signature",
		task:    unsigned,
		wantErr: true,
	},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			//cp := copyTrustedTask(tc.task)
			if err := tc.task.verifyTask(ctx, k8sclient, tektonClient); (err != nil) != tc.wantErr {
				t.Errorf("verifyResources() get err %v, wantErr %t", err, tc.wantErr)
			}
		})
	}

}
