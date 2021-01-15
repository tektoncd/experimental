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
	"github.com/tektoncd/pipeline/pkg/client/injection/reconciler/pipeline/v1alpha1/run"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/logging"

	v1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	v1 "k8s.io/api/core/v1"
	reconciler "knative.dev/pkg/reconciler"
)

const (
	// ReasonFailedValidation indicates that the reason for failure status is that Run failed runtime validation
	ReasonFailedValidation = "RunValidationFailed"

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
	return reconciler.NewEvent(v1.EventTypeNormal, "RunReconciled", "Run reconciled: \"%s/%s\"", namespace, name)
}

// Reconciler implements controller.Reconciler for Run resources.
type Reconciler struct {
}

// Check that our Reconciler implements Interface
var _ run.Interface = (*Reconciler)(nil)

// ReconcileKind implements Interface.ReconcileKind.
func (r *Reconciler) ReconcileKind(ctx context.Context, run *v1alpha1.Run) reconciler.Event {
	logger := logging.FromContext(ctx)
	logger.Infof("Reconciling Run %s/%s", run.Namespace, run.Name)

	// If the Run has not started, initialize the Condition and set the start time.
	if !run.HasStarted() {
		logger.Infof("Starting new Run %s/%s", run.Namespace, run.Name)
		run.Status.InitializeConditions()
		// In case node time was not synchronized, when controller has been scheduled to other nodes.
		if run.Status.StartTime.Sub(run.CreationTimestamp.Time) < 0 {
			logger.Warnf("Run %s/%s createTimestamp %s is after the Run started %s", run.Namespace, run.Name, run.CreationTimestamp, run.Status.StartTime)
			run.Status.StartTime = &run.CreationTimestamp
		}
	}

	if run.IsDone() {
		logger.Infof("Run %s/%s is done", run.Namespace, run.Name)
		return nil
	}

	if err := validate(run); err != nil {
		logger.Errorf("Run %s/%s is invalid because of %s", run.Namespace, run.Name, err)
		run.Status.MarkRunFailed(ReasonFailedValidation,
			"Run can't be run because it has an invalid spec - %v", err)
		return nil
	}

	// Create a program environment configured with the standard library of CEL functions and macros
	env, err := cel.NewEnv(cel.Declarations())
	if err != nil {
		logger.Errorf("Couldn't create a program env with standard library of CEL functions & macros when reconciling Run %s/%s: %v", run.Namespace, run.Name, err)
		return err
	}

	var runResults []v1alpha1.RunResult
	for _, param := range run.Spec.Params {
		// Combine the Parse and Check phases CEL program compilation to produce an Ast and associated issues
		ast, iss := env.Compile(param.Value.StringVal)
		if iss.Err() != nil {
			logger.Errorf("CEL expression %s could not be parsed when reconciling Run %s/%s: %v", param.Name, run.Namespace, run.Name, iss.Err())
			run.Status.MarkRunFailed(ReasonSyntaxError,
				"CEL expression %s could not be parsed", param.Name, iss.Err())
			return nil
		}

		// Generate an evaluable instance of the Ast within the environment
		prg, err := env.Program(ast)
		if err != nil {
			logger.Errorf("CEL expression %s could not be evaluated when reconciling Run %s/%s: %v", param.Name, run.Namespace, run.Name, err)
			run.Status.MarkRunFailed(ReasonEvaluationError,
				"CEL expression %s could not be evaluated", param.Name, err)
			return nil
		}

		// Evaluate the CEL expression (Ast)
		out, _, err := prg.Eval(map[string]interface{}{})
		if err != nil {
			logger.Errorf("CEL expression %s could not be evaluated when reconciling Run %s/%s: %v", param.Name, run.Namespace, run.Name, err)
			run.Status.MarkRunFailed(ReasonEvaluationError,
				"CEL expression %s could not be evaluated", param.Name, err)
			return nil
		}

		// Evaluation of CEL expression was successful
		logger.Infof("CEL expression %s evaluated successfully when reconciling Run %s/%s", param.Name, run.Namespace, run.Name)
		runResults = append(runResults, v1alpha1.RunResult{
			Name:  param.Name,
			Value: fmt.Sprintf("%s", out.ConvertToType(types.StringType).Value()),
		})
	}

	// All CEL expressions were evaluated successfully
	run.Status.Results = append(run.Status.Results, runResults...)
	run.Status.MarkRunSucceeded(ReasonEvaluationSuccess,
		"CEL expressions were evaluated successfully")

	return newReconciledNormal(run.Namespace, run.Name)
}

func validate(run *v1alpha1.Run) (errs *apis.FieldError) {
	errs = errs.Also(validateExpressionsProvided(run))
	errs = errs.Also(validateExpressionsType(run))
	return errs
}

func validateExpressionsProvided(run *v1alpha1.Run) (errs *apis.FieldError) {
	if len(run.Spec.Params) == 0 {
		errs = errs.Also(apis.ErrMissingField("params"))
	}
	return errs
}

func validateExpressionsType(run *v1alpha1.Run) (errs *apis.FieldError) {
	for _, param := range run.Spec.Params {
		if param.Value.StringVal == "" {
			errs = errs.Also(apis.ErrInvalidValue(fmt.Sprintf("CEL expression parameter %s must be a string", param.Name),
				"value").ViaFieldKey("params", param.Name))
		}
	}
	return errs
}
