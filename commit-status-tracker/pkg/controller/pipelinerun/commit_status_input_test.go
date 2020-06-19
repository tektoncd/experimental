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
	"testing"

	tb "github.com/tektoncd/experimental/commit-status-tracker/test/builder"
)

func TestAnnotationByName(t *testing.T) {
	nt := []struct {
		name string
		opts []tb.PipelineRunOp
		want string
	}{
		{"no labels", nil, "default"},
		{"no matching label",
			[]tb.PipelineRunOp{tb.PipelineRunAnnotation("testing", "app")},
			"default"},
		{"with matching label",
			[]tb.PipelineRunOp{tb.PipelineRunAnnotation(statusContextName, "test-lint")},
			"test-lint"},
	}

	for _, tt := range nt {
		r := tb.PipelineRun("test-pipeline-run-with-labels", "foo", tt.opts...)
		if b := getAnnotationByName(r, statusContextName, "default"); b != tt.want {
			t.Errorf("Context() %s got %v, want %v", tt.name, b, tt.want)
		}
	}
}
