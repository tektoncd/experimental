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
)

type PipelineTaskToWorkspaces map[string]map[string]string

// getNewWorkspaceMapping will create an object that maps from the mapped workspaces in each pipeline task in pTasks to
// the pipeline level workspace that is actually used. It returns a map where the keys are pipeline task names, and
// the values are dictionaries that map each of the task's declared workspaces to the actual workspaces used.
func getNewWorkspaceMapping(pTasks []v1beta1.PipelineTask) PipelineTaskToWorkspaces {
	mapping := PipelineTaskToWorkspaces{}

	for _, pTask := range pTasks {
		mapping[pTask.Name] = map[string]string{}
		for _, wsBinding := range pTask.Workspaces {
			mapping[pTask.Name][wsBinding.Name] = wsBinding.Workspace
		}
	}

	return mapping
}

// getUnboundOptionalWorkspaces returns a list of all the optional workspaces that are declared in taskSpecs but not actually
// bound in newWorkspaceMapping, or an error if an unbound workspace is not optional.
func getUnboundOptionalWorkspaces(taskSpecs map[string]*v1beta1.TaskSpec, newWorkspaceMapping PipelineTaskToWorkspaces) ([]v1beta1.WorkspaceDeclaration, error) {
	optionalWS := []v1beta1.WorkspaceDeclaration{}
	for pt, mappings := range newWorkspaceMapping {
		taskSpec := taskSpecs[pt]
		for _, ws := range taskSpec.Workspaces {
			if _, ok := mappings[ws.Name]; !ok {
				if !ws.Optional {
					return nil, fmt.Errorf("workspace %s is not bound in %s but is not optional", ws.Name, pt)
				}
				optionalWS = append(optionalWS, ws)
			}
		}
	}
	return optionalWS, nil
}
