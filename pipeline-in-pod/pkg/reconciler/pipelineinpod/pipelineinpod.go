package pipelineinpod

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	cprv1alpha1 "github.com/tektoncd/experimental/pipeline-in-pod/pkg/apis/colocatedpipelinerun/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	clientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"github.com/tektoncd/pipeline/pkg/client/injection/reconciler/pipeline/v1alpha1/run"
	"github.com/tektoncd/pipeline/pkg/reconciler/events"
	"github.com/tektoncd/pipeline/pkg/reconciler/pipelinerun/resources"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/controller"
	logging "knative.dev/pkg/logging"
	"knative.dev/pkg/reconciler"
)

const (
	ReasonCouldntGetPipeline = "ReasonCouldntGetPipeline"
	// ReasonRunFailedValidation indicates that the reason for failure status is that Run failed validation
	ReasonRunFailedValidation     = "ReasonRunFailedValidation"
	ReasonParameterMissing        = "ReasonParameterMissing"
	ReasonTimedOut                = "ReasonTimedOut"
	ReasonCouldntGetTask          = "ReasonCouldntGetTask"
	ReasonInvalidWorkspaceBinding = "ReasonInvalidWorkspaceBinding"
)

// Reconciler implements controller.Reconciler for Run resources.
type Reconciler struct {
	pipelineClientSet clientset.Interface
	kubeClientSet     kubernetes.Interface
	Images            pipeline.Images
	entrypointCache   EntrypointCache
}

// Check that our Reconciler implements Interface
var _ run.Interface = (*Reconciler)(nil)

// ReconcileKind implements Interface.ReconcileKind.
func (r *Reconciler) ReconcileKind(ctx context.Context, run *v1alpha1.Run) reconciler.Event {
	logger := logging.FromContext(ctx)

	if run.UID == "" {
		run.SetUID(types.UID(uuid.New().String()))
	}
	beforeCondition := run.Status.GetCondition(apis.ConditionSucceeded)

	logger.Infof("Reconciling Run %s/%s at %v", run.Namespace, run.Name, time.Now())
	// If the Run has not started, initialize the Condition and set the start time.
	if !run.HasStarted() {
		logger.Infof("Starting new Run %s/%s", run.Namespace, run.Name)
		run.Status.InitializeConditions()
		// In case node time was not synchronized, when controller has been scheduled to other nodes.
		if run.Status.StartTime.Sub(run.CreationTimestamp.Time) < 0 {
			logger.Warnf("Run %s/%s createTimestamp %s is after the Run started %s", run.Namespace, run.Name, run.CreationTimestamp, run.Status.StartTime)
			run.Status.StartTime = &run.CreationTimestamp
		}
		// Send the "Started" event
		afterCondition := run.Status.GetCondition(apis.ConditionSucceeded)
		events.Emit(ctx, nil, afterCondition, run)
	}

	cpr, err := ToColocatedPipelineRun(run)
	if err != nil {
		return controller.NewPermanentError(fmt.Errorf("error translating to colocated pipeline run: %s", err))
	}
	getPipelineFunc, err := GetPipelineFunc(ctx, r.kubeClientSet, r.pipelineClientSet, &cpr)

	if err != nil {
		logger.Errorf("Failed to fetch pipeline func for run %s: %w", run.Name, err)
		cpr.Status.MarkFailed(ReasonCouldntGetPipeline, "Error retrieving pipeline for colocatedpipelinerun %s/%s: %s", cpr.Namespace, cpr.Name, err)
		return controller.NewPermanentError(err)
	}

	if cpr.IsDone() {
		logger.Infof("Run %s/%s is done", run.Namespace, run.Name)
		return nil
	}

	// We are not using run.HasTimedOut because run timeouts are ignored in favor of colocatedpipelinerun timeouts
	if hasTimedOut(ctx, cpr) {
		timeout := cpr.PipelineTimeout(ctx)
		logger.Infof("Run %s/%s timed out after %s", run.Namespace, run.Name, timeout)
		err = r.failColocatedPipelineRun(ctx, &cpr, ReasonTimedOut, fmt.Sprintf("timed out after %s", timeout))
		if err != nil {
			return fmt.Errorf("error failing colocatedpipelinerun: %s", err)
		}
		err = UpdateRunFromColocatedPipelineRun(run, cpr)
		if err != nil {
			return controller.NewPermanentError(fmt.Errorf("error translating colocatedpipelinerun to run: %s", err))
		}
		return nil
	}
	var merr error

	if err := r.reconcile(ctx, run.ObjectMeta, &cpr, getPipelineFunc); err != nil {
		logger.Errorf("Reconcile error: %v", err.Error())
		merr = multierror.Append(merr, controller.NewPermanentError(err))
	}

	err = UpdateRunFromColocatedPipelineRun(run, cpr)
	if err != nil {
		return controller.NewPermanentError(fmt.Errorf("error translating colocatedpipelinerun to run: %s", err))
	}
	afterCondition := run.Status.GetCondition(apis.ConditionSucceeded)
	events.Emit(ctx, beforeCondition, afterCondition, run)

	if run.Status.StartTime != nil {
		// Compute the time since the task started.
		elapsed := time.Since(run.Status.StartTime.Time)
		// Snooze this resource until the timeout has elapsed.
		return controller.NewRequeueAfter(cpr.PipelineTimeout(ctx) - elapsed)
	}
	return merr
}

