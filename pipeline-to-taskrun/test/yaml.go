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

// copied from https://github.com/tektoncd/pipeline/blob/main/test/yaml.go

package test

import (
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"testing"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/client/clientset/versioned/scheme"
	"k8s.io/apimachinery/pkg/runtime"
)

func mustParseYAML(t *testing.T, yaml string, i runtime.Object) {
	if _, _, err := scheme.Codecs.UniversalDeserializer().Decode([]byte(yaml), nil, i); err != nil {
		t.Fatalf("mustParseYAML (%s): %v", yaml, err)
	}
}

func MustParseTaskRun(t *testing.T, yaml string) *v1beta1.TaskRun {
	var tr v1beta1.TaskRun
	yaml = `apiVersion: tekton.dev/v1beta1
kind: TaskRun
` + yaml
	mustParseYAML(t, yaml, &tr)
	return &tr
}

func MustParseTask(t *testing.T, yaml string) *v1beta1.Task {
	var task v1beta1.Task
	yaml = `apiVersion: tekton.dev/v1beta1
kind: Task
` + yaml
	mustParseYAML(t, yaml, &task)
	return &task
}

func MustParsePipelineRun(t *testing.T, yaml string) *v1beta1.PipelineRun {
	var pr v1beta1.PipelineRun
	yaml = `apiVersion: tekton.dev/v1beta1
kind: PipelineRun
` + yaml
	mustParseYAML(t, yaml, &pr)
	return &pr
}

func MustParsePipeline(t *testing.T, yaml string) *v1beta1.Pipeline {
	var pipeline v1beta1.Pipeline
	yaml = `apiVersion: tekton.dev/v1beta1
kind: Pipeline
` + yaml
	mustParseYAML(t, yaml, &pipeline)
	return &pipeline
}

func MustParseRun(t *testing.T, yaml string) *v1alpha1.Run {
	var run v1alpha1.Run
	yaml = `apiVersion: tekton.dev/v1alpha1
kind: Run
` + yaml
	mustParseYAML(t, yaml, &run)
	return &run
}
