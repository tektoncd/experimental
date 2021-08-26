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

package celevalrun

import (
	"context"
	"encoding/json"
	"fmt"
	gocel "github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	celtypes "github.com/google/cel-go/common/types"
	"github.com/hashicorp/go-multierror"
	"github.com/tektoncd/experimental/celeval/pkg/apis/celeval"
	celevalv1alpha1 "github.com/tektoncd/experimental/celeval/pkg/apis/celeval/v1alpha1"
	celevalclientset "github.com/tektoncd/experimental/celeval/pkg/client/clientset/versioned"
	listersceleval "github.com/tektoncd/experimental/celeval/pkg/client/listers/celeval/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	clientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"github.com/tektoncd/pipeline/pkg/client/injection/reconciler/pipeline/v1alpha1/run"
	listersalpha "github.com/tektoncd/pipeline/pkg/client/listers/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/reconciler/events"
	"go.uber.org/zap"
	"google.golang.org/genproto/googleapis/api/expr/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/reconciler"
	"reflect"
)

const (
	// CELEvalLabelKey is the label identifier for a CELEval, which is added to the Run
	CELEvalLabelKey = "/CELEval"

	// ManagedByLabelKey is the label identifier for CELEval, which is added to the Run
	ManagedByLabelKey = "app.kubernetes.io/managed-by"
)

// Reconciler implements controller.Reconciler for Run resources.
type Reconciler struct {
	pipelineClientSet clientset.Interface
	celEvalClientSet  celevalclientset.Interface
	runLister         listersalpha.RunLister
	celEvalLister     listersceleval.CELEvalLister
}

// Check that our Reconciler implements Interface
var _ run.Interface = (*Reconciler)(nil)

// ReconcileKind implements Interface.ReconcileKind.
func (r *Reconciler) ReconcileKind(ctx context.Context, run *v1alpha1.Run) reconciler.Event {
	var merr error
	logger := logging.FromContext(ctx)
	logger.With(zap.String("Run", fmt.Sprintf("%s/%s", run.Namespace, run.Name)))
	logger.Infof("Reconciling Run")

	if !run.HasStarted() {
		logger.Infof("Starting new Run")
		run.Status.InitializeConditions()
		// In case node time was not synchronized, when controller has been scheduled to other nodes.
		if run.Status.StartTime.Sub(run.CreationTimestamp.Time) < 0 {
			logger.Warnf("Run createTimestamp %s is after the Run started %s", run.CreationTimestamp, run.Status.StartTime)
			run.Status.StartTime = &run.CreationTimestamp
		}
		afterCondition := run.Status.GetCondition(apis.ConditionSucceeded)
		events.Emit(ctx, nil, afterCondition, run)
	}

	if run.IsDone() {
		logger.Infof("Run is done")
		return nil
	}

	beforeCondition := run.Status.GetCondition(apis.ConditionSucceeded)

	status := &celevalv1alpha1.CELEvalStatus{}
	if err := run.Status.DecodeExtraFields(status); err != nil {
		run.Status.MarkRunFailed(celevalv1alpha1.CELEvalRunReasonInternalError.String(),
			"Internal error calling DecodeExtraFields: %v", err)
		logger.Errorf("DecodeExtraFields error: %v", err.Error())
	}

	if err := r.reconcile(ctx, run, status); err != nil {
		logger.Errorf("Reconcile error: %v", err.Error())
		merr = multierror.Append(merr, err)
	}

	if err := r.updateLabelsAndAnnotations(ctx, run); err != nil {
		logger.Warn("Failed to update Run labels/annotations", zap.Error(err))
		merr = multierror.Append(merr, err)
	}

	if err := run.Status.EncodeExtraFields(status); err != nil {
		run.Status.MarkRunFailed(celevalv1alpha1.CELEvalRunReasonCouldntGetCELEval.String(),
			"Internal error calling EncodeExtraFields: %v", err)
		logger.Errorf("EncodeExtraFields error: %v", err.Error())
	}

	afterCondition := run.Status.GetCondition(apis.ConditionSucceeded)
	events.Emit(ctx, beforeCondition, afterCondition, run)

	// Only transient errors that should retry the reconcile are returned.
	return merr
}

