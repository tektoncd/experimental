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
	"log"
	"reflect"
	"strconv"
	"strings"
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
	_                runreconciler.Interface = (*Reconciler)(nil)
	cancelPatchBytes []byte
)

func init() {
	fmt.Printf("initing controller at %v", time.Now())

	var err error
	patches := []jsonpatch.JsonPatchOperation{{
		Operation: "add",
		Path:      "/spec/status",
		Value:     v1beta1.TaskRunSpecStatusCancelled,
	}}
	cancelPatchBytes, err = json.Marshal(patches)
	if err != nil {
		log.Fatalf("failed to marshal patch bytes in order to cancel: %v", err)
	}
}

// ReconcileKind compares the actual state with the desired, and attempts to converge the two.
// It then updates the Status block of the Run resource with the current status of the resource.
func (c *Reconciler) ReconcileKind(ctx context.Context, run *v1alpha1.Run) pkgreconciler.Event {
	var merr error
	logger := logging.FromContext(ctx)
	logger.Infof("Reconciling Run %s/%s at %v", run.Namespace, run.Name, time.Now())

	// Check that the Run references a TaskLoop CRD.  The logic is controller.go should ensure that only this type of Run
	// is reconciled this controller but it never hurts to do some bullet-proofing.
	if run.Spec.Ref != nil &&
		(run.Spec.Ref.APIVersion != taskloopv1alpha1.SchemeGroupVersion.String() ||
			run.Spec.Ref.Kind != taskloop.TaskLoopControllerName) {
		logger.Errorf("Received control for a Run %s/%s that does not reference a TaskLoop custom CRD", run.Namespace, run.Name)
		return nil
	}

	if run.Spec.Spec != nil {
		logger.Errorf("Received control for a spec based run.")
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

	if err := c.updateLabelsAndAnnotations(ctx, run); err != nil {
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
	logger.Infof("After Condition of reconcile run: %v", afterCondition)
	// Only transient errors that should retry the reconcile are returned.
	return merr
}

func (c *Reconciler) reconcile(ctx context.Context, run *v1alpha1.Run, status *taskloopv1alpha1.TaskLoopRunStatus) error {
	logger := logging.FromContext(ctx)

	// Get the TaskLoop referenced by the Run
	taskLoopMeta, taskLoopSpec, err := c.getTaskLoop(ctx, run)
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

	// Update the status of the TaskRuns created from this Run on prior reconciliations.
	// updateTaskRunStatus() also handles TaskRun cancellation and retry.
	// It returns the total number of TaskRuns that are running now, the highest
	// iteration number processed so far, and an indicator whether any TaskRun has failed.
	totalRunning, highestIteration, taskRunFailed, err := c.updateTaskRunStatus(ctx, logger, run, status, taskLoopSpec)
	if err != nil {
		return fmt.Errorf("error updating TaskRun status for Run %s/%s: %w", run.Namespace, run.Name, err)
	}

	// Check if the run was cancelled.  Since updateTaskRunStatus() handled cancelling any running TaskRuns
	// the only thing to do here is to determine if all running TaskRuns have finished.
	if run.IsCancelled() {
		// If no TaskRuns are running, mark the Run as failed.
		if totalRunning == 0 {
			run.Status.MarkRunFailed(v1alpha1.RunReasonCancelled,
				"Run %s/%s was cancelled", run.Namespace, run.Name)
		} else {
			// The Run is still running until the TaskRuns process their cancel requests.
			run.Status.MarkRunRunning(taskloopv1alpha1.TaskLoopRunReasonRunning.String(),
				"Cancelling TaskRuns")
		}
		return nil
	}

	// Check if the Run is done.
	//   1) TaskRuns were created for all iterations OR a TaskRun has failed.
	//      (TaskRun failure stops submission of any remaining iterations.)
	//   2) All TaskRuns are done.  If there are TaskRuns running then wait
	//      for them to complete before marking the Run complete.
	if highestIteration == totalIterations || taskRunFailed {
		if totalRunning == 0 {
			if taskRunFailed {
				run.Status.MarkRunFailed(taskloopv1alpha1.TaskLoopRunReasonFailed.String(),
					"One or more TaskRuns have failed")
			} else {
				run.Status.MarkRunSucceeded(taskloopv1alpha1.TaskLoopRunReasonSucceeded.String(),
					"All TaskRuns completed successfully")
			}
		} else {
			// Update number of iterations completed.
			run.Status.MarkRunRunning(taskloopv1alpha1.TaskLoopRunReasonRunning.String(),
				"Iterations completed: %d", highestIteration-totalRunning)
		}
		return nil
	}

	// Create TaskRuns for the next iterations.  Continue creating them until the concurrency
	// limit is reached.  If the limit is unspecified, it defaults to 1 (sequential execution).
	// If the limit is 0 or negative, then TaskRuns are created for all iterations at once.
	nextIteration := highestIteration + 1
	concurrency := 1
	if taskLoopSpec.Concurrency != nil {
		concurrency = *taskLoopSpec.Concurrency
	}
	for nextIteration <= totalIterations && (concurrency <= 0 || totalRunning < concurrency) {
		// Create a TaskRun to run the next iteration.
		tr, err := c.createTaskRun(ctx, logger, taskLoopSpec, run, nextIteration)
		if err != nil {
			return fmt.Errorf("error creating TaskRun from Run %s: %w", run.Name, err)
		}
		status.TaskRuns[tr.Name] = &taskloopv1alpha1.TaskLoopTaskRunStatus{
			Iteration: nextIteration,
			Status:    &tr.Status,
		}
		totalRunning++
		nextIteration++
	}

	run.Status.MarkRunRunning(taskloopv1alpha1.TaskLoopRunReasonRunning.String(),
		"Iterations completed: %d", nextIteration-totalRunning-1)

	return nil
}

func (c *Reconciler) getTaskLoop(ctx context.Context, run *v1alpha1.Run) (*metav1.ObjectMeta, *taskloopv1alpha1.TaskLoopSpec, error) {
	taskLoopMeta := metav1.ObjectMeta{}
	taskLoopSpec := taskloopv1alpha1.TaskLoopSpec{}
	if run.Spec.Ref != nil && run.Spec.Ref.Name != "" {
		// Use the k8 client to get the TaskLoop rather than the lister.  This avoids a timing issue where
		// the TaskLoop is not yet in the lister cache if it is created at nearly the same time as the Run.
		// See https://github.com/tektoncd/pipeline/issues/2740 for discussion on this issue.
		//
		// tl, err := c.taskLoopLister.TaskLoops(run.Namespace).Get(run.Spec.Ref.Name)
		tl, err := c.taskloopClientSet.CustomV1alpha1().TaskLoops(run.Namespace).Get(ctx, run.Spec.Ref.Name, metav1.GetOptions{})
		if err != nil {
			run.Status.MarkRunFailed(taskloopv1alpha1.TaskLoopRunReasonCouldntGetTaskLoop.String(),
				"Error retrieving TaskLoop for Run %s/%s: %s",
				run.Namespace, run.Name, err)
			return nil, nil, fmt.Errorf("Error retrieving TaskLoop for Run %s: %w", fmt.Sprintf("%s/%s", run.Namespace, run.Name), err)
		}
		taskLoopMeta = tl.ObjectMeta
		taskLoopSpec = tl.Spec
	} else if run.Spec.Spec != nil {
		// @Andrea the following message is printed only if we restart the controller and not otherwise.
		fmt.Printf("recieved a run.spec.spec, but it is not yet supported %v/%v", run.Name, run)
		taskLoopMeta = metav1.ObjectMeta{Name: run.Name}
	} else {
		// Run does not require name but for TaskLoop it does.
		run.Status.MarkRunFailed(taskloopv1alpha1.TaskLoopRunReasonCouldntGetTaskLoop.String(),
			"Missing spec.ref.name for Run %s/%s",
			run.Namespace, run.Name)
		return nil, nil, fmt.Errorf("Missing spec.ref.name for Run %s", fmt.Sprintf("%s/%s", run.Namespace, run.Name))
	}
	return &taskLoopMeta, &taskLoopSpec, nil
}

func (c *Reconciler) createTaskRun(ctx context.Context, logger *zap.SugaredLogger, tls *taskloopv1alpha1.TaskLoopSpec, run *v1alpha1.Run, iteration int) (*v1beta1.TaskRun, error) {

	// Create name for TaskRun from Run name plus iteration number.
	trName := names.SimpleNameGenerator.RestrictLengthWithRandomSuffix(fmt.Sprintf("%s-%s", run.Name, fmt.Sprintf("%05d", iteration)))

	tr := &v1beta1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:            trName,
			Namespace:       run.Namespace,
			OwnerReferences: []metav1.OwnerReference{run.GetOwnerReference()},
			Labels:          getTaskRunLabels(run, strconv.Itoa(iteration), true),
			Annotations:     getTaskRunAnnotations(run),
		},
		Spec: v1beta1.TaskRunSpec{
			Params:             getParameters(run, tls, iteration),
			Timeout:            tls.Timeout,
			ServiceAccountName: run.Spec.ServiceAccountName,
			PodTemplate:        run.Spec.PodTemplate,
			Workspaces:         run.Spec.Workspaces,
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
	return c.pipelineClientSet.TektonV1beta1().TaskRuns(run.Namespace).Create(ctx, tr, metav1.CreateOptions{})

}

func (c *Reconciler) retryTaskRun(ctx context.Context, tr *v1beta1.TaskRun) (*v1beta1.TaskRun, error) {
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
	return c.pipelineClientSet.TektonV1beta1().TaskRuns(tr.Namespace).UpdateStatus(ctx, tr, metav1.UpdateOptions{})
}

func (c *Reconciler) updateLabelsAndAnnotations(ctx context.Context, run *v1alpha1.Run) error {
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
		_, err = c.pipelineClientSet.TektonV1alpha1().Runs(run.Namespace).Patch(ctx, run.Name, types.MergePatchType, patch, metav1.PatchOptions{})
		return err
	}
	return nil
}

func (c *Reconciler) updateTaskRunStatus(ctx context.Context, logger *zap.SugaredLogger, run *v1alpha1.Run, status *taskloopv1alpha1.TaskLoopRunStatus,
	taskLoopSpec *taskloopv1alpha1.TaskLoopSpec) (totalRunning int, highestIteration int, taskRunFailed bool, retryableErr error) {
	if status.TaskRuns == nil {
		status.TaskRuns = make(map[string]*taskloopv1alpha1.TaskLoopTaskRunStatus)
	}
	// List TaskRuns associated with this Run.  These TaskRuns should be recorded in the Run status but it's
	// possible that this reconcile call has been passed stale status which doesn't include a previous update.
	// Find the TaskRuns by matching labels.  Do not include the propagated labels from the Run.
	// The user could change them during the lifetime of the Run so the current labels may not be set on the
	// previously created TaskRuns.
	taskRunLabels := getTaskRunLabels(run, "", false)
	taskRuns, err := c.taskRunLister.TaskRuns(run.Namespace).List(labels.SelectorFromSet(taskRunLabels))
	if err != nil {
		retryableErr = fmt.Errorf("could not list TaskRuns %#v", err)
		return
	}
	if taskRuns == nil || len(taskRuns) == 0 {
		return
	}
	for _, tr := range taskRuns {
		lbls := tr.GetLabels()
		iterationStr := lbls[taskloop.GroupName+taskLoopIterationLabelKey]
		iteration, err := strconv.Atoi(iterationStr)
		if err != nil {
			logger.Errorf("Error converting iteration number in TaskRun %s:  %#v", tr.Name, err)
			run.Status.MarkRunFailed(taskloopv1alpha1.TaskLoopRunReasonFailedValidation.String(),
				"Error converting iteration number in TaskRun %s:  %#v", tr.Name, err)
			return
		}
		status.TaskRuns[tr.Name] = &taskloopv1alpha1.TaskLoopTaskRunStatus{
			Iteration: iteration,
			Status:    &tr.Status,
		}
		// If the TaskRun was created before the Run says it was started, then change the Run's
		// start time.  This happens when this reconcile call has been passed stale status that
		// doesn't have the start time set.  The reconcile call will set a new start time that
		// is later than TaskRuns it previously created.  The Run start time is adjusted back
		// to compensate for this problem.
		if tr.CreationTimestamp.Before(run.Status.CompletionTime) {
			run.Status.CompletionTime = tr.CreationTimestamp.DeepCopy()
		}
		// Handle TaskRun cancellation and retry.
		if err := c.processTaskRun(ctx, logger, tr, run, status, taskLoopSpec); err != nil {
			retryableErr = fmt.Errorf("error processing TaskRun %s: %#v", tr.Name, err)
			return
		}
		if iteration > highestIteration {
			highestIteration = iteration
		}
		if !tr.IsDone() {
			totalRunning++
		} else {
			if !tr.IsSuccessful() {
				taskRunFailed = true
			}
		}
	}
	return
}

func (c *Reconciler) processTaskRun(ctx context.Context, logger *zap.SugaredLogger, tr *v1beta1.TaskRun,
	run *v1alpha1.Run, status *taskloopv1alpha1.TaskLoopRunStatus, taskLoopSpec *taskloopv1alpha1.TaskLoopSpec) error {
	// If the TaskRun is running and the Run is cancelled, cancel the TaskRun.
	if !tr.IsDone() {
		if run.IsCancelled() && !tr.IsCancelled() {
			logger.Infof("Run %s/%s is cancelled.  Cancelling TaskRun %s.", run.Namespace, run.Name, tr.Name)
			if _, err := c.pipelineClientSet.TektonV1beta1().TaskRuns(run.Namespace).Patch(ctx, tr.Name, types.JSONPatchType, cancelPatchBytes, metav1.PatchOptions{}); err != nil {
				return fmt.Errorf("Failed to patch TaskRun `%s` with cancellation: %v", tr.Name, err)
			}
		}
	} else {
		// If the TaskRun failed, then retry it if possible.
		if !tr.IsSuccessful() && !run.IsCancelled() {
			retriesDone := len(tr.Status.RetriesStatus)
			retries := taskLoopSpec.Retries
			if retriesDone < retries {
				retryTr, err := c.retryTaskRun(ctx, tr)
				if err != nil {
					return fmt.Errorf("error retrying TaskRun %s from Run %s: %w", tr.Name, run.Name, err)
				}
				status.TaskRuns[retryTr.Name] = &taskloopv1alpha1.TaskLoopTaskRunStatus{
					Iteration: status.TaskRuns[retryTr.Name].Iteration,
					Status:    &retryTr.Status,
				}
			}
		}
	}
	return nil
}

func computeIterations(run *v1alpha1.Run, tls *taskloopv1alpha1.TaskLoopSpec) (int, error) {
	// Find the iterate parameter.
	numberOfIterations := -1
	for _, p := range run.Spec.Params {
		if p.Name == tls.IterateParam {
			if p.Value.Type == v1beta1.ParamTypeString {
				// If we got a string param, split it into an array, one item per line
				p.Value.ArrayVal = strings.Split(strings.TrimSuffix(p.Value.StringVal, "\n"), "\n")
			}
			numberOfIterations = len(p.Value.ArrayVal)
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
			if p.Value.Type == v1beta1.ParamTypeString {
				// If we got a string param, split it into an array, one item per line
				p.Value.ArrayVal = strings.Split(strings.TrimSuffix(p.Value.StringVal, "\n"), "\n")
			}
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

func getTaskRunLabels(run *v1alpha1.Run, iterationStr string, includeRunLabels bool) map[string]string {
	// Propagate labels from Run to TaskRun.
	labels := make(map[string]string, len(run.ObjectMeta.Labels)+1)
	if includeRunLabels {
		for key, val := range run.ObjectMeta.Labels {
			labels[key] = val
		}
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
