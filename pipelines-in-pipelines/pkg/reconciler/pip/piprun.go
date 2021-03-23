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
	"log"
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
	"gomodules.xyz/jsonpatch/v2"
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

	// ReasonRunFailedCancelPipelineRun indicates that the reason for failure status is that Run failed
	// to cancel PipelineRun
	ReasonRunFailedCancelPipelineRun = "ReasonRunFailedCancelPipelineRun"
)

// Reconciler implements controller.Reconciler for Run resources.
type Reconciler struct {
	pipelineClientSet clientset.Interface
	runLister         listersalpha.RunLister
	pipelineRunLister listers.PipelineRunLister
}

// Check that our Reconciler implements Interface
var _ run.Interface = (*Reconciler)(nil)
var cancelPipelineRunPatchBytes []byte

func init() {
	var err error
	cancelPipelineRunPatchBytes, err = json.Marshal([]jsonpatch.JsonPatchOperation{{
		Operation: "add",
		Path:      "/spec/status",
		Value:     v1beta1.PipelineRunSpecStatusCancelled,
	}})
	if err != nil {
		log.Fatalf("failed to marshal PipelineRun cancel patch bytes: %v", err)
	}
}

// ReconcileKind implements Interface.ReconcileKind.
func (r *Reconciler) ReconcileKind(ctx context.Context, run *v1alpha1.Run) reconciler.Event {
	logger := logging.FromContext(ctx)

	if run.Spec.Ref == nil ||
		run.Spec.Ref.APIVersion != v1beta1.SchemeGroupVersion.String() || run.Spec.Ref.Kind != kind {
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

	if run.IsCancelled() {
		_, err := r.pipelineClientSet.TektonV1beta1().PipelineRuns(run.Namespace).Patch(ctx, run.Name, types.JSONPatchType, cancelPipelineRunPatchBytes, metav1.PatchOptions{}, "")
		if err != nil {
			run.Status.MarkRunFailed(ReasonRunFailedCancelPipelineRun,
				"Run got an error cancelling pipelineRun - %v", err)
		}

		run.Status.MarkRunFailed(v1alpha1.RunReasonCancelled,
			"Run %s/%s was cancelled", run.Namespace, run.Name)

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

func (r *Reconciler) reconcile(ctx context.Context, run *v1alpha1.Run) error {
	logger := logging.FromContext(ctx)

	// confirm the run spec is valid
	if err := validate(run); err != nil {
		logger.Errorf("Run %s/%s is invalid because of %v", run.Namespace, run.Name, err)
		run.Status.MarkRunFailed(ReasonRunFailedValidation,
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
		run.Status.MarkRunFailed(ReasonRunFailedCreatingPipelineRun,
			"Run got an error creating pipelineRun - %v", err)
	}

	return nil
}

func updateRunStatus(ctx context.Context, run *v1alpha1.Run, pipelineRun *v1beta1.PipelineRun) error {
	logger := logging.FromContext(ctx)

	c := pipelineRun.GetStatusCondition().GetCondition(apis.ConditionSucceeded)
	if c.IsTrue() {
		logger.Infof("PipelineRun created by Run %s/%s has succeeded", run.Namespace, run.Name)
		run.Status.MarkRunSucceeded(c.Reason, c.Message)
	} else if c.IsFalse() {
		logger.Infof("PipelineRun created by Run %s/%s has failed", run.Namespace, run.Name)
		run.Status.MarkRunFailed(c.Reason, c.Message)
	} else if c.IsUnknown() {
		logger.Infof("PipelineRun created by Run %s/%s is still running", run.Namespace, run.Name)
		run.Status.MarkRunRunning(c.Reason, c.Message)
	} else {
		logger.Errorf("PipelineRun created by Run %s/%s has an unexpected ConditionSucceeded", run.Namespace, run.Name)
		return fmt.Errorf("unexpected ConditionSucceded - %s", c)
	}

	return nil
}

func validate(run *v1alpha1.Run) (errs *apis.FieldError) {
	if run.Spec.Ref.Name == "" {
		errs = errs.Also(apis.ErrMissingField("name"))
	}
	return errs
}

func (r *Reconciler) getPipelineRun(ctx context.Context, run *v1alpha1.Run) *v1beta1.PipelineRun {
	logger := logging.FromContext(ctx)

	pr, err := r.pipelineRunLister.PipelineRuns(run.Namespace).Get(run.Name)
	if err != nil {
		logger.Errorf("Run %s/%s got an error fetching PipelineRun - %v", run.Namespace, run.Name, err)
		return nil
	}

	logger.Infof("Found a PipelineRun object %s", pr.Name)
	return pr
}

func (r *Reconciler) createPipelineRun(ctx context.Context, run *v1alpha1.Run) (*v1beta1.PipelineRun, error) {
	logger := logging.FromContext(ctx)

	pr := &v1beta1.PipelineRun{
		ObjectMeta: getObjectMeta(run),
		Spec:       getPipelineRunSpec(run),
	}

	logger.Infof("Creating a new PipelineRun object %s", pr.Name)
	return r.pipelineClientSet.TektonV1beta1().PipelineRuns(run.Namespace).Create(ctx, pr, metav1.CreateOptions{})
}

func getObjectMeta(run *v1alpha1.Run) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:            run.Name,
		Namespace:       run.Namespace,
		OwnerReferences: []metav1.OwnerReference{run.GetOwnerReference()},
		Labels:          getPipelineRunLabels(run),
		Annotations:     getPipelineRunAnnotations(run),
	}
}

func getPipelineRunSpec(run *v1alpha1.Run) v1beta1.PipelineRunSpec {
	return v1beta1.PipelineRunSpec{
		PipelineRef:        getPipelineRef(run),
		Params:             run.Spec.Params,
		ServiceAccountName: run.Spec.ServiceAccountName,
		PodTemplate:        run.Spec.PodTemplate,
		Workspaces:         run.Spec.Workspaces,
	}
}

func getPipelineRef(run *v1alpha1.Run) *v1beta1.PipelineRef {
	return &v1beta1.PipelineRef{
		Name:       run.Spec.Ref.Name,
		APIVersion: pipeline.GroupName,
	}
}

func getPipelineRunLabels(run *v1alpha1.Run) map[string]string {
	labels := make(map[string]string, len(run.ObjectMeta.Labels)+1)
	for key, val := range run.ObjectMeta.Labels {
		labels[key] = val
	}
	labels[pipeline.GroupName+pipeline.RunKey] = run.Name
	labels[pipeline.GroupName+pipeline.PipelineLabelKey] = run.Spec.Ref.Name
	return labels
}

func getPipelineRunAnnotations(run *v1alpha1.Run) map[string]string {
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