func (c *Reconciler) reconcile(ctx context.Context, run *v1alpha1.Run, status *celevalv1alpha1.CELEvalStatus) error {
	logger := logging.FromContext(ctx)

	CELEvalMeta, CELEvalSpec, err := c.getCELEval(ctx, run)
	if err != nil {
		return nil
	}

	storeCELEvalSpec(status, CELEvalSpec)

	propagateCELEvalLabelsAndAnnotations(run, CELEvalMeta)

	if err := CELEvalSpec.Validate(); err != nil {
		logger.Errorf("Run is invalid because of %s", err)
		run.Status.MarkRunFailed(celevalv1alpha1.CELEvalRunReasonFailedValidation.String(),
			"Run can't be run because it has an invalid spec - %v", err)
		return nil
	}

	variablesMap := getVariablesMap(CELEvalSpec.Variables)
	// Create a program environment configured with the standard library of CEL functions and macros
	env, err := gocel.NewEnv(gocel.Declarations(getCELEvalEnvironmentDeclarations(variablesMap)...))
	if err != nil {
		logger.Errorf("Couldn't create a program env with standard library of CEL functions & macros when reconciling Run: %v", err)
		return err
	}

	for _, expression := range CELEvalSpec.Expressions {
		// Combine the Parse and Check phases CEL program compilation to produce an Ast and associated issues
		ast, iss := env.Compile(expression.Value.StringVal)
		if iss.Err() != nil {
			logger.Errorf("CEL expression %s could not be parsed when reconciling Run: %v", expression.Name, iss.Err())
			run.Status.MarkRunFailed(celevalv1alpha1.CELEvalRunReasonSyntaxError.String(),
				"CEL expression %s could not be parsed", expression.Name, iss.Err())
			return nil
		}

		// Generate an evaluable instance of the Ast within the environment
		prg, err := env.Program(ast)
		if err != nil {
			logger.Errorf("CEL expression %s could not be evaluated when reconciling Run: %v", expression.Name, err)
			run.Status.MarkRunFailed(celevalv1alpha1.CELEvalRunReasonEvaluationError.String(),
				"CEL expression %s could not be evaluated", expression.Name, err)
			return nil
		}

		// Evaluate the CEL expression (Ast)
		variablesMap := getVariablesMap(CELEvalSpec.Variables)
		out, _, err := prg.Eval(variablesMap)
		if err != nil {
			logger.Errorf("CEL expression %s could not be evaluated when reconciling Run: %v", expression.Name, err)
			run.Status.MarkRunFailed(celevalv1alpha1.CELEvalRunReasonEvaluationError.String(),
				"CEL expression %s could not be evaluated", expression.Name, err)
			return nil
		}

		// Evaluation of CEL expression was successful
		logger.Infof("CEL expression %s evaluated successfully when reconciling Run", expression.Name)
		status.Results = append(status.Results, v1alpha1.RunResult{
			Name:  expression.Name,
			Value: fmt.Sprintf("%s", out.ConvertToType(celtypes.StringType).Value()),
		})
	}

	// All CEL expressions were evaluated successfully
	run.Status.Results = append(run.Status.Results, status.Results...)
	run.Status.MarkRunSucceeded(celevalv1alpha1.CELEvalRunReasonEvaluationSuccess.String(),
		"CEL expressions were evaluated successfully")

	return nil
}

