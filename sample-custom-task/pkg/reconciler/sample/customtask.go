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

package sample

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	runreconciler "github.com/tektoncd/pipeline/pkg/client/injection/reconciler/pipeline/v1alpha1/run"
	"github.com/tektoncd/pipeline/pkg/reconciler/events"

	"knative.dev/pkg/apis"
	"knative.dev/pkg/logging"
	pkgreconciler "knative.dev/pkg/reconciler"
)

// Reconciler reconciles a SampleSource object
type Reconciler struct{}

// Check that our Reconciler implements Run reconciler Interface
var _ runreconciler.Interface = (*Reconciler)(nil)

func (r *Reconciler) ReconcileKind(ctx context.Context, run *v1alpha1.Run) pkgreconciler.Event {
	logger := logging.FromContext(ctx)
	logger.Infof("Reconciling Run %s/%s at %v", run.Namespace, run.Name, time.Now())

	// Check that the Run references the API group, version, and kind that we want to process.
	// The logic in controller.go should ensure that only this type of Run is reconciled by this controller
	// but it never hurts to do some bullet-proofing.
	if run.Spec.Ref == nil || run.Spec.Ref.APIVersion != schemeGroupVersion.String() || run.Spec.Ref.Kind != kind {
		logger.Errorf("Received control for a Run %s/%s that does not reference apiVersion %s or kind %s", run.Namespace, run.Name, schemeGroupVersion.String(), kind)
		return nil
	}

	// Read the initial condition
	beforeCondition := run.Status.GetCondition(apis.ConditionSucceeded)

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
		// on the event to perform user facing initialisations.
		afterCondition := run.Status.GetCondition(apis.ConditionSucceeded)
		events.Emit(ctx, nil, afterCondition, run)
		beforeCondition = afterCondition
	}

	if run.IsDone() {
		logger.Infof("Run %s/%s is done", run.Namespace, run.Name)
		return nil
	}

	reconcile(run)

	afterCondition := run.Status.GetCondition(apis.ConditionSucceeded)
	events.Emit(ctx, beforeCondition, afterCondition, run)

	return nil
}

func reconcile(run *v1alpha1.Run) {
	// This sample does not use a custom resource definition.
	// If the user provides an object name then treat it as an error.
	if run.Spec.Ref.Name != "" {
		run.Status.MarkRunFailed("Failed", "This custom task does not use a custom resource definition so a reference to an object name is not expected")
		return
	}

	stringValue, err := getStringParameter(run, "string")
	if err != nil {
		run.Status.MarkRunFailed("Failed", err.Error())
		return
	}

	patternValue, err := getStringParameter(run, "pattern")
	if err != nil {
		run.Status.MarkRunFailed("Failed", err.Error())
		return
	}

	matched, err := regexp.MatchString(patternValue, stringValue)
	if err != nil {
		run.Status.MarkRunFailed("Failed", "Regular expression failure: %v", err.Error())
		return
	}

	run.Status.Results = []v1alpha1.RunResult{{
		Name:  "match",
		Value: strconv.FormatBool(matched),
	}}

	run.Status.MarkRunSucceeded("Succeeded", "Regular expression evaluated and result set.")
}

func getStringParameter(run *v1alpha1.Run, name string) (string, error) {
	param := run.Spec.GetParam(name)
	if param == nil {
		return "", fmt.Errorf("Missing parameter %q", name)
	}
	if param.Value.Type != v1beta1.ParamTypeString {
		return "", fmt.Errorf("Parameter %q has incorrect parameter type", name)
	}
	return param.Value.StringVal, nil
}
