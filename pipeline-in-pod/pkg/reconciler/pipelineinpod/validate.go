package pipelineinpod

import (
	"fmt"

	"github.com/tektoncd/experimental/pipeline-in-pod/pkg/apis/colocatedpipelinerun/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

// ValidateWorkspaceBindings validates that the Workspaces expected by a Pipeline are provided by a ColocatedPipelineRun.
func ValidateWorkspaceBindings(p *v1beta1.PipelineSpec, cpr *v1alpha1.ColocatedPipelineRun) error {
	pipelineRunWorkspaces := make(map[string]v1beta1.WorkspaceBinding)
	for _, binding := range cpr.Spec.Workspaces {
		pipelineRunWorkspaces[binding.Name] = binding
	}

	for _, ws := range p.Workspaces {
		if ws.Optional {
			continue
		}
		if _, ok := pipelineRunWorkspaces[ws.Name]; !ok {
			return fmt.Errorf("pipeline requires workspace with name %q be provided by colocatedpipelinerun", ws.Name)
		}
	}
	return nil
}
