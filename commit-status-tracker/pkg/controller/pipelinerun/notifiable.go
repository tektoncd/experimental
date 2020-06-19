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
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

// isNotifiablePipelineRun returns true if this PipelineRun should report its
// completion status as a GitHub status.
func isNotifiablePipelineRun(pr *pipelinev1.PipelineRun) bool {
	for k, v := range pr.Annotations {
		if k == notifiableName && v == "true" {
			return true
		}
	}
	return false
}