func (r *Reconciler) reconcile(ctx context.Context, runMeta metav1.ObjectMeta, cpr *cprv1alpha1.ColocatedPipelineRun, pipelineFunc GetPipeline) error {
	logger := logging.FromContext(ctx)
	if err := cprv1alpha1.ValidateColocatedPipelineRun(cpr); err != nil {
		logger.Errorf("Run %s/%s is invalid because of %v", cpr.Namespace, cpr.Name, err)
		cpr.Status.MarkFailed(ReasonRunFailedValidation, "Run has an invalid spec: %v", err)
		return controller.NewPermanentError(fmt.Errorf("run %s/%s is invalid because of %v", cpr.Namespace, cpr.Name, err))
	}

	meta, pipelineSpec, err := GetPipelineData(ctx, cpr, pipelineFunc)
	if err != nil {
		logger.Errorf("Failed to determine Pipeline spec to use for pipelinerun %s: %v", cpr.Name, err)
		cpr.Status.MarkFailed(ReasonCouldntGetPipeline,
			"Error retrieving pipeline for pipelinerun %s/%s: %s", cpr.Namespace, cpr.Name, err)
		return controller.NewPermanentError(err)
	}

	// Ensure that the ColocatedPipelineRun provides all the parameters required by the Pipeline
	if err := resources.ValidateRequiredParametersProvided(&pipelineSpec.Params, &cpr.Spec.Params); err != nil {
		// This Run has failed, so we need to mark it as failed and stop reconciling it
		cpr.Status.MarkFailed(ReasonParameterMissing,
			"ColocatedPipelineRun %s parameters is missing some parameters required by Pipeline %s's parameters: %s",
			cpr.Namespace, cpr.Name, err)
		return controller.NewPermanentError(err)
	}
	// Ensure that the workspaces expected by the Pipeline are provided by the ColocatedPipelineRun.
	if err := ValidateWorkspaceBindings(pipelineSpec, cpr); err != nil {
		cpr.Status.MarkFailed(ReasonInvalidWorkspaceBinding,
			"ColocatedPipelineRun %s/%s doesn't bind Pipeline %s/%s's Workspaces correctly: %s",
			cpr.Namespace, cpr.Name, cpr.Namespace, meta.Name, err)
		return controller.NewPermanentError(err)
	}

	pipelineSpec = ApplyParametersToPipeline(pipelineSpec, cpr)
	pipelineSpec = ApplyWorkspacesToPipeline(pipelineSpec, cpr)
	storePipelineSpecAndMergeMeta(cpr, pipelineSpec, meta)
	r.applyTasks(ctx, cpr)
	pod, err := r.getPodForColocatedPipelineRun(ctx, cpr)
	if err != nil {
		logger.Errorf("Error getting pod for colocatedPipelineRun %s: %s", cpr.Name, err)
		return err
	}
	if pod == nil {
		pod, err = r.createPod(ctx, runMeta, cpr, pipelineSpec)
		if err != nil {
			logger.Errorf("Error creating pod for ColocatedPipelineRun %s: %v", cpr.Name, err)
			return err
		}
	}
	logger.Infof("updating PR %s status with status of pod %s", cpr.Name, pod.Name)
	err = updateContainerStatuses(logger, &cpr.Status, pod)
	if err != nil {
		return err
	}
	cprs, err := MakeColocatedPipelineRunStatus(logger, *cpr, pod)
	if err != nil {
		return err
	}
	cpr.Status = cprs
	return nil
}

