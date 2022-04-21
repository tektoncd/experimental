package pipelineinpod

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-multierror"
	cprv1alpha1 "github.com/tektoncd/experimental/pipeline-in-pod/pkg/apis/colocatedpipelinerun/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/logging"
)

func (r *Reconciler) getTaskSpec(ctx context.Context, cpr *cprv1alpha1.ColocatedPipelineRun, pipelineTask v1beta1.PipelineTask) (v1beta1.TaskSpec, error) {
	logger := logging.FromContext(ctx)
	var ts v1beta1.TaskSpec
	if cpr.Status.ChildStatuses != nil {
		for _, status := range cpr.Status.ChildStatuses {
			if status.PipelineTaskName != pipelineTask.Name {
				continue
			}
			return *status.Spec, nil
		}
	}
	// If task spec is not populated yet in child status, either get it from the pipeline task embedded spec,
	// or fetch it via the pipeline task taskref.
	if pipelineTask.TaskSpec != nil {
		return pipelineTask.TaskSpec.TaskSpec, nil
	}
	if pipelineTask.TaskRef == nil {
		return ts, fmt.Errorf("both task spec and task ref are nil")
	}
	task, err := r.pipelineClientSet.TektonV1beta1().Tasks(cpr.Namespace).Get(ctx, pipelineTask.TaskRef.Name, metav1.GetOptions{})
	if err != nil {
		return ts, err
	}
	logger.Infof("fetched task %s for pipeline task %s", task.Name, pipelineTask.Name)
	return task.Spec, nil
}

// Fetches tasks and writes them to cpr.Status.ChildStatus[].Spec along with pipeline task name.
// Substitutes parameters into the task specs.
// Initializes cpr.Status.ChildStatus[].StepStatuses with step names.
func (r *Reconciler) applyTasks(ctx context.Context, cpr *cprv1alpha1.ColocatedPipelineRun) error {
	//logger := logging.FromContext(ctx)
	if cpr.Status.PipelineSpec == nil {
		return nil
	}
	if len(cpr.Status.ChildStatuses) == len(cpr.Status.PipelineSpec.Tasks) {
		return nil
	}
	if len(cpr.Status.ChildStatuses) != 0 {
		// no support for matrix yet
		return fmt.Errorf("child statuses does not match pipeline spec: %d child statuses and %d pipeline tasks",
			len(cpr.Status.ChildStatuses), len(cpr.Status.PipelineSpec.Tasks))
	}
	var merr error
	for _, pt := range cpr.Status.PipelineSpec.Tasks {
		taskSpec, err := r.getTaskSpec(ctx, cpr, pt)
		if err != nil {
			merr = multierror.Append(merr, err)
		}
		taskSpec.SetDefaults(ctx)
		taskSpec = *ApplyParametersToTask(&taskSpec, &pt)
		var steps []v1beta1.StepState
		for _, step := range taskSpec.Steps {
			steps = append(steps, v1beta1.StepState{Name: step.Name})
		}
		cpr.Status.ChildStatuses = append(cpr.Status.ChildStatuses, cprv1alpha1.ChildStatus{
			PipelineTaskName: pt.Name,
			Spec:             &taskSpec, // no support for custom tasks yet
			StepStatuses:     steps,
		})
	}
	return merr
}
