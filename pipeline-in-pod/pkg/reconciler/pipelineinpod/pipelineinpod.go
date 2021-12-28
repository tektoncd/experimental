package pipelineinpod

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	clientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	pipelinerunreconciler "github.com/tektoncd/pipeline/pkg/client/injection/reconciler/pipeline/v1beta1/pipelinerun"
	listers "github.com/tektoncd/pipeline/pkg/client/listers/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/reconciler/events"
	"github.com/tektoncd/pipeline/pkg/reconciler/pipelinerun/resources"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/reconciler"
)

const (
	ReasonCouldntGetPipeline = "ReasonCouldntGetPipeline"
)

// Reconciler implements controller.Reconciler for Run resources.
type Reconciler struct {
	pipelineClientSet clientset.Interface
	kubeClientSet     kubernetes.Interface
	pipelineRunLister listers.PipelineRunLister
	Images            pipeline.Images
	entrypointCache   EntrypointCache
}

// Check that our Reconciler implements Interface
var _ pipelinerunreconciler.Interface = (*Reconciler)(nil)

// ReconcileKind implements Interface.ReconcileKind.
func (r *Reconciler) ReconcileKind(ctx context.Context, pr *v1beta1.PipelineRun) reconciler.Event {
	logger := logging.FromContext(ctx)
	logger.Infof("Reconciling PipelineRun %s/%s at %v", pr.Namespace, pr.Name, time.Now())
	before := pr.Status.GetCondition(apis.ConditionSucceeded)

	// If the PipelineRun has not started, initialize the Condition and set the start time.
	if !pr.HasStarted() {
		logger.Infof("Starting new PipelineRun %s/%s", pr.Namespace, pr.Name)
		pr.Status.InitializeConditions()
		// In case node time was not synchronized, when controller has been scheduled to other nodes.
		if pr.Status.StartTime.Sub(pr.CreationTimestamp.Time) < 0 {
			logger.Warnf("Run %s/%s createTimestamp %s is after the PipelineRun started %s", pr.Namespace, pr.Name, pr.CreationTimestamp, pr.Status.StartTime)
			pr.Status.StartTime = &pr.CreationTimestamp
		}
		// Send the "Started" event
		afterCondition := pr.Status.GetCondition(apis.ConditionSucceeded)
		events.Emit(ctx, nil, afterCondition, pr)
	}

	getPipelineFunc, err := resources.GetPipelineFunc(ctx, r.kubeClientSet, r.pipelineClientSet, pr)
	if err != nil {
		logger.Errorf("Failed to fetch pipeline func for pipeline %s: %w", pr.Spec.PipelineRef.Name, err)
		pr.Status.MarkFailed(ReasonCouldntGetPipeline, "Error retrieving pipeline for pipelinerun %s/%s: %s",
			pr.Namespace, pr.Name, err)
		return r.finishReconcileUpdateEmitEvents(ctx, pr, before, nil)
	}

	if pr.IsDone() {
		logger.Infof("Run %s/%s is done", pr.Namespace, pr.Name)
		return r.finishReconcileUpdateEmitEvents(ctx, pr, before, nil)
	}

	// Make sure that the PipelineRun status is in sync with the actual TaskRuns
	/*
		err = r.updatePipelineRunStatusFromInformer(ctx, pr)
		if err != nil {
			// This should not fail. Return the error so we can re-try later.
			logger.Errorf("Error while syncing the pipelinerun status: %v", err.Error())
			return r.finishReconcileUpdateEmitEvents(ctx, pr, before, err)
		} */

	// Reconcile this copy of the pipelinerun and then write back any status or label
	// updates regardless of whether the reconciliation errored out.
	if err = r.reconcile(ctx, pr, getPipelineFunc); err != nil {
		logger.Errorf("Reconcile error: %v", err.Error())
	}

	if err = r.finishReconcileUpdateEmitEvents(ctx, pr, before, err); err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) finishReconcileUpdateEmitEvents(ctx context.Context, pr *v1beta1.PipelineRun, beforeCondition *apis.Condition, previousError error) error {
	logger := logging.FromContext(ctx)

	afterCondition := pr.Status.GetCondition(apis.ConditionSucceeded)
	events.Emit(ctx, beforeCondition, afterCondition, pr)
	_, err := r.updateLabelsAndAnnotations(ctx, pr)
	if err != nil {
		logger.Warn("Failed to update PipelineRun labels/annotations", err)
		events.EmitError(controller.GetEventRecorder(ctx), err, pr)
	}

	merr := multierror.Append(previousError, err).ErrorOrNil()
	if controller.IsPermanentError(previousError) {
		return controller.NewPermanentError(merr)
	}
	return merr
}

