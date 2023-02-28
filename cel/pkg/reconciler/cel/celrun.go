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

package cel

import (
	"context"
	"fmt"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/tektoncd/pipeline/pkg/client/injection/reconciler/pipeline/v1beta1/customrun"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/logging"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	v1 "k8s.io/api/core/v1"
	reconciler "knative.dev/pkg/reconciler"
)

const (
	// ReasonFailedValidation indicates that the reason for failure status is that CustomRun failed runtime validation
	ReasonFailedValidation = "CustomRunValidationFailed"

	// ReasonSyntaxError indicates that the reason for failure status is that a CEL expression couldn't be parsed
	ReasonSyntaxError = "SyntaxError"

	// ReasonEvaluationError indicates that the reason for failure status is that a CEL expression couldn't be evaluated
	// typically due to evaluation environment or executable program
	ReasonEvaluationError = "EvaluationError"

	// ReasonEvaluationSuccess indicates that the reason for the success status is that all CEL expressions were
	// evaluated successfully and the results were produced
	ReasonEvaluationSuccess = "EvaluationSuccess"
)

// newReconciledNormal makes a new reconciler event with event type Normal, and reason RunReconciled.
func newReconciledNormal(namespace, name string) reconciler.Event {
	return reconciler.NewEvent(v1.EventTypeNormal, "CustomRunReconciled", "CustomRun reconciled: \"%s/%s\"", namespace, name)
}

// Reconciler implements controller.Reconciler for Run resources.
type Reconciler struct {
}

// Check that our Reconciler implements Interface
var _ customrun.Interface = (*Reconciler)(nil)

// ReconcileKind implements Interface.ReconcileKind.
func (r *Reconciler) ReconcileKind(ctx context.Context, customRun *v1beta1.CustomRun) reconciler.Event {
	logger := logging.FromContext(ctx)
	logger.Infof("Reconciling CustomRun %s/%s", customRun.Namespace, customRun.Name)

	// If the Run has not started, initialize the Condition and set the start time.
	if !customRun.HasStarted() {
		logger.Infof("Starting new CustomRun %s/%s", customRun.Namespace, customRun.Name)
		customRun.Status.InitializeConditions()
		// In case node time was not synchronized, when controller has been scheduled to other nodes.
		if customRun.Status.StartTime.Sub(customRun.CreationTimestamp.Time) < 0 {
			logger.Warnf("CustomRun %s/%s createTimestamp %s is after the CustomRun started %s", customRun.Namespace, customRun.Name, customRun.CreationTimestamp, customRun.Status.StartTime)
			customRun.Status.StartTime = &customRun.CreationTimestamp
		}
	}

	if customRun.IsDone() {
		logger.Infof("CustomRun %s/%s is done", customRun.Namespace, customRun.Name)
		return nil
	}

	if err := validate(customRun); err != nil {
		logger.Errorf("CustomRun %s/%s is invalid because of %s", customRun.Namespace, customRun.Name, err)
		customRun.Status.MarkCustomRunFailed(ReasonFailedValidation,
			"CustomRun can't be run because it has an invalid spec - %v", err)
		return nil
	}

	// Create a program environment configured with the standard library of CEL functions and macros
	env, err := cel.NewEnv(cel.Declarations())
	if err != nil {
		logger.Errorf("Couldn't create a program env with standard library of CEL functions & macros when reconciling CustomRun %s/%s: %v", customRun.Namespace, customRun.Name, err)
		return err
	}

	var runResults []v1beta1.CustomRunResult
	for _, param := range customRun.Spec.Params {
		// Combine the Parse and Check phases CEL program compilation to produce an Ast and associated issues
		ast, iss := env.Compile(param.Value.StringVal)
		if iss.Err() != nil {
			logger.Errorf("CEL expression %s could not be parsed when reconciling CustomRun %s/%s: %v", param.Name, customRun.Namespace, customRun.Name, iss.Err())
			customRun.Status.MarkCustomRunFailed(ReasonSyntaxError,
				"CEL expression %s could not be parsed", param.Name, iss.Err())
			return nil
		}

		// Generate an evaluable instance of the Ast within the environment
		prg, err := env.Program(ast)
		if err != nil {
			logger.Errorf("CEL expression %s could not be evaluated when reconciling Run %s/%s: %v", param.Name, customRun.Namespace, customRun.Name, err)
			customRun.Status.MarkCustomRunFailed(ReasonEvaluationError,
				"CEL expression %s could not be evaluated", param.Name, err)
			return nil
		}

		// Evaluate the CEL expression (Ast)
		out, _, err := prg.Eval(map[string]interface{}{})
		if err != nil {
			logger.Errorf("CEL expression %s could not be evaluated when reconciling Run %s/%s: %v", param.Name, customRun.Namespace, customRun.Name, err)
			customRun.Status.MarkCustomRunFailed(ReasonEvaluationError,
				"CEL expression %s could not be evaluated", param.Name, err)
			return nil
		}

		// Evaluation of CEL expression was successful
		logger.Infof("CEL expression %s evaluated successfully when reconciling Run %s/%s", param.Name, customRun.Namespace, customRun.Name)
		runResults = append(runResults, v1beta1.CustomRunResult{
			Name:  param.Name,
			Value: fmt.Sprintf("%s", out.ConvertToType(types.StringType).Value()),
		})
	}

	// All CEL expressions were evaluated successfully
	customRun.Status.Results = append(customRun.Status.Results, runResults...)
	customRun.Status.MarkCustomRunSucceeded(ReasonEvaluationSuccess,
		"CEL expressions were evaluated successfully")

	return newReconciledNormal(customRun.Namespace, customRun.Name)
}

func validate(customRun *v1beta1.CustomRun) (errs *apis.FieldError) {
	errs = errs.Also(validateExpressionsProvided(customRun))
	errs = errs.Also(validateExpressionsType(customRun))
	return errs
}

func validateExpressionsProvided(customRun *v1beta1.CustomRun) (errs *apis.FieldError) {
	if len(customRun.Spec.Params) == 0 {
		errs = errs.Also(apis.ErrMissingField("params"))
	}
	return errs
}

func validateExpressionsType(customRun *v1beta1.CustomRun) (errs *apis.FieldError) {
	for _, param := range customRun.Spec.Params {
		if param.Value.StringVal == "" {
			errs = errs.Also(apis.ErrInvalidValue(fmt.Sprintf("CEL expression parameter %s must be a string", param.Name),
				"value").ViaFieldKey("params", param.Name))
		}
	}
	return errs
}
