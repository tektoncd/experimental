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

package convert_test

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/tektoncd/experimental/workflows/pkg/apis/workflows/v1alpha1"
	"github.com/tektoncd/experimental/workflows/pkg/client/clientset/versioned/scheme"
	"github.com/tektoncd/experimental/workflows/pkg/convert"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestToPipelineRun(t *testing.T) {
	tests := []struct {
		name     string
		workflow *v1alpha1.Workflow
		want     *pipelinev1beta1.PipelineRun
	}{{
		name: "workflow with pipeline ref",
		workflow: MustParseWorkflow(t, "basic-workflow", "some-namespace", `
spec:
  pipeline:
    git:
      url: https://github.com/tektoncd/pipeline
      revision: main
      pathInRepo: tekton/release-pipeline.yaml
  serviceAccountName: pipelines-release
  timeout: 
    pipeline: 10s
  params:
  - name: imageRegistry
    default: tekton-releases-nightly
    type: string
  workspaces:
  - name: workarea
    emptyDir: {}
  - name: release-secret
    secret:
      secretName: release-secret
`),
		want: MustParsePipelineRun(t, `
metadata:
  generateName: basic-workflow-run-
  namespace: some-namespace
spec:
  pipelineRef:
    resolver: git
    params:
    - name: url
      value: https://github.com/tektoncd/pipeline
    - name: revision
      value: main
    - name: pathInRepo
      value: tekton/release-pipeline.yaml
  params:
  - name: imageRegistry
    value: tekton-releases-nightly
  workspaces:
  - name: workarea
    emptyDir: {}
  - name: release-secret
    secret:
      secretName: release-secret
  serviceAccountName: pipelines-release
  timeouts:
    pipeline: 10s
`),
	}, {
		name: "workflow with pipelineSpec",
		workflow: MustParseWorkflow(t, "basic-workflow", "some-namespace", `
spec:
  pipeline:
    spec:
      tasks:
      - name: task-with-no-params
        taskRef: 
          name: some-task
`),
		want: MustParsePipelineRun(t, `
metadata:
  generateName: basic-workflow-run-
  namespace: some-namespace
spec:
  serviceAccountName: default 
  pipelineSpec:
    tasks:
    - name: task-with-no-params
      taskRef: 
        name: some-task 
`),
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := convert.ToPipelineRun(tt.workflow)
			if err != nil {
				t.Errorf("ToPipelineRun() error = %v", err)
				return
			}
			if diff := cmp.Diff(tt.want, got, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("ToPipelineRun() failed. Diff (-want/+got): %s", diff)
			}
		})
	}
}

func mustParseYAML(t *testing.T, yaml string, i runtime.Object) {
	if _, _, err := scheme.Codecs.UniversalDeserializer().Decode([]byte(yaml), nil, i); err != nil {
		t.Fatalf("mustParseYAML (%s): %v", yaml, err)
	}
}

// MustParsePipelineRun takes YAML and parses it into a *v1beta1.PipelineRun
func MustParsePipelineRun(t *testing.T, yaml string) *pipelinev1beta1.PipelineRun {
	var pr pipelinev1beta1.PipelineRun
	yaml = `apiVersion: tekton.dev/v1beta1
kind: PipelineRun
` + yaml
	mustParseYAML(t, yaml, &pr)
	return &pr
}

// MustParsePipelineRun takes YAML and parses it into a *v1beta1.PipelineRun
func MustParseWorkflow(t *testing.T, name, namespace, yaml string) *v1alpha1.Workflow {
	var w v1alpha1.Workflow
	yaml = fmt.Sprintf(`apiVersion: tekton.dev/v1alpha1
kind: Workflow
metadata:
  name: %s
  namespace: %s
`+yaml, name, namespace)
	mustParseYAML(t, yaml, &w)
	return &w
}
