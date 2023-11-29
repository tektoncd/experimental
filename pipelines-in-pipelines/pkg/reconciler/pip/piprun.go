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

package pip

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	clientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"github.com/tektoncd/pipeline/pkg/client/injection/reconciler/pipeline/v1beta1/customrun"
	listers "github.com/tektoncd/pipeline/pkg/client/listers/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/reconciler/events"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
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
	customRunLister   listers.CustomRunLister
	pipelineRunLister listers.PipelineRunLister
}

// Check that our Reconciler implements Interface
var _ customrun.Interface = (*Reconciler)(nil)

// ReconcileKind implements Interface.ReconcileKind.
func (r *Reconciler) ReconcileKind(ctx context.Context, run *v1beta1.CustomRun) reconciler.Event {
	logger := logging.FromContext(ctx)

	if run.Spec.CustomRef == nil ||
		run.Spec.CustomRef.APIVersion != v1beta1.SchemeGroupVersion.String() || run.Spec.CustomRef.Kind != kind {
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

	// If the run has been cancelled, cancel the pipelineRun
	if run.IsCancelled() {
		if err := r.cancelPipelineRun(ctx, run); err != nil {
			logger.Errorf("Failed to cancel PipelineRun created by CustomRun %s/%s due to %v", run.Namespace, run.Name, err)
		}
	}

	if run.IsDone() {
		logger.Infof("Run %s/%s is done", run.Namespace, run.Name)
		return nil
	}

	var merr error

	beforeCondition := run.Status.GetCondition(apis.ConditionSucceeded)

	if err := r.reconcile(ctx, run); err != nil {
		logger.Errorf("Reconcile error: %v", err.Error())
		merr = multierror.Append(merr, err)
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

func (r *Reconciler) reconcile(ctx context.Context, run *v1beta1.CustomRun) error {
	logger := logging.FromContext(ctx)

	// confirm the run spec is valid
	if err := validate(run); err != nil {
		logger.Errorf("Run %s/%s is invalid because of %v", run.Namespace, run.Name, err)
		run.Status.MarkCustomRunFailed(ReasonRunFailedValidation,
			"Run can't be run because it has an invalid spec - %v", err)
		return controller.NewPermanentError(fmt.Errorf("run %s/%s is invalid because of %v", run.Namespace, run.Name, err))
	}

	// fetch the pipelinerun and, if present, update the run status
	if pr := r.getPipelineRun(ctx, run); pr != nil {
		return updateRunStatus(ctx, run, pr)
	}

	// pipelinerun doesn't exist yet, create a new pipelinerun
	if _, err := r.createPipelineRun(ctx, run); err != nil {
		logger.Errorf("Run %s/%s got an error creating PipelineRun - %v", run.Namespace, run.Name, err)
		run.Status.MarkCustomRunFailed(ReasonRunFailedCreatingPipelineRun,
			"Run got an error creating pipelineRun - %v", err)
	}

	return nil
}

func updateRunStatus(ctx context.Context, run *v1beta1.CustomRun, pipelineRun *v1beta1.PipelineRun) error {
	logger := logging.FromContext(ctx)

	c := pipelineRun.GetStatusCondition().GetCondition(apis.ConditionSucceeded)
	if c.IsTrue() {
		logger.Infof("PipelineRun created by CustomRun %s/%s has succeeded", run.Namespace, run.Name)
		run.Status.MarkCustomRunSucceeded(c.Reason, c.Message)
		propagateResults(run, pipelineRun)
	} else if c.IsFalse() {
		logger.Infof("PipelineRun created by CustomRun %s/%s has failed", run.Namespace, run.Name)
		run.Status.MarkCustomRunFailed(c.Reason, c.Message)
	} else if c.IsUnknown() {
		logger.Infof("PipelineRun created by CustomRun %s/%s is still running", run.Namespace, run.Name)
		if c != nil {
			run.Status.MarkCustomRunRunning(c.Reason, c.Message)
		} else {
			run.Status.MarkCustomRunRunning("Running", fmt.Sprintf("PipelineRun %s/%s has not yet completed", pipelineRun.Namespace, pipelineRun.Name))
		}
	} else {
		logger.Errorf("PipelineRun created by CustomRun %s/%s has an unexpected ConditionSucceeded", run.Namespace, run.Name)
		return fmt.Errorf("unexpected ConditionSucceded - %s", c)
	}

	return nil
}

func propagateResults(run *v1beta1.CustomRun, pipelineRun *v1beta1.PipelineRun) {
	pipelineResults := pipelineRun.Status.PipelineResults
	for _, pipelineResult := range pipelineResults {
		run.Status.Results = append(run.Status.Results, v1beta1.CustomRunResult{
			Name:  pipelineResult.Name,
			Value: pipelineResult.Value.StringVal,
		})
	}
}

func validate(run *v1beta1.CustomRun) (errs *apis.FieldError) {
	if run.Spec.CustomRef.Name == "" {
		errs = errs.Also(apis.ErrMissingField("name"))
	}
	return errs
}

func (r *Reconciler) getPipelineRun(ctx context.Context, run *v1beta1.CustomRun) *v1beta1.PipelineRun {
	logger := logging.FromContext(ctx)

	pr, err := r.pipelineRunLister.PipelineRuns(run.Namespace).Get(run.Name)
	if err != nil {
		logger.Errorf("Run %s/%s got an error fetching PipelineRun - %v", run.Namespace, run.Name, err)
		return nil
	}

	logger.Infof("Found a PipelineRun object %s", pr.Name)
	return pr
}

func (r *Reconciler) createPipelineRun(ctx context.Context, run *v1beta1.CustomRun) (*v1beta1.PipelineRun, error) {
	logger := logging.FromContext(ctx)

	var ownerPipelineRun *v1beta1.PipelineRun
	var err error
	ownerPipelineRunName := getOwnerPipelineRunName(run)
	if ownerPipelineRunName != "" {
		ownerPipelineRun, err = r.pipelineRunLister.PipelineRuns(run.Namespace).Get(ownerPipelineRunName)
		if err != nil {
			logger.Errorf("Failed to fetch the owner PipelineRun - %v", run.Namespace, ownerPipelineRunName, err)
		}
	}

	pr := &v1beta1.PipelineRun{
		ObjectMeta: getObjectMeta(run),
		Spec:       getPipelineRunSpec(run, ownerPipelineRun),
	}

	logger.Infof("Creating a new PipelineRun object %s", pr.Name)
	return r.pipelineClientSet.TektonV1beta1().PipelineRuns(run.Namespace).Create(ctx, pr, metav1.CreateOptions{})
}

func getObjectMeta(run *v1beta1.CustomRun) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      run.Name,
		Namespace: run.Namespace,
		OwnerReferences: []metav1.OwnerReference{
			*metav1.NewControllerRef(run, schema.GroupVersionKind{
				Group:   v1beta1.SchemeGroupVersion.Group,
				Version: v1beta1.SchemeGroupVersion.Version,
				Kind:    pipeline.CustomRunControllerName,
			}),
		},
		Labels:      getPipelineRunLabels(run),
		Annotations: getPipelineRunAnnotations(run),
	}
}

func (r *Reconciler) cancelPipelineRun(ctx context.Context, run *v1beta1.CustomRun) error {
	if pr := r.getPipelineRun(ctx, run); pr != nil {
		logger := logging.FromContext(ctx)
		logger.Infof("Cancelling PipelineRun created by CustomRun %s/%s", pr.Namespace, pr.Name)
		mergePatch := map[string]interface{}{
			"spec": map[string]interface{}{
				"status": v1beta1.PipelineRunSpecStatusCancelled,
			},
		}
		patch, err := json.Marshal(mergePatch)
		if err != nil {
			return err
		}
		_, err = r.pipelineClientSet.TektonV1beta1().PipelineRuns(pr.Namespace).Patch(ctx, pr.Name, types.MergePatchType, patch, metav1.PatchOptions{})
		return err
	}
	return nil
}

func getPipelineRunSpec(run *v1beta1.CustomRun, ownerPipelineRun *v1beta1.PipelineRun) v1beta1.PipelineRunSpec {
	pipelineRunSpec := v1beta1.PipelineRunSpec{
		PipelineRef:        getPipelineRef(run),
		Params:             run.Spec.Params,
		ServiceAccountName: run.Spec.ServiceAccountName,
		Workspaces:         run.Spec.Workspaces,
		Timeout:            run.Spec.Timeout,
	}
	if ownerPipelineRun != nil {
		pipelineRunSpec.PodTemplate = ownerPipelineRun.Spec.PodTemplate
	}
	return pipelineRunSpec
}

func getOwnerPipelineRunName(run *v1beta1.CustomRun) string {
	for _, ref := range run.GetOwnerReferences() {
		if ref.Kind == pipeline.PipelineRunControllerName {
			return ref.Name
		}
	}
	return ""
}

func getPipelineRef(run *v1beta1.CustomRun) *v1beta1.PipelineRef {
	if run.Spec.CustomRef.Name == "" {
		return nil
	}
	return &v1beta1.PipelineRef{
		Name:       run.Spec.CustomRef.Name,
		APIVersion: pipeline.GroupName,
	}
}

func getPipelineRunLabels(run *v1beta1.CustomRun) map[string]string {
	labels := make(map[string]string, len(run.ObjectMeta.Labels)+1)
	for key, val := range run.ObjectMeta.Labels {
		labels[key] = val
	}
	labels[pipeline.CustomRunKey] = run.Name
	labels[pipeline.PipelineLabelKey] = run.Spec.CustomRef.Name
	return labels
}

func getPipelineRunAnnotations(run *v1beta1.CustomRun) map[string]string {
	annotations := make(map[string]string, len(run.ObjectMeta.Annotations)+1)
	for key, val := range run.ObjectMeta.Annotations {
		annotations[key] = val
	}
	return annotations
}

func (r *Reconciler) updateLabelsAndAnnotations(ctx context.Context, run *v1beta1.CustomRun) error {
	newRun, err := r.customRunLister.CustomRuns(run.Namespace).Get(run.Name)
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
		_, err = r.pipelineClientSet.TektonV1beta1().CustomRuns(run.Namespace).Patch(ctx, run.Name, types.MergePatchType, patch, metav1.PatchOptions{})
		return err
	}
	return nil
}
