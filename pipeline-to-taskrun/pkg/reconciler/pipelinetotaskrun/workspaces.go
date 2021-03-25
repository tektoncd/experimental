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
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

type PipelineTaskToWorkspaces map[string]map[string]string

// getNewWorkspaceMapping will create an object that maps from the mapped workspaces in each pipeline task in pTasks to
// the pipeline level workspace that is actually used. It returns a map where they keys are pipeline task names, and
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