func (r *Reconciler) reconcile(ctx context.Context, pr *v1beta1.PipelineRun, getPipelineFunc resources.GetPipeline) error {
	logger := logging.FromContext(ctx)
	var pod *corev1.Pod

	// Get pod associated with pipelinerun, if it exists
	podName, err := getPodName(pr)
	if err != nil {
		logger.Errorf("Error getting pod name associated with PR %s: %s", pr.Name, err)
		return err
	}
	if podName != "" {
		pod, err = r.kubeClientSet.CoreV1().Pods(pr.Namespace).Get(ctx, podName, metav1.GetOptions{})
		if k8serrors.IsNotFound(err) {
			// Keep going, this will result in the Pod being created below.
			logger.Infof("Pod %s not found for PR %s", podName, pr.Name)
		} else if err != nil {
			// This is considered a transient error, so we return error, do not update
			// the task run condition, and return an error which will cause this key to
			// be requeued for reconcile.
			logger.Errorf("Error getting pod %q: %v", podName, err)
			return err
		}
	} else {
		// List pods that have a label with this PipelineRun name.  Do not include other labels from the
		// PipelineRun in this selector.  The user could change them during the lifetime of the PipelineRun so the
		// current labels may not be set on a previously created Pod.
		labelSelector := fmt.Sprintf("%s=%s", pipeline.PipelineRunLabelKey, pr.Name)
		pos, err := r.kubeClientSet.CoreV1().Pods(pr.Namespace).List(ctx, metav1.ListOptions{
			LabelSelector: labelSelector,
		})
		if err != nil {
			logger.Errorf("Error listing pods: %v", err)
			return err
		}
		for index := range pos.Items {
			po := pos.Items[index]
			if metav1.IsControlledBy(&po, pr) && !DidPipelineRunFail(po.Status.ContainerStatuses) {
				pod = &po
				logger.Infof("Got pod via label selector")
			}
		}
	}

	// if it does not exist, get pipeline from pipelinerun and use it to create a pod
	if pod == nil {
		meta, pipelineSpec, err := resources.GetPipelineData(ctx, pr, getPipelineFunc)
		if err != nil {
			logger.Errorf("Failed to determine Pipeline spec to use for pipelinerun %s: %v", pr.Name, err)
			pr.Status.MarkFailed(ReasonCouldntGetPipeline,
				"Error retrieving pipeline for pipelinerun %s/%s: %s",
				pr.Namespace, pr.Name, err)
			return controller.NewPermanentError(err)
		}
		storePipelineSpecAndMergeMeta(pr, pipelineSpec, meta)
		addTasksToPipelineRun(logger, pr, pipelineSpec)
		// TODO: resolve tasks
		// TODO: validate pipeline
		logger.Infof("Creating pod for PipelineRun %s in namespace %s", pr.Name, pr.Namespace)
		pod, err = r.createPod(ctx, pr, pipelineSpec)
		if err != nil {
			logger.Errorf("Error creating pod for PipelineRun %s: %v", pr.Name, err)
			return err
		}
		err = applyPodName(pr, pod.Name)
		if err != nil {
			logger.Errorf("Error applying pod name to PipelineRun %s: %v", pr.Name, err)
			return err
		}
	}

	// Update pipelinerun status with pod status
	logger.Infof("updating PR %s status with status of pod %s", pr.Name, pod.Name)
	prs, err := MakePipelineRunStatus(logger, *pr, pod)
	if err != nil {
		return err
	}
	pr.Status = prs
	return nil
}

func getPodName(pr *v1beta1.PipelineRun) (string, error) {
	podName := ""
	for _, trs := range pr.Status.TaskRuns {
		if podName != "" && (trs.Status.PodName != podName) {
			return "", fmt.Errorf("TaskRuns have different pod names: %s and %s", podName, trs.Status.PodName)
		}
		podName = trs.Status.PodName
	}
	return podName, nil
}

func applyPodName(pr *v1beta1.PipelineRun, podName string) error {
	if podName == "" {
		return fmt.Errorf("empty pod name")
	}
	for _, trs := range pr.Status.TaskRuns {
		trs.Status.PodName = podName
	}
	return nil
}

func addTasksToPipelineRun(logger *zap.SugaredLogger, pr *v1beta1.PipelineRun, ps *v1beta1.PipelineSpec) {
	// TODO (maybe): Add Task specs to pr Spec?
	// Add taskrun statuses to pr status
	for _, pt := range ps.Tasks {
		prtrs := v1beta1.PipelineRunTaskRunStatus{Status: &v1beta1.TaskRunStatus{}}
		pr.Status.TaskRuns[pt.Name] = &prtrs
		logger.Infof("Adding status for TR %s to PR %s", pt.Name, pr.Name)
	}
}

func storePipelineSpecAndMergeMeta(pr *v1beta1.PipelineRun, ps *v1beta1.PipelineSpec, meta *metav1.ObjectMeta) error {
	// Only store the PipelineSpec once, if it has never been set before.
	if pr.Status.PipelineSpec == nil {
		pr.Status.PipelineSpec = ps

		// Propagate labels from Pipeline to PipelineRun.
		if pr.ObjectMeta.Labels == nil {
			pr.ObjectMeta.Labels = make(map[string]string, len(meta.Labels)+1)
		}
		for key, value := range meta.Labels {
			pr.ObjectMeta.Labels[key] = value
		}
		pr.ObjectMeta.Labels[pipeline.PipelineLabelKey] = meta.Name

		// Propagate annotations from Pipeline to PipelineRun.
		if pr.ObjectMeta.Annotations == nil {
			pr.ObjectMeta.Annotations = make(map[string]string, len(meta.Annotations))
		}
		for key, value := range meta.Annotations {
			pr.ObjectMeta.Annotations[key] = value
		}
	}
	return nil
}

func (r *Reconciler) createPod(ctx context.Context, pr *v1beta1.PipelineRun, ps *v1beta1.PipelineSpec) (*corev1.Pod, error) {
	pod, err := getPod(ctx, pr, ps, r.Images, r.entrypointCache)
	if err != nil {
		return nil, err
	}
	pod, err = r.kubeClientSet.CoreV1().Pods(pr.Namespace).Create(ctx, pod, metav1.CreateOptions{})
	return pod, err
}

func (c *Reconciler) updateLabelsAndAnnotations(ctx context.Context, pr *v1beta1.PipelineRun) (v1beta1.PipelineRun, error) {
	// TODO
	return *pr, nil
}
