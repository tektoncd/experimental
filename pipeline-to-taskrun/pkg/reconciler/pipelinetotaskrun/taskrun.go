package pipelinetotaskrun

import (
	"fmt"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

func getMergedTaskRun(run *v1alpha1.Run, pSpec *v1beta1.PipelineSpec, taskSpecs map[string]*v1beta1.TaskSpec) (*v1beta1.TaskRun, error) {
	sequence, err := putTasksInOrder(pSpec.Tasks)
	if err != nil {
		return nil, fmt.Errorf("couldn't find valid order for tasks: %v", err)
	}

	// we'll be declaring and mapping one workspace per provided workspace and eliminating the indirection added by the
	// workspaces declared by the Task. This will make sure that is volume claim templates are used, only one volume
	// will be created for each.
	newWorkspaceMapping := getNewWorkspaceMapping(sequence)

	// replace all param values with pipeline level params so we can ignore them from now on
	sequenceWithAppliedParams := applyPipelineLevelParams(sequence, run.Spec.Params)

	tr := &v1beta1.TaskRun{
		ObjectMeta: getObjectMeta(run),
		Spec: v1beta1.TaskRunSpec{
			ServiceAccountName: run.Spec.ServiceAccountName,
			TaskSpec:           &v1beta1.TaskSpec{},
			Workspaces:         run.Spec.Workspaces,
		},
	}

	for _, w := range pSpec.Workspaces {
		// mapping only the supported workspace declaration fields
		tr.Spec.TaskSpec.Workspaces = append(tr.Spec.TaskSpec.Workspaces, v1beta1.WorkspaceDeclaration{
			Name:        w.Name,
			Description: w.Description,
		})
	}

	// if an optional workspace isn't provided, we don't need to remap it but we still need to declare it
	// in order for any variable interpolation to work
	optionalWS, err := getUnboundOptionalWorkspaces(taskSpecs, newWorkspaceMapping)
	if err != nil {
		return nil, fmt.Errorf("invalid workspace binding for %s wasn't caught by validation: %v", run.Name, err)
	}
	for _, ws := range optionalWS {
		tr.Spec.TaskSpec.Workspaces = append(tr.Spec.TaskSpec.Workspaces, v1beta1.WorkspaceDeclaration{
			Name:        ws.Name,
			Description: ws.Description,
			Optional:    ws.Optional,
		})
	}

	for _, pTask := range sequenceWithAppliedParams {
		pti, err := NewPipelineTaskInfo(pTask, taskSpecs)
		if err != nil {
			return nil, fmt.Errorf("couldn't construct object to hold pipeline task info for %s: %v", pTask.Name, err)
		}

		pti = pti.NamespaceParams()
		pti = pti.NamespaceSteps()
		pti = pti.RenameWorkspaces(newWorkspaceMapping[pTask.Name])

		tr.Spec.Params = append(tr.Spec.Params, pti.ProvidedParamValues...)
		tr.Spec.TaskSpec.Params = append(tr.Spec.TaskSpec.Params, pti.TaskDeclaredParams...)
		tr.Spec.TaskSpec.Steps = append(tr.Spec.TaskSpec.Steps, pti.Steps...)
		// we don't support mapping results but we need to declare them in order for steps that write
		// results to be able to write to the dirs they expect
		tr.Spec.TaskSpec.Results = append(tr.Spec.TaskSpec.Results, pti.Results...)
	}

	return tr, nil
}
