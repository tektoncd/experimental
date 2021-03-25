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
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	clientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"github.com/tektoncd/pipeline/pkg/client/injection/reconciler/pipeline/v1alpha1/run"
	listersalpha "github.com/tektoncd/pipeline/pkg/client/listers/pipeline/v1alpha1"
	listers "github.com/tektoncd/pipeline/pkg/client/listers/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/reconciler/events"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/reconciler"
)

const (
	// ReasonRunFailedValidation indicates that the reason for failure status is that Run failed validation
	ReasonRunFailedValidation = "ReasonRunFailedValidation"

	// ReasonRunFailedCreatingPipelineRun indicates that the reason for failure status is that Run failed
	// to create PipelineRun
	ReasonRunFailedCreatingPipelineRun = "ReasonRunFailedCreatingPipelineRun"
)

// Reconciler implements controller.Reconciler for Run resources.
type Reconciler struct {
	pipelineClientSet clientset.Interface
	runLister         listersalpha.RunLister
	taskRunLister     listers.TaskRunLister
}

// Check that our Reconciler implements Interface
var _ run.Interface = (*Reconciler)(nil)

// ReconcileKind implements Interface.ReconcileKind.
func (r *Reconciler) ReconcileKind(ctx context.Context, run *v1alpha1.Run) reconciler.Event {
	logger := logging.FromContext(ctx)

	if run.Spec.Ref == nil ||
		run.Spec.Ref.APIVersion != v1alpha1.SchemeGroupVersion.String() || run.Spec.Ref.Kind != kind {
		logger.Warn("Should not have been notified about Run %s/%s; will do nothing", run.Namespace, run.Name)
		return nil
	}

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

	if run.IsDone() {
		logger.Infof("Run %s/%s is done", run.Namespace, run.Name)
		return nil
	}

	var merr error

	beforeCondition := run.Status.GetCondition(apis.ConditionSucceeded)

	if err := r.reconcile(ctx, run); err != nil {
		logger.Errorf("Reconcile error: %v", err.Error())
		merr = multierror.Append(merr, controller.NewPermanentError(err))
	}

	if err := r.updateLabelsAndAnnotations(ctx, run); err != nil {
		logger.Warn("Failed to update Run labels/annotations", zap.Error(err))
		merr = multierror.Append(merr, err)
	}

	afterCondition := run.Status.GetCondition(apis.ConditionSucceeded)
	events.Emit(ctx, beforeCondition, afterCondition, run)

	// Only transient errors that should retry the reconcile are returned
	return merr

}