func (c *Reconciler) getCELEval(ctx context.Context, run *v1alpha1.Run) (*metav1.ObjectMeta, *celevalv1alpha1.CELEvalSpec, error) {
	if run.Spec.Ref == nil || run.Spec.Ref.Name == "" {
		// Run does not require name but for CELEval it does
		run.Status.MarkRunFailed(celevalv1alpha1.CELEvalRunReasonCouldntGetCELEval.String(),
			"Missing spec.ref.name for Run %s/%s",
			run.Namespace, run.Name)
		return nil, nil, fmt.Errorf("missing spec.ref.name for Run %s", fmt.Sprintf("%s/%s", run.Namespace, run.Name))
	}
	// Use the k8s client to get the CELEval rather than the lister.  This avoids a timing issue where
	// the CELEval is not yet in the lister cache if it is created at nearly the same time as the Run.
	// See https://github.com/tektoncd/pipeline/issues/2740 for discussion on this issue.
	cs, err := c.celEvalClientSet.CustomV1alpha1().CELEvals(run.Namespace).Get(ctx, run.Spec.Ref.Name, metav1.GetOptions{})
	if err != nil {
		run.Status.MarkRunFailed(celevalv1alpha1.CELEvalRunReasonCouldntGetCELEval.String(),
			"Error retrieving CELEval for Run %s/%s: %s",
			run.Namespace, run.Name, err)
		return nil, nil, fmt.Errorf("error retrieving CELEval for Run %s: %w", fmt.Sprintf("%s/%s", run.Namespace, run.Name), err)
	}

	return &cs.ObjectMeta, &cs.Spec, nil
}

func storeCELEvalSpec(status *celevalv1alpha1.CELEvalStatus, cs *celevalv1alpha1.CELEvalSpec) {
	// Only store the CELEvalSpec once, if it has never been set before
	if status.Spec == nil {
		status.Spec = cs
	}
}

func propagateCELEvalLabelsAndAnnotations(run *v1alpha1.Run, CELEvalMeta *metav1.ObjectMeta) {
	// Propagate labels from CELEval to Run
	if run.ObjectMeta.Labels == nil {
		run.ObjectMeta.Labels = make(map[string]string, len(CELEvalMeta.Labels)+1)
	}
	for key, value := range CELEvalMeta.Labels {
		run.ObjectMeta.Labels[key] = value
	}
	run.ObjectMeta.Labels[celeval.GroupName+CELEvalLabelKey] = CELEvalMeta.Name
	run.ObjectMeta.Labels[ManagedByLabelKey] = celeval.ControllerName

	// Propagate annotations from CELEval to Run
	if run.ObjectMeta.Annotations == nil {
		run.ObjectMeta.Annotations = make(map[string]string, len(CELEvalMeta.Annotations))
	}
	for key, value := range CELEvalMeta.Annotations {
		run.ObjectMeta.Annotations[key] = value
	}
}

func (r *Reconciler) updateLabelsAndAnnotations(ctx context.Context, run *v1alpha1.Run) error {
	newRun, err := r.runLister.Runs(run.Namespace).Get(run.Name)
	if err != nil {
		return fmt.Errorf("error getting Run %s when updating labels/annotations: %w", run.Name, err)
	}
	if reflect.DeepEqual(run.ObjectMeta.Labels, newRun.ObjectMeta.Labels) && reflect.DeepEqual(run.ObjectMeta.Annotations, newRun.ObjectMeta.Annotations) {
		return nil
	}

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

func getVariablesMap(variables []*v1beta1.Param) map[string]interface{} {
	variablesMap := make(map[string]interface{})
	for _, variable := range variables {
		variablesMap[variable.Name] = variable.Value.StringVal
	}
	return variablesMap
}

func getCELEvalEnvironmentDeclarations(variablesMap map[string]interface{}) []*expr.Decl {
	var CELEvalEnvironmentDeclarations []*expr.Decl
	for variableName := range variablesMap {
		CELEvalEnvironmentDeclarations = append(CELEvalEnvironmentDeclarations, decls.NewVar(variableName, decls.String))
	}
	return CELEvalEnvironmentDeclarations
}
