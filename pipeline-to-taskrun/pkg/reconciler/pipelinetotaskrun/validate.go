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
	"fmt"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"knative.dev/pkg/apis"
)

func validateRun(run *v1alpha1.Run) (errs *apis.FieldError) {
	if run.Spec.Ref.Name == "" {
		errs = errs.Also(apis.ErrMissingField("name"))
	}
	for _, w := range run.Spec.Workspaces {
		if w.SubPath != "" {
			errs = errs.Also(apis.ErrDisallowedFields("spec.workspaces.subpath"))
		}
	}
	return errs
}

func validatePipelineTask(pTask *v1beta1.PipelineTask) error {
	if pTask.Timeout != nil {
		return fmt.Errorf("task level timeouts are not yet supported; declared a timeout %v", pTask.Timeout)
	}
	if pTask.Retries != 0 {
		return fmt.Errorf("task level retries are not yet supported; declared a %d retries", pTask.Retries)
	}
	if len(pTask.WhenExpressions) > 0 {
		return fmt.Errorf("when expressions are not supported")
	}
	if len(pTask.Conditions) > 0 {
		return fmt.Errorf("conditions are not supported")
	}
	if pTask.TaskRef != nil && pTask.TaskRef.Kind != "" && pTask.TaskRef.Kind != "Task" {
		return fmt.Errorf("custom tasks are not supported")
	}
	if pTask.TaskRef == nil {
		if err := validateEmbeddedTaskSpec(pTask.TaskSpec); err != nil {
			return fmt.Errorf("embedded task spec for %s is invalid: %v", pTask.Name, err)
		}
	}
	for _, w := range pTask.Workspaces {
		if w.SubPath != "" {
			return fmt.Errorf("subpaths for workspaces are not yet supported using subpath %s with workspace %s", w.SubPath, w.Name)
		}
	}
	return nil
}

func validateEmbeddedTaskSpec(embeddedSpec *v1beta1.EmbeddedTask) error {
	if len(embeddedSpec.Metadata.Labels) > 0 || len(embeddedSpec.Metadata.Annotations) > 0 {
		return fmt.Errorf("annotations and labels for embedded task specs are not yet supported")
	}
	return nil
}

func validateTaskSpec(taskSpec *v1beta1.TaskSpec) error {
	if taskSpec.StepTemplate != nil {
		return fmt.Errorf("step templates are not supported")
	}
	if taskSpec.Sidecars != nil {
		return fmt.Errorf("sidecars are not supported")
	}
	if taskSpec.Resources != nil {
		return fmt.Errorf("pipelineresources are not supported")
	}
	if len(taskSpec.Volumes) > 0 {
		return fmt.Errorf("volumes are not supported")
	}
	for _, step := range taskSpec.Steps {
		if len(step.VolumeMounts) > 0 {
			return fmt.Errorf("volume mounts are not supported")
		}
		if len(step.Workspaces) > 0 {
			return fmt.Errorf("isolated workspaces are not supported but %s is trying to use them", step.Name)
		}
	}
	for _, w := range taskSpec.Workspaces {
		if w.MountPath != "" {
			return fmt.Errorf("mountPaths are not supported but trying to mount %s to %s", w.Name, w.MountPath)
		}
		if w.ReadOnly {
			return fmt.Errorf("readOnly workspaces are not supported but %s is readOnly", w.Name)
		}
		if w.Optional {
			return fmt.Errorf("optional workspaces are not supported but %s is optional", w.Name)
		}
	}
	for _, p := range taskSpec.Params {
		if p.Type == v1beta1.ParamTypeArray {
			return fmt.Errorf("array params are not yet supported but %s is a param of type array", p.Name)
		}
	}
	return nil
}

func validatePipelineSpec(pSpec *v1beta1.PipelineSpec) error {
	if len(pSpec.Finally) > 0 {
		return fmt.Errorf("finally tasks are not supported")
	}

	if len(pSpec.Results) > 0 {
		return fmt.Errorf("mapping of pipeline level results not supported")
	}
	for _, pTask := range pSpec.Tasks {
		if err := validatePipelineTask(&pTask); err != nil {
			return fmt.Errorf("pipeline task %s is invalid: %v", pTask.Name, err)
		}
	}
	return nil
}

func validateTaskSpecs(taskSpecs map[string]*v1beta1.TaskSpec) error {
	for _, taskSpec := range taskSpecs {
		if err := validateTaskSpec(taskSpec); err != nil {
			return fmt.Errorf("task spec is invalid: %v", err)
		}
	}
	return nil
}
