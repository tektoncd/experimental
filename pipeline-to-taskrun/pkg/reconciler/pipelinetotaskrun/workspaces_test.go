/*
Copyright 2021 The Tekton Authors

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

package pipelinetotaskrun

import (
	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/experimental/pipeline-to-taskrun/test"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/test/diff"
	"testing"
)

func TestGetNewWorkspaceMapping(t *testing.T) {
	p := test.MustParsePipeline(t, `
spec:
  tasks:
  - name: grab-source
    workspaces:
    - name: output
      workspace: where-it-all-happens
  - name: run-tests
    workspaces:
    - name: source
      workspace: where-it-all-happens
  - name: upload-results
    workspaces:
    - name: source
      workspace: where-it-all-happens
    - name: credentials
      workspace: gcs-creds
`)
	var pTasks []v1beta1.PipelineTask
	for _, ptask := range p.Spec.Tasks {
		pTasks = append(pTasks, ptask)
	}
	expectedMapping := PipelineTaskToWorkspaces{
		"grab-source": {
			"output": "where-it-all-happens",
		},
		"run-tests": {
			"source": "where-it-all-happens",
		},
		"upload-results": {
			"source":      "where-it-all-happens",
			"credentials": "gcs-creds",
		},
	}

	mapping := getNewWorkspaceMapping(pTasks)

	if d := cmp.Diff(expectedMapping, mapping); d != "" {
		t.Errorf("Did not get expected workspace mapping: %v", diff.PrintWantGot(d))
	}
}

func TestGetNewWorkspaceMappingNoWorkspaces(t *testing.T) {
	p := test.MustParsePipeline(t, `
spec:
  tasks:
  - name: grab-source
  - name: run-tests
  - name: upload-results
`)
	var pTasks []v1beta1.PipelineTask
	for _, ptask := range p.Spec.Tasks {
		pTasks = append(pTasks, ptask)
	}
	expectedMapping := PipelineTaskToWorkspaces{
		"grab-source":    {},
		"run-tests":      {},
		"upload-results": {},
	}

	mapping := getNewWorkspaceMapping(pTasks)

	if d := cmp.Diff(expectedMapping, mapping); d != "" {
		t.Errorf("Did not get expected workspace mapping: %v", diff.PrintWantGot(d))
	}
}