func (r *Reconciler) createPod(ctx context.Context, runMeta metav1.ObjectMeta, cpr *cprv1alpha1.ColocatedPipelineRun, ps *v1beta1.PipelineSpec) (*corev1.Pod, error) {
	volumes, volumeMounts, err := ApplyWorkspacesToTasks(ctx, runMeta, cpr)
	if err != nil {
		return nil, err
	}
	tasks, err := getPipelineTaskSpecs(ctx, &cpr.Status)
	if err != nil {
		return nil, err
	}
	pod, containerMappings, err := getPod(ctx, runMeta, cpr, tasks, r.Images, r.entrypointCache, volumes, volumeMounts)
	if err != nil {
		return nil, controller.NewPermanentError(err)
	}
	for i, childStatus := range cpr.Status.ChildStatuses {
		stepInfo := containerMappings[childStatus.PipelineTaskName]
		for j, stepStatus := range childStatus.StepStatuses {
			containerName, ok := stepInfo[stepStatus.Name]
			if !ok {
				return nil, fmt.Errorf("no container found for step %s in pipeline task %s", stepStatus.Name, childStatus.PipelineTaskName)
			}
			cpr.Status.ChildStatuses[i].StepStatuses[j].ContainerName = containerName
		}
	}
	pod, err = r.kubeClientSet.CoreV1().Pods(cpr.Namespace).Create(ctx, pod, metav1.CreateOptions{})
	return pod, err
}

// use cpr.ChildStatuses[].Spec as source of truth instead of modifying spec embedded in pipeline
// this spec has params and workspaces substituted
func getPipelineTaskSpecs(ctx context.Context, cpr *cprv1alpha1.ColocatedPipelineRunStatus) ([]v1beta1.PipelineTask, error) {
	var tasks []v1beta1.PipelineTask
	if cpr.PipelineSpec == nil {
		return tasks, fmt.Errorf("no pipeline spec")
	}
	taskSpecs := make(map[string]v1beta1.TaskSpec)
	for _, childStatus := range cpr.ChildStatuses {
		if childStatus.Spec != nil {
			taskSpecs[childStatus.PipelineTaskName] = *childStatus.Spec
		} else {
			return tasks, fmt.Errorf("could not get spec for pipeline task %s", childStatus.PipelineTaskName)
		}
	}

	for _, task := range cpr.PipelineSpec.Tasks {
		pt := task.DeepCopy()
		spec, ok := taskSpecs[pt.Name]
		if !ok {
			return tasks, fmt.Errorf("could not get spec for pipeline task %s", pt.Name)
		}
		pt.TaskSpec = &v1beta1.EmbeddedTask{TaskSpec: *spec.DeepCopy()}
		tasks = append(tasks, *pt)
	}
	return tasks, nil
}

func (r *Reconciler) getPodForColocatedPipelineRun(ctx context.Context, cpr *cprv1alpha1.ColocatedPipelineRun) (*corev1.Pod, error) {
	logger := logging.FromContext(ctx)
	labelSelector := fmt.Sprintf("%s=%s", cprv1alpha1.ColocatedPipelineRunLabelKey, cpr.Name)
	pods, err := r.kubeClientSet.CoreV1().Pods(cpr.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		logger.Errorf("Error listing pods: %v", err)
		return nil, err
	}
	for _, p := range pods.Items {
		if metav1.IsControlledBy(&p, cpr) {
			return &p, nil
		}
	}
	return nil, nil
}

func hasTimedOut(ctx context.Context, cpr cprv1alpha1.ColocatedPipelineRun) bool {
	if cpr.Status.StartTime == nil || cpr.Status.StartTime.IsZero() {
		return false
	}
	timeout := cpr.PipelineTimeout(ctx)
	runtime := time.Since(cpr.Status.StartTime.Time)
	return runtime > timeout
}

func (c *Reconciler) failColocatedPipelineRun(ctx context.Context, cpr *cprv1alpha1.ColocatedPipelineRun, reason, message string) error {
	logger := logging.FromContext(ctx)

	logger.Warnf("stopping colocatedPipelineRun %q because of %q", cpr.Name, reason)
	cpr.Status.MarkFailed(reason, message)

	pod, err := c.getPodForColocatedPipelineRun(ctx, cpr)
	if err != nil {
		logger.Errorf("Error getting pod for colocatedPipelineRun %s: %s", cpr.Name, err)
		return err
	}
	if pod == nil {
		logger.Info("No pod created for ColocatedPipelineRun %s", cpr.Name)
		return nil
	}

	err = c.kubeClientSet.CoreV1().Pods(cpr.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		logger.Infof("Failed to terminate pod: %v", err)
		return err
	}

	for i, task := range cpr.Status.ChildStatuses {
		for j, step := range task.StepStatuses {
			if step.Running != nil {
				step.Terminated = &corev1.ContainerStateTerminated{
					ExitCode:   1,
					StartedAt:  step.Running.StartedAt,
					FinishedAt: *cpr.Status.CompletionTime,
					Reason:     reason,
				}
				step.Running = nil
				cpr.Status.ChildStatuses[i].StepStatuses[j] = step
			}

			if step.Waiting != nil {
				step.Terminated = &corev1.ContainerStateTerminated{
					ExitCode:   1,
					FinishedAt: *cpr.Status.CompletionTime,
					Reason:     reason,
				}
				step.Waiting = nil
				cpr.Status.ChildStatuses[i].StepStatuses[j] = step
			}
		}
	}
	return nil
}
