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

package controller

import (
	"testing"

	"k8s.io/apimachinery/pkg/util/diff"
)

func TestName(t *testing.T) {
	for _, tc := range []struct {
		nameAnnotation string
		wantName       string
	}{
		{
			nameAnnotation: "{{ .Namespace }}/{{ .Name }}",
			wantName:       "default/echo-6b4fn-echo-xrxq4",
		},
		{
			nameAnnotation: `{{ index .Labels "tekton.dev/pipelineTask" }}`,
			wantName:       "echo",
		},
	} {
		t.Run(tc.nameAnnotation, func(t *testing.T) {
			tr := taskrun("testdata/taskrun.yaml")
			tr.Annotations[key("name")] = tc.nameAnnotation

			name, err := nameFor(tr)

			if tc.wantName != name {
				t.Errorf("-want,+got:\n%s", diff.StringDiff(tc.wantName, name))
			}

			if err != nil {
				t.Fatalf("Name: %v", err)
			}
		})
	}
}
