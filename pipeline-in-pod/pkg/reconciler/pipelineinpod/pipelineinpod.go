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
	ReasonRunFailedValidation = "ReasonRunFailedValidation"
	ReasonParameterMissing    = "ReasonParameterMissing"
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

	cpr, err := toColocatedPipelineRun(run)
	if err != nil {
		return controller.NewPermanentError(fmt.Errorf("error translating to colocated pipeline run: %s", err))
	}
	getPipelineFunc, err := GetPipelineFunc(ctx, r.kubeClientSet, r.pipelineClientSet, &cpr)

	if err != nil {
		logger.Errorf("Failed to fetch pipeline func for run %s: %w", run.Name, err)
		cpr.Status.MarkFailed(ReasonCouldntGetPipeline, "Error retrieving pipeline for colocatedpipelinerun %s/%s: %s", cpr.Namespace, cpr.Name, err)
		return controller.NewPermanentError(err)
	}

	if run.IsDone() {
		logger.Infof("Run %s/%s is done", run.Namespace, run.Name)
		return nil
	}

	var merr error

	if err := r.reconcile(ctx, run.ObjectMeta, &cpr, getPipelineFunc); err != nil {
		logger.Errorf("Reconcile error: %v", err.Error())
		merr = multierror.Append(merr, controller.NewPermanentError(err))
	}

	err = updateRunFromColocatedPipelineRun(run, cpr)
	if err != nil {
		return controller.NewPermanentError(fmt.Errorf("error translating colocatedpipelinerun to run: %s", err))
	}

	afterCondition := run.Status.GetCondition(apis.ConditionSucceeded)
	events.Emit(ctx, beforeCondition, afterCondition, run)
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

	// Ensure that the PipelineRun provides all the parameters required by the Pipeline
	if err := resources.ValidateRequiredParametersProvided(&pipelineSpec.Params, &cpr.Spec.Params); err != nil {
		// This Run has failed, so we need to mark it as failed and stop reconciling it
		cpr.Status.MarkFailed(ReasonParameterMissing,
			"ColocatedPipelineRun %s parameters is missing some parameters required by Pipeline %s's parameters: %s",
			cpr.Namespace, cpr.Name, err)
		return controller.NewPermanentError(err)
	}
	pipelineSpec = ApplyParameters(pipelineSpec, cpr)
	storePipelineSpecAndMergeMeta(cpr, pipelineSpec, meta)
	r.applyParamsAndStoreTaskSpecs(ctx, cpr)
	var pod *corev1.Pod
	labelSelector := fmt.Sprintf("%s=%s", cprv1alpha1.ColocatedPipelineRunLabelKey, cpr.Name)
	pods, err := r.kubeClientSet.CoreV1().Pods(cpr.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		logger.Errorf("Error listing pods: %v", err)
		return err
	}
	for _, p := range pods.Items {
		if metav1.IsControlledBy(&p, cpr) {
			pod = &p
		}
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
	tasks, err := r.getTaskSpecs(&cpr.Status)
	if err != nil {
		return nil, err
	}
	pod, containerMappings, err := getPod(ctx, runMeta, cpr, tasks, r.Images, r.entrypointCache)
	if err != nil {
		return nil, err
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

// fetches task specs (if not inlined in pipeline spec) and copies them to CPR.Status.ChildStatuses
func (c *Reconciler) applyParamsAndStoreTaskSpecs(ctx context.Context, cpr *cprv1alpha1.ColocatedPipelineRun) error {
	logger := logging.FromContext(ctx)
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
	for i, pt := range cpr.Status.PipelineSpec.Tasks {
		var taskSpec *v1beta1.TaskSpec
		var steps []v1beta1.StepState
		if pt.TaskSpec != nil {
			for _, step := range pt.TaskSpec.Steps {
				steps = append(steps, v1beta1.StepState{Name: step.Name})
			}
			cpr.Status.PipelineSpec.Tasks[i].TaskSpec.TaskSpec = *ApplyParametersToTask(&pt.TaskSpec.TaskSpec, &pt)
		} else if pt.TaskRef != nil {
			// fetch task synchronously for now
			// this should be async in the real implementation
			task, err := c.pipelineClientSet.TektonV1beta1().Tasks(cpr.Namespace).Get(ctx, pt.TaskRef.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			taskSpec = ApplyParametersToTask(&task.Spec, &pt)
			for _, step := range taskSpec.Steps {
				steps = append(steps, v1beta1.StepState{Name: step.Name})
			}
			logger.Infof("fetched task %s for pipeline task %s", task.Name, pt.Name)
		} else {
			return fmt.Errorf("both task spec and task ref are nil")
		}

		cpr.Status.ChildStatuses = append(cpr.Status.ChildStatuses, cprv1alpha1.ChildStatus{
			PipelineTaskName: pt.Name,
			Spec:             taskSpec, // no support for custom tasks yet
			StepStatuses:     steps,
		})
	}
	return nil
}

// returns a list of pipeline tasks with task specs embedded
func (c *Reconciler) getTaskSpecs(cpr *cprv1alpha1.ColocatedPipelineRunStatus) ([]v1beta1.PipelineTask, error) {
	var tasks []v1beta1.PipelineTask
	if cpr.PipelineSpec == nil {
		return tasks, fmt.Errorf("no pipeline spec")
	}
	taskSpecs := make(map[string]v1beta1.TaskSpec)
	for _, childStatus := range cpr.ChildStatuses {
		if childStatus.Spec != nil {
			taskSpecs[childStatus.PipelineTaskName] = *childStatus.Spec
		}
	}

	for _, task := range cpr.PipelineSpec.Tasks {
		pt := task.DeepCopy()
		if pt.TaskSpec == nil {
			spec, ok := taskSpecs[pt.Name]
			if !ok {
				return tasks, fmt.Errorf("could not get spec for pipeline task %s", pt.Name)
			}
			pt.TaskSpec = &v1beta1.EmbeddedTask{TaskSpec: *spec.DeepCopy()}
		}
		tasks = append(tasks, *pt)
	}
	return tasks, nil
}
