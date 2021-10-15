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
    - name: secret
      workspace: gcs-creds
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
			"secret": "gcs-creds",
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

func TestGetUnboundOptionalWorkspaces(t *testing.T) {
	mapping := PipelineTaskToWorkspaces{
		"grab-source": {
			"output": "where-it-all-happens",
		},
		"run-tests": {
			"source": "where-it-all-happens",
			"secret": "gcs-creds",
		},
		"upload-results": {
			"source":      "where-it-all-happens",
			"credentials": "gcs-creds",
		},
	}
	tasks := []*v1beta1.Task{
		test.MustParseTask(t, `
spec:
  workspaces:
  - name: output
  - name: ssh-directory
    optional: true
`),
		test.MustParseTask(t, `
spec:
  workspaces:
  - name: source
  - name: secret
    optional: true
`),
		test.MustParseTask(t, `
spec:
  workspaces:
  - name: source
  - name: credentials
`),
	}
	taskSpecs := map[string]*v1beta1.TaskSpec{
		"grab-source":    &tasks[0].Spec,
		"run-tests":      &tasks[1].Spec,
		"upload-results": &tasks[2].Spec,
	}
	expectedOptionalWS := []v1beta1.WorkspaceDeclaration{
		taskSpecs["grab-source"].Workspaces[1],
	}

	optionalWS, err := getUnboundOptionalWorkspaces(taskSpecs, mapping)
	if err != nil {
		t.Fatalf("Did not expect error when getting optional workspaces but got %v", err)
	}

	if d := cmp.Diff(expectedOptionalWS, optionalWS); d != "" {
		t.Errorf("Did not get expected optional workspaces: %v", diff.PrintWantGot(d))
	}
}

func TestGetUnboundOptionalWorkspacesInvalid(t *testing.T) {
	mapping := PipelineTaskToWorkspaces{
		"grab-source": {},
	}
	tasks := []*v1beta1.Task{
		test.MustParseTask(t, `
spec:
  workspaces:
  - name: output
`),
	}
	taskSpecs := map[string]*v1beta1.TaskSpec{
		"grab-source": &tasks[0].Spec,
	}

	_, err := getUnboundOptionalWorkspaces(taskSpecs, mapping)
	if err == nil {
		t.Fatalf("Expected error when required workspace is not bound but got none")
	}
}
