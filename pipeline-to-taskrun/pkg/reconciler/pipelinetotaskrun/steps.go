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
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/names"
	"github.com/tektoncd/pipeline/pkg/reconciler/pipelinerun/resources"
	resources2 "github.com/tektoncd/pipeline/pkg/reconciler/taskrun/resources"
	"github.com/tektoncd/pipeline/pkg/substitution"
)

// PipelineTaskInfo holds all of the info needed to run a pipeline task
type PipelineTaskInfo struct {
	// Name is the name of the pipeline Task
	Name string

	// TaskDeclaredParams are the parameters that the referenced Task spec declared
	TaskDeclaredParams []v1beta1.ParamSpec

	// ProvidedParamValues are the parameter values that were provided in the pipeline task
	ProvidedParamValues []v1beta1.Param

	// Steps are the steps the Task declared
	Steps []v1beta1.Step

	// Results are the results the Task declared
	Results []v1beta1.TaskResult
}

// NewPipelineTaskInfo will construct an object that will hold all the info needed to run the pipeline task
func NewPipelineTaskInfo(pTask v1beta1.PipelineTask, taskSpecs map[string]*v1beta1.TaskSpec) (PipelineTaskInfo, error) {
	taskSpec, ok := taskSpecs[pTask.Name]
	if !ok {
		return PipelineTaskInfo{}, fmt.Errorf("expected taskspec wasn't present in map for %q", pTask.Name)
	}
	return PipelineTaskInfo{
		Name:                pTask.Name,
		TaskDeclaredParams:  taskSpec.Params,
		ProvidedParamValues: pTask.Params,
		Steps:               taskSpec.Steps,
		Results:             taskSpec.Results,
	}, nil
}

func namespaceName(namespace, name string) string {
	return fmt.Sprintf("%s-%s", namespace, name)
}

func getStepName(ptaskName, stepName string) string {
	// leave unnamed steps unnamed
	if stepName != "" {
		// namespace the step ames to avoid collisions
		stepName = namespaceName(ptaskName, stepName)
		// make sure the newly generated name won't result in an invalid task spec
		stepName = names.SimpleNameGenerator.RestrictLength(stepName)
	}
	return stepName
}

// applyPipelineLevelParams will do variable replacement for all params in pTasks which are using Pipeline level
// params as their values.
func applyPipelineLevelParams(pTasks []v1beta1.PipelineTask, runSpecParams []v1beta1.Param) []v1beta1.PipelineTask {
	// we're taking advantage of the parameter variable replacement libs in Tekton Pipelines, which expect to apply
	// replacement onto entire Pipeline Specs from PipelineRuns
	tempPipelineSpec := &v1beta1.PipelineSpec{Tasks: pTasks}
	tempPipelineRun := &v1beta1.PipelineRun{Spec: v1beta1.PipelineRunSpec{Params: runSpecParams}}

	// replace the value with the pipeline param resolved value
	tempPipelineSpec = resources.ApplyParameters(tempPipelineSpec, tempPipelineRun)
	return tempPipelineSpec.Tasks
}

