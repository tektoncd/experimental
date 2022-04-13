package pipelineinpod

import (
	"fmt"

	cprv1alpha1 "github.com/tektoncd/experimental/pipeline-in-pod/pkg/apis/colocatedpipelinerun/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/reconciler/pipelinerun/resources"
	taskresources "github.com/tektoncd/pipeline/pkg/reconciler/taskrun/resources"
)

func ApplyParameters(p *v1beta1.PipelineSpec, cpr *cprv1alpha1.ColocatedPipelineRun) *v1beta1.PipelineSpec {
	// This assumes that the PipelineRun inputs have been validated against what the Pipeline requests.

	// stringReplacements is used for standard single-string stringReplacements, while arrayReplacements contains arrays
	// that need to be further processed.
	stringReplacements := map[string]string{}
	arrayReplacements := map[string][]string{}

	patterns := []string{
		"params.%s",
		"params[%q]",
		"params['%s']",
	}

	// Set all the default stringReplacements
	for _, p := range p.Params {
		if p.Default != nil {
			if p.Default.Type == v1beta1.ParamTypeString {
				for _, pattern := range patterns {
					stringReplacements[fmt.Sprintf(pattern, p.Name)] = p.Default.StringVal
				}
			} else {
				for _, pattern := range patterns {
					arrayReplacements[fmt.Sprintf(pattern, p.Name)] = p.Default.ArrayVal
				}
			}
		}
	}
	// Set and overwrite params with the ones from the PipelineRun
	for _, p := range cpr.Spec.Params {
		if p.Value.Type == v1beta1.ParamTypeString {
			for _, pattern := range patterns {
				stringReplacements[fmt.Sprintf(pattern, p.Name)] = p.Value.StringVal
			}
		} else {
			for _, pattern := range patterns {
				arrayReplacements[fmt.Sprintf(pattern, p.Name)] = p.Value.ArrayVal
			}
		}
	}

	return resources.ApplyReplacements(p, stringReplacements, arrayReplacements)
}

func ApplyParametersToTask(spec *v1beta1.TaskSpec, pt *v1beta1.PipelineTask) *v1beta1.TaskSpec {
	// This assumes that the TaskRun inputs have been validated against what the Task requests.

	// stringReplacements is used for standard single-string stringReplacements, while arrayReplacements contains arrays
	// that need to be further processed.
	var defaults []v1beta1.ParamSpec
	if len(spec.Params) > 0 {
		defaults = append(defaults, spec.Params...)
	}
	stringReplacements := map[string]string{}
	arrayReplacements := map[string][]string{}

	patterns := []string{
		"params.%s",
		"params[%q]",
		"params['%s']",
	}

	// Set all the default stringReplacements
	for _, p := range defaults {
		if p.Default != nil {
			if p.Default.Type == v1beta1.ParamTypeString {
				for _, pattern := range patterns {
					stringReplacements[fmt.Sprintf(pattern, p.Name)] = p.Default.StringVal
				}
			} else {
				for _, pattern := range patterns {
					arrayReplacements[fmt.Sprintf(pattern, p.Name)] = p.Default.ArrayVal
				}
			}
		}
	}
	// Set and overwrite params with the ones from the Pipeline Task
	for _, p := range pt.Params {
		if p.Value.Type == v1beta1.ParamTypeString {
			for _, pattern := range patterns {
				stringReplacements[fmt.Sprintf(pattern, p.Name)] = p.Value.StringVal
			}
		} else {
			for _, pattern := range patterns {
				arrayReplacements[fmt.Sprintf(pattern, p.Name)] = p.Value.ArrayVal
			}
		}
	}
	return taskresources.ApplyReplacements(spec, stringReplacements, arrayReplacements)
}