func (r *Reconciler) reconcile(ctx context.Context, run *v1alpha1.Run) error {
	logger := logging.FromContext(ctx)

	// confirm the run spec is valid
	if err := validateRun(run); err != nil {
		logger.Errorf("Run %s/%s is invalid because of %v", run.Namespace, run.Name, err)
		run.Status.MarkRunFailed(ReasonRunFailedValidation,
			"Run has an invalid spec: %v", err)
		return controller.NewPermanentError(fmt.Errorf("run %s/%s is invalid because of %v", run.Namespace, run.Name, err))
	}

	// fetch the taskrun and, if present, update the run status
	tr, err := getTaskRunIfExists(r.taskRunLister, run.Namespace, run.Name)
	if err != nil {
		logger.Errorf("Run %s/%s got an error fetching taskRun: %v", run.Namespace, run.Name, err)
		return fmt.Errorf("couldn't fetch taskrun %v", err)
	}
	if tr != nil {
		logger.Infof("Found a TaskRun object %s", tr.Name)
		return updateRunStatus(ctx, run, tr)
	}

	// get the pipeline that we're going to be running in a taskrun
	pSpec, err := getPipelineSpec(ctx, r.pipelineClientSet.TektonV1beta1(), run.Namespace, run.Spec.Ref.Name)
	if err != nil {
		run.Status.MarkRunFailed(ReasonRunFailedValidation,
			"Pipeline couldn't be fetched - %v", err)
		return controller.NewPermanentError(fmt.Errorf("run %s/%s is invalid because of %v", run.Namespace, run.Name, err))
		return fmt.Errorf("couldn't fetch pipeline spec %s: %v", run.Spec.Ref.Name, err)
	}
	if err := validatePipelineSpec(pSpec); err != nil {
		run.Status.MarkRunFailed(ReasonRunFailedValidation,
			"Pipeline is invalid - %v", err)
		return fmt.Errorf("pipeline spec for %s is invalid: %v", run.Spec.Ref.Name, err)
	}

	// get all the tasks we need to run this pipeline
	taskSpecs, err := getTaskSpecs(ctx, r.pipelineClientSet.TektonV1beta1(), pSpec, run.Namespace)
	if err != nil {
		run.Status.MarkRunFailed(ReasonRunFailedValidation,
			"Not all of the pipeline's tasks could be fetched - %v", err)
		return fmt.Errorf("couldn't fetch pipeline's tasks: %v", err)
	}
	if err := validateTaskSpecs(taskSpecs); err != nil {
		run.Status.MarkRunFailed(ReasonRunFailedValidation,
			"Not all tasks are valid - %v", err)
		return fmt.Errorf("pipeline's tasks are invalid: %v", err)
	}

	// use the tasks, the run and the pipeline to form a merged taskrun
	tr, err = getMergedTaskRun(run, pSpec, taskSpecs)
	if err != nil {
		run.Status.MarkRunFailed(ReasonRunFailedValidation,
			"Could not merge Tasks into TaskRun for the pipeline - %v", err)
		return fmt.Errorf("couldn't create taskrun for pipeline %s: %v", run.Spec.Ref.Name, err)
	}

	// create the taskrun
	logger.Infof("Creating a new TaskRun object %s", tr.Name)
	if _, err := r.pipelineClientSet.TektonV1beta1().TaskRuns(run.Namespace).Create(ctx, tr, metav1.CreateOptions{}); err != nil {
		logger.Errorf("Run %s/%s got an error creating TaskRun - %v", run.Namespace, run.Name, err)
		run.Status.MarkRunFailed(ReasonRunFailedCreatingPipelineRun,
			"Run got an error creating pipelineRun - %v", err)
		return err
	}

	return nil
}

func updateRunStatus(ctx context.Context, run *v1alpha1.Run, taskRun *v1beta1.TaskRun) error {
	logger := logging.FromContext(ctx)

	c := taskRun.GetStatusCondition().GetCondition(apis.ConditionSucceeded)
	if c.IsTrue() {
		logger.Infof("TaskRun created by Run %s/%s has succeeded", run.Namespace, run.Name)
		run.Status.MarkRunSucceeded(c.Reason, c.Message)
	} else if c.IsFalse() {
		logger.Infof("TaskRun created by Run %s/%s has failed", run.Namespace, run.Name)
		run.Status.MarkRunFailed(c.Reason, c.Message)
	} else if c.IsUnknown() {
		logger.Infof("TaskRun created by Run %s/%s is still running", run.Namespace, run.Name)

		reason, message := "", ""
		if c != nil {
			reason, message = c.Reason, c.Message
		}
		run.Status.MarkRunRunning(reason, message)
	} else {
		logger.Errorf("TaskRun created by Run %s/%s has an unexpected ConditionSucceeded", run.Namespace, run.Name)
		return fmt.Errorf("unexpected ConditionSucceded - %s", c)
	}

	return nil
}

func getObjectMeta(run *v1alpha1.Run) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:            run.Name,
		Namespace:       run.Namespace,
		OwnerReferences: []metav1.OwnerReference{run.GetOwnerReference()},
		Labels:          getLabels(run),
		Annotations:     getAnnotations(run),
	}
}

func getLabels(run *v1alpha1.Run) map[string]string {
	labels := make(map[string]string, len(run.ObjectMeta.Labels)+1)
	for key, val := range run.ObjectMeta.Labels {
		labels[key] = val
	}
	labels[pipeline.GroupName+pipeline.RunKey] = run.Name
	return labels
}

func getAnnotations(run *v1alpha1.Run) map[string]string {
	annotations := make(map[string]string, len(run.ObjectMeta.Annotations)+1)
	for key, val := range run.ObjectMeta.Annotations {
		annotations[key] = val
	}
	return annotations
}

func (r *Reconciler) updateLabelsAndAnnotations(ctx context.Context, run *v1alpha1.Run) error {
	newRun, err := r.runLister.Runs(run.Namespace).Get(run.Name)
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
		_, err = r.pipelineClientSet.TektonV1alpha1().Runs(run.Namespace).Patch(ctx, run.Name, types.MergePatchType, patch, metav1.PatchOptions{})
		return err
	}
	return nil
}