// NamespaceParams will return a new PipelineTaskInfo in which the names of all the declared params and
// provided values are updated such that the param name is prefaced by the name of the pipeline task. All uses of the
// params will be updated in the steps as well.
func (pti PipelineTaskInfo) NamespaceParams() PipelineTaskInfo {
	updatedPti := PipelineTaskInfo{
		Name:    pti.Name,
		Results: pti.Results,
	}

	// namespace the params by renaming them
	for _, p := range pti.TaskDeclaredParams {
		pName := namespaceName(pti.Name, p.Name)
		updatedPti.TaskDeclaredParams = append(updatedPti.TaskDeclaredParams, v1beta1.ParamSpec{
			Name:        pName,
			Type:        p.Type,
			Description: p.Description,
			Default:     p.Default,
		})
	}

	// get the values for each renamed param
	for _, p := range pti.ProvidedParamValues {
		pName := namespaceName(pti.Name, p.Name)
		updatedPti.ProvidedParamValues = append(updatedPti.ProvidedParamValues, v1beta1.Param{
			Name:  pName,
			Value: p.Value,
		})
	}

	// create a mapping of the replacements that can be used to update the steps
	replacements := map[string]string{}
	for _, p := range pti.TaskDeclaredParams {
		pName := namespaceName(pti.Name, p.Name)
		// this is the format that ApplyReplacements expects the replacements to arrive in; it infers the surrounding
		// dollar sign and brackets
		existing := fmt.Sprintf("params.%s", p.Name)
		// we'll replace the resulting wrapped existing param reference with the variable replacement syntax for
		// our renamed version
		renamed := fmt.Sprintf("$(params.%s)", pName)
		replacements[existing] = renamed
	}

	for i := range updatedPti.ProvidedParamValues {
		updatedPti.ProvidedParamValues[i].Value.StringVal = substitution.ApplyReplacements(updatedPti.ProvidedParamValues[i].Value.StringVal, replacements)
	}

	updatedTaskSpec := resources2.ApplyReplacements(&v1beta1.TaskSpec{Steps: pti.Steps}, replacements, nil)
	updatedPti.Steps = updatedTaskSpec.Steps
	return updatedPti
}

// NamespaceSteps will return a new PipelineTaskInfo in which the names of all steps are updated so that they are
// prefaced by the name of the pipeline task.
func (pti PipelineTaskInfo) NamespaceSteps() PipelineTaskInfo {
	updatedPti := PipelineTaskInfo{
		Name:                pti.Name,
		TaskDeclaredParams:  pti.TaskDeclaredParams,
		ProvidedParamValues: pti.ProvidedParamValues,
		Results:             pti.Results,
	}
	for _, step := range pti.Steps {
		updatedStep := step.DeepCopy()
		updatedStep.Name = getStepName(pti.Name, updatedStep.Name)
		updatedPti.Steps = append(updatedPti.Steps, *updatedStep)
	}
	return updatedPti
}

// RenameWorkspaces will return a new PipelineTask info in which all references to the keys in newMapping
// are updated to the values.
func (pti PipelineTaskInfo) RenameWorkspaces(newMapping map[string]string) PipelineTaskInfo {
	updatedPti := PipelineTaskInfo{
		Name:    pti.Name,
		Results: pti.Results,
	}

	// create a mapping of the replacements that can be used to update the steps
	replacements := map[string]string{}
	for oldName, newName := range newMapping {
		// this value will be blank for optional workspaces that aren't provided
		// for those we shouldn't do any replacement
		if newName == "" {
			continue
		}
		// we need to explicitly replace every known workspace variable
		for _, variable := range []string{"path", "bound", "claim", "volume"} {
			// this is the format that ApplyReplacements expects the replacements to arrive in; it infers the surrounding
			// dollar sign and brackets
			existing := fmt.Sprintf("workspaces.%s.%s", oldName, variable)
			// we'll replace the resulting wrapped existing param reference with the variable replacement syntax for
			// our renamed version
			renamed := fmt.Sprintf("$(workspaces.%s.%s)", newName, variable)
			replacements[existing] = renamed
		}
	}

	for _, p := range pti.ProvidedParamValues {
		updatedParam := v1beta1.Param{
			Name: p.Name,
			Value: v1beta1.ArrayOrString{
				Type: p.Value.Type,
				// not yet supporting array types
				StringVal: substitution.ApplyReplacements(p.Value.StringVal, replacements),
			},
		}
		updatedPti.ProvidedParamValues = append(updatedPti.ProvidedParamValues, updatedParam)
	}

	updatedTaskSpec := resources2.ApplyReplacements(
		&v1beta1.TaskSpec{
			Params: pti.TaskDeclaredParams,
			Steps:  pti.Steps,
		}, replacements, nil)
	updatedPti.TaskDeclaredParams = updatedTaskSpec.Params
	updatedPti.Steps = updatedTaskSpec.Steps

	return updatedPti
}
