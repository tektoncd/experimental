/*
Copyright 2020 The Tekton Authors

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

package tasklooprun

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/tektoncd/experimental/task-loops/pkg/apis/taskloop"
	taskloopv1alpha1 "github.com/tektoncd/experimental/task-loops/pkg/apis/taskloop/v1alpha1"
	taskloopclientset "github.com/tektoncd/experimental/task-loops/pkg/client/clientset/versioned"
	listerstaskloop "github.com/tektoncd/experimental/task-loops/pkg/client/listers/taskloop/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	clientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	runreconciler "github.com/tektoncd/pipeline/pkg/client/injection/reconciler/pipeline/v1alpha1/run"
	listersalpha "github.com/tektoncd/pipeline/pkg/client/listers/pipeline/v1alpha1"
	listers "github.com/tektoncd/pipeline/pkg/client/listers/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/names"
	"github.com/tektoncd/pipeline/pkg/reconciler/events"
	"go.uber.org/zap"
	"gomodules.xyz/jsonpatch/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/logging"
	pkgreconciler "knative.dev/pkg/reconciler"
)

const (
	// taskLoopLabelKey is the label identifier for a TaskLoop.  This label is added to the Run and its TaskRuns.
	taskLoopLabelKey = "/taskLoop"

	// taskLoopRunLabelKey is the label identifier for a Run.  This label is added to the Run's TaskRuns.
	taskLoopRunLabelKey = "/run"

	// taskLoopIterationLabelKey is the label identifier for the iteration number.  This label is added to the Run's TaskRuns.
	taskLoopIterationLabelKey = "/taskLoopIteration"
)

// Reconciler implements controller.Reconciler for Configuration resources.
type Reconciler struct {
	pipelineClientSet clientset.Interface
	taskloopClientSet taskloopclientset.Interface
	runLister         listersalpha.RunLister
	taskLoopLister    listerstaskloop.TaskLoopLister
	taskRunLister     listers.TaskRunLister
}

var (
	// Check that our Reconciler implements runreconciler.Interface
	_ runreconciler.Interface = (*Reconciler)(nil)
)

// ReconcileKind compares the actual state with the desired, and attempts to converge the two.
// It then updates the Status block of the Run resource with the current status of the resource.
func (c *Reconciler) ReconcileKind(ctx context.Context, run *v1alpha1.Run) pkgreconciler.Event {
	var merr error
	logger := logging.FromContext(ctx)
	logger.Infof("Reconciling Run %s/%s at %v", run.Namespace, run.Name, time.Now())

	// Check that the Run references a TaskLoop CRD.  The logic is controller.go should ensure that only this type of Run
	// is reconciled this controller but it never hurts to do some bullet-proofing.
	if run.Spec.Ref == nil ||
		run.Spec.Ref.APIVersion != taskloopv1alpha1.SchemeGroupVersion.String() ||
		run.Spec.Ref.Kind != taskloop.TaskLoopControllerName {
		logger.Errorf("Received control for a Run %s/%s that does not reference a TaskLoop custom CRD", run.Namespace, run.Name)
		return nil
	}

	// If the Run has not started, initialize the Condition and set the start time.
	if !run.HasStarted() {
		logger.Infof("Starting new Run %s/%s", run.Namespace, run.Name)
		run.Status.InitializeConditions()
		// In case node time was not synchronized, when controller has been scheduled to other nodes.
		if run.Status.StartTime.Sub(run.CreationTimestamp.Time) < 0 {
			logger.Warnf("Run %s createTimestamp %s is after the Run started %s", run.Name, run.CreationTimestamp, run.Status.StartTime)
			run.Status.StartTime = &run.CreationTimestamp
		}
		// Emit events. During the first reconcile the status of the Run may change twice
		// from not Started to Started and then to Running, so we need to sent the event here
		// and at the end of 'Reconcile' again.
		// We also want to send the "Started" event as soon as possible for anyone who may be waiting
		// on the event to perform user facing initialisations, such has reset a CI check status
		afterCondition := run.Status.GetCondition(apis.ConditionSucceeded)
		events.Emit(ctx, nil, afterCondition, run)
	}

	if run.IsDone() {
		logger.Infof("Run %s/%s is done", run.Namespace, run.Name)
		return nil
	}

	// Store the condition before reconcile
	beforeCondition := run.Status.GetCondition(apis.ConditionSucceeded)

	status := &taskloopv1alpha1.TaskLoopRunStatus{}
	if err := run.Status.DecodeExtraFields(status); err != nil {
		run.Status.MarkRunFailed(taskloopv1alpha1.TaskLoopRunReasonInternalError.String(),
			"Internal error calling DecodeExtraFields: %v", err)
		logger.Errorf("DecodeExtraFields error: %v", err.Error())
	}

	// Reconcile the Run
	if err := c.reconcile(ctx, run, status); err != nil {
		logger.Errorf("Reconcile error: %v", err.Error())
		merr = multierror.Append(merr, err)
	}

	if err := c.updateLabelsAndAnnotations(run); err != nil {
		logger.Warn("Failed to update Run labels/annotations", zap.Error(err))
		merr = multierror.Append(merr, err)
	}

	if err := run.Status.EncodeExtraFields(status); err != nil {
		run.Status.MarkRunFailed(taskloopv1alpha1.TaskLoopRunReasonInternalError.String(),
			"Internal error calling EncodeExtraFields: %v", err)
		logger.Errorf("EncodeExtraFields error: %v", err.Error())
	}

	afterCondition := run.Status.GetCondition(apis.ConditionSucceeded)
	events.Emit(ctx, beforeCondition, afterCondition, run)

	// Only transient errors that should retry the reconcile are returned.
	return merr
}

func (c *Reconciler) reconcile(ctx context.Context, run *v1alpha1.Run, status *taskloopv1alpha1.TaskLoopRunStatus) error {
	logger := logging.FromContext(ctx)

	// Get the TaskLoop referenced by the Run
	taskLoopMeta, taskLoopSpec, err := c.getTaskLoop(run)
	if err != nil {
		return nil
	}

	// Store the fetched TaskLoopSpec on the Run for auditing
	storeTaskLoopSpec(status, taskLoopSpec)

	// Propagate labels and annotations from TaskLoop to Run.
	propagateTaskLoopLabelsAndAnnotations(run, taskLoopMeta)

	// Validate TaskLoop spec
	if err := taskLoopSpec.Validate(ctx); err != nil {
		run.Status.MarkRunFailed(taskloopv1alpha1.TaskLoopRunReasonFailedValidation.String(),
			"TaskLoop %s/%s can't be Run; it has an invalid spec: %s",
			taskLoopMeta.Namespace, taskLoopMeta.Name, err)
		return nil
	}

	// Determine how many iterations of the Task will be done.
	totalIterations, err := computeIterations(run, taskLoopSpec)
	if err != nil {
		run.Status.MarkRunFailed(taskloopv1alpha1.TaskLoopRunReasonFailedValidation.String(),
			"Cannot determine number of iterations: %s", err)
		return nil
	}

	// Update status of TaskRuns.  Return the TaskRun representing the highest loop iteration.
	highestIteration, highestIterationTr, err := c.updateTaskRunStatus(logger, run, status)
	if err != nil {
		return fmt.Errorf("error updating TaskRun status for Run %s/%s: %w", run.Namespace, run.Name, err)
	}

	// Check the status of the TaskRun for the highest iteration.
	if highestIterationTr != nil {
		// If it's not done, wait for it to finish or cancel it if the run is cancelled.
		if !highestIterationTr.IsDone() {
			if run.IsCancelled() {
				logger.Infof("Run %s/%s is cancelled.  Cancelling TaskRun %s.", run.Namespace, run.Name, highestIterationTr.Name)
				b, err := getCancelPatch()
				if err != nil {
					return fmt.Errorf("Failed to make patch to cancel TaskRun %s: %v", highestIterationTr.Name, err)
				}
				if _, err := c.pipelineClientSet.TektonV1beta1().TaskRuns(run.Namespace).Patch(highestIterationTr.Name, types.JSONPatchType, b, ""); err != nil {
					run.Status.MarkRunRunning(taskloopv1alpha1.TaskLoopRunReasonCouldntCancel.String(),
						"Failed to patch TaskRun `%s` with cancellation: %v", highestIterationTr.Name, err)
					return nil
				}
				// Update status. It is still running until the TaskRun is actually cancelled.
				run.Status.MarkRunRunning(taskloopv1alpha1.TaskLoopRunReasonRunning.String(),
					"Cancelling TaskRun %s", highestIterationTr.Name)
				return nil
			}
			run.Status.MarkRunRunning(taskloopv1alpha1.TaskLoopRunReasonRunning.String(),
				"Iterations completed: %d", highestIteration-1)
			return nil
		}
		// If it failed, then retry the task if possible.  Otherwise fail the Run.
		if !highestIterationTr.IsSuccessful() {
			if run.IsCancelled() {
				run.Status.MarkRunFailed(taskloopv1alpha1.TaskLoopRunReasonCancelled.String(),
					"Run %s/%s was cancelled",
					run.Namespace, run.Name)
			} else {
				retriesDone := len(highestIterationTr.Status.RetriesStatus)
				retries := taskLoopSpec.Retries
				if retriesDone < retries {
					highestIterationTr, err = c.retryTaskRun(highestIterationTr)
					if err != nil {
						return fmt.Errorf("error retrying TaskRun %s from Run %s: %w", highestIterationTr.Name, run.Name, err)
					}
					status.TaskRuns[highestIterationTr.Name] = &taskloopv1alpha1.TaskLoopTaskRunStatus{
						Iteration: highestIteration,
						Status:    &highestIterationTr.Status,
					}
				} else {
					run.Status.MarkRunFailed(taskloopv1alpha1.TaskLoopRunReasonFailed.String(),
						"TaskRun %s has failed", highestIterationTr.Name)
				}
			}
			return nil
		}
	}

	// Move on to the next iteration (or the first iteration if there was no TaskRun).
	// Check if the Run is done.
	nextIteration := highestIteration + 1
	if nextIteration > totalIterations {
		run.Status.MarkRunSucceeded(taskloopv1alpha1.TaskLoopRunReasonSucceeded.String(),
			"All TaskRuns completed successfully")
		return nil
	}

	// Before starting up another TaskRun, check if the run was cancelled.
	if run.IsCancelled() {
		run.Status.MarkRunFailed(taskloopv1alpha1.TaskLoopRunReasonCancelled.String(),
			"Run %s/%s was cancelled",
			run.Namespace, run.Name)
		return nil
	}

	// Create a TaskRun to run this iteration.
	tr, err := c.createTaskRun(logger, taskLoopSpec, run, nextIteration)
	if err != nil {
		return fmt.Errorf("error creating TaskRun from Run %s: %w", run.Name, err)
	}

	status.TaskRuns[tr.Name] = &taskloopv1alpha1.TaskLoopTaskRunStatus{
		Iteration: nextIteration,
		Status:    &tr.Status,
	}

	run.Status.MarkRunRunning(taskloopv1alpha1.TaskLoopRunReasonRunning.String(),
		"Iterations completed: %d", highestIteration)

	return nil
}

func (c *Reconciler) getTaskLoop(run *v1alpha1.Run) (*metav1.ObjectMeta, *taskloopv1alpha1.TaskLoopSpec, error) {
	taskLoopMeta := metav1.ObjectMeta{}
	taskLoopSpec := taskloopv1alpha1.TaskLoopSpec{}
	if run.Spec.Ref != nil && run.Spec.Ref.Name != "" {
		// Use the k8 client to get the TaskLoop rather than the lister.  This avoids a timing issue where
		// the TaskLoop is not yet in the lister cache if it is created at nearly the same time as the Run.
		// See https://github.com/tektoncd/pipeline/issues/2740 for discussion on this issue.
		//
		// tl, err := c.taskLoopLister.TaskLoops(run.Namespace).Get(run.Spec.Ref.Name)
		tl, err := c.taskloopClientSet.CustomV1alpha1().TaskLoops(run.Namespace).Get(run.Spec.Ref.Name, metav1.GetOptions{})
		if err != nil {
			run.Status.MarkRunFailed(taskloopv1alpha1.TaskLoopRunReasonCouldntGetTaskLoop.String(),
				"Error retrieving TaskLoop for Run %s/%s: %s",
				run.Namespace, run.Name, err)
			return nil, nil, fmt.Errorf("Error retrieving TaskLoop for Run %s: %w", fmt.Sprintf("%s/%s", run.Namespace, run.Name), err)
		}
		taskLoopMeta = tl.ObjectMeta
		taskLoopSpec = tl.Spec
	} else {
		// Run does not require name but for TaskLoop it does.
		run.Status.MarkRunFailed(taskloopv1alpha1.TaskLoopRunReasonCouldntGetTaskLoop.String(),
			"Missing spec.ref.name for Run %s/%s",
			run.Namespace, run.Name)
		return nil, nil, fmt.Errorf("Missing spec.ref.name for Run %s", fmt.Sprintf("%s/%s", run.Namespace, run.Name))
	}
	return &taskLoopMeta, &taskLoopSpec, nil
}

func (c *Reconciler) createTaskRun(logger *zap.SugaredLogger, tls *taskloopv1alpha1.TaskLoopSpec, run *v1alpha1.Run, iteration int) (*v1beta1.TaskRun, error) {

	// Create name for TaskRun from Run name plus iteration number.
	trName := names.SimpleNameGenerator.RestrictLengthWithRandomSuffix(fmt.Sprintf("%s-%s", run.Name, fmt.Sprintf("%05d", iteration)))

	tr := &v1beta1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:            trName,
			Namespace:       run.Namespace,
			OwnerReferences: []metav1.OwnerReference{run.GetOwnerReference()},
			Labels:          getTaskRunLabels(run, strconv.Itoa(iteration)),
			Annotations:     getTaskRunAnnotations(run),
		},
		Spec: v1beta1.TaskRunSpec{
			Params:             getParameters(run, tls, iteration),
			Timeout:            tls.Timeout,
			ServiceAccountName: "",  // TODO: Implement service account name
			PodTemplate:        nil, // TODO: Implement pod template
		}}

	if tls.TaskRef != nil {
		tr.Spec.TaskRef = &v1beta1.TaskRef{
			Name: tls.TaskRef.Name,
			Kind: tls.TaskRef.Kind,
		}
	} else if tls.TaskSpec != nil {
		tr.Spec.TaskSpec = tls.TaskSpec
	}

	logger.Infof("Creating a new TaskRun object %s", trName)
	return c.pipelineClientSet.TektonV1beta1().TaskRuns(run.Namespace).Create(tr)

}

func (c *Reconciler) retryTaskRun(tr *v1beta1.TaskRun) (*v1beta1.TaskRun, error) {
	newStatus := *tr.Status.DeepCopy()
	newStatus.RetriesStatus = nil
	tr.Status.RetriesStatus = append(tr.Status.RetriesStatus, newStatus)
	tr.Status.StartTime = nil
	tr.Status.CompletionTime = nil
	tr.Status.PodName = ""
	tr.Status.SetCondition(&apis.Condition{
		Type:   apis.ConditionSucceeded,
		Status: corev1.ConditionUnknown,
	})
	return c.pipelineClientSet.TektonV1beta1().TaskRuns(tr.Namespace).UpdateStatus(tr)
}

func (c *Reconciler) updateLabelsAndAnnotations(run *v1alpha1.Run) error {
	newRun, err := c.runLister.Runs(run.Namespace).Get(run.Name)
	if err != nil {
		return fmt.Errorf("error getting Run %s when updating labels/annotations: %w", run.Name, err)
	}
	if !reflect.DeepEqual(run.ObjectMeta.Labels, newRun.ObjectMeta.Labels) || !reflect.DeepEqual(run.ObjectMeta.Annotations, newRun.ObjectMeta.Annotations) {
		mergePatch := map[string]interface{}{
			"metadata": map[string]interface{}{
				"labels":      run.ObjectMeta.Labels,
				"annotations": run.ObjectMeta.Annotations,
			},
		}
		patch, err := json.Marshal(mergePatch)
		if err != nil {
			return err
		}
		_, err = c.pipelineClientSet.TektonV1alpha1().Runs(run.Namespace).Patch(run.Name, types.MergePatchType, patch)
		return err
	}
	return nil
}

func (c *Reconciler) updateTaskRunStatus(logger *zap.SugaredLogger, run *v1alpha1.Run, status *taskloopv1alpha1.TaskLoopRunStatus) (int, *v1beta1.TaskRun, error) {
	highestIteration := 0
	var highestIterationTr *v1beta1.TaskRun = nil
	if status.TaskRuns == nil {
		status.TaskRuns = make(map[string]*taskloopv1alpha1.TaskLoopTaskRunStatus)
	}
	taskRunLabels := getTaskRunLabels(run, "")
	taskRuns, err := c.taskRunLister.TaskRuns(run.Namespace).List(labels.SelectorFromSet(taskRunLabels))
	if err != nil {
		return 0, nil, fmt.Errorf("could not list TaskRuns %#v", err)
	}
	if taskRuns == nil || len(taskRuns) == 0 {
		return 0, nil, nil
	}
	for _, tr := range taskRuns {
		lbls := tr.GetLabels()
		iterationStr := lbls[taskloop.GroupName+taskLoopIterationLabelKey]
		iteration, err := strconv.Atoi(iterationStr)
		if err != nil {
			run.Status.MarkRunFailed(taskloopv1alpha1.TaskLoopRunReasonFailedValidation.String(),
				"Error converting iteration number in TaskRun %s:  %#v", tr.Name, err)
			logger.Errorf("Error converting iteration number in TaskRun %s:  %#v", tr.Name, err)
			return 0, nil, nil
		}
		status.TaskRuns[tr.Name] = &taskloopv1alpha1.TaskLoopTaskRunStatus{
			Iteration: iteration,
			Status:    &tr.Status,
		}
		if iteration > highestIteration {
			highestIteration = iteration
			highestIterationTr = tr
		}
	}
	return highestIteration, highestIterationTr, nil
}

func getCancelPatch() ([]byte, error) {
	patches := []jsonpatch.JsonPatchOperation{{
		Operation: "add",
		Path:      "/spec/status",
		Value:     v1beta1.TaskRunSpecStatusCancelled,
	}}
	patchBytes, err := json.Marshal(patches)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal patch bytes in order to cancel: %v", err)
	}
	return patchBytes, nil
}

func computeIterations(run *v1alpha1.Run, tls *taskloopv1alpha1.TaskLoopSpec) (int, error) {
	// Find the iterate parameter.
	numberOfIterations := -1
	for _, p := range run.Spec.Params {
		if p.Name == tls.IterateParam {
			if p.Value.Type == v1beta1.ParamTypeArray {
				numberOfIterations = len(p.Value.ArrayVal)
				break
			} else {
				return 0, fmt.Errorf("The value of the iterate parameter %q is not an array", tls.IterateParam)
			}
		}
	}
	if numberOfIterations == -1 {
		return 0, fmt.Errorf("The iterate parameter %q was not found", tls.IterateParam)
	}
	return numberOfIterations, nil
}

func getParameters(run *v1alpha1.Run, tls *taskloopv1alpha1.TaskLoopSpec, iteration int) []v1beta1.Param {
	out := make([]v1beta1.Param, len(run.Spec.Params))
	for i, p := range run.Spec.Params {
		if p.Name == tls.IterateParam {
			out[i] = v1beta1.Param{
				Name:  p.Name,
				Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: p.Value.ArrayVal[iteration-1]},
			}
		} else {
			out[i] = run.Spec.Params[i]
		}
	}
	return out
}

func getTaskRunAnnotations(run *v1alpha1.Run) map[string]string {
	// Propagate annotations from Run to TaskRun.
	annotations := make(map[string]string, len(run.ObjectMeta.Annotations)+1)
	for key, val := range run.ObjectMeta.Annotations {
		annotations[key] = val
	}
	return annotations
}

func getTaskRunLabels(run *v1alpha1.Run, iterationStr string) map[string]string {
	// Propagate labels from Run to TaskRun.
	labels := make(map[string]string, len(run.ObjectMeta.Labels)+1)
	for key, val := range run.ObjectMeta.Labels {
		labels[key] = val
	}
	// Note: The Run label uses the normal Tekton group name.
	labels[pipeline.GroupName+taskLoopRunLabelKey] = run.Name
	if iterationStr != "" {
		labels[taskloop.GroupName+taskLoopIterationLabelKey] = iterationStr
	}
	return labels
}

func propagateTaskLoopLabelsAndAnnotations(run *v1alpha1.Run, taskLoopMeta *metav1.ObjectMeta) {
	// Propagate labels from TaskLoop to Run.
	if run.ObjectMeta.Labels == nil {
		run.ObjectMeta.Labels = make(map[string]string, len(taskLoopMeta.Labels)+1)
	}
	for key, value := range taskLoopMeta.Labels {
		run.ObjectMeta.Labels[key] = value
	}
	run.ObjectMeta.Labels[taskloop.GroupName+taskLoopLabelKey] = taskLoopMeta.Name

	// Propagate annotations from TaskLoop to Run.
	if run.ObjectMeta.Annotations == nil {
		run.ObjectMeta.Annotations = make(map[string]string, len(taskLoopMeta.Annotations))
	}
	for key, value := range taskLoopMeta.Annotations {
		run.ObjectMeta.Annotations[key] = value
	}
}

func storeTaskLoopSpec(status *taskloopv1alpha1.TaskLoopRunStatus, tls *taskloopv1alpha1.TaskLoopSpec) {
	// Only store the TaskLoopSpec once, if it has never been set before.
	if status.TaskLoopSpec == nil {
		status.TaskLoopSpec = tls
	}
}
