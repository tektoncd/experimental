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

package v1alpha1

import (
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/apis/run/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CELEval evaluates a CELEval expression with given environment variables
// +k8s:openapi-gen=true
type CELEval struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata"`

	// Spec holds the desired state of the CELEval from the client
	// +optional
	Spec CELEvalSpec `json:"spec"`
}

// CELEvalSpec defines the desired state of the CELEval
type CELEvalSpec struct {
	// Variables to be configured in the CELEval environment before evaluation
	// +optional
	Variables []*v1beta1.Param `json:"variables,omitempty"`

	// Expressions are a list of CELEval expressions to be evaluated given the environment Variables
	Expressions []*v1beta1.Param `json:"expressions"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CELEvalList contains a list of CELEvals
type CELEvalList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CELEval `json:"items"`
}

// CELEvalRunReason represents a reason for the Run "Succeeded" condition
type CELEvalRunReason string

const (
	// CELEvalRunReasonFailedValidation indicates that the reason for failure status is that CELEval failed validation
	CELEvalRunReasonFailedValidation CELEvalRunReason = "RunValidationFailed"

	// CELEvalRunReasonSyntaxError indicates that the reason for failure status is that a CELEval expression couldn't be parsed
	CELEvalRunReasonSyntaxError CELEvalRunReason = "SyntaxError"

	// CELEvalRunReasonEvaluationError indicates that the reason for failure status is that a CELEval expression couldn't be
	// evaluated typically due to evaluation environment or executable program
	CELEvalRunReasonEvaluationError CELEvalRunReason = "EvaluationError"

	// CELEvalRunReasonEvaluationSuccess indicates that the reason for the success status is that all CELEval expressions were
	// evaluated successfully and the evaluation results were produced
	CELEvalRunReasonEvaluationSuccess CELEvalRunReason = "EvaluationSuccess"

	// CELEvalEvalRunReasonCouldntGetCELEval indicates that the reason for the failure status is that the Run could not find the CELEval
	CELEvalRunReasonCouldntGetCELEval CELEvalRunReason = "CouldntGetCELEval"

	// CELEvalRunReasonInternalError indicates that the CELEval failed due to an internal error in the reconciler
	CELEvalRunReasonInternalError CELEvalRunReason = "InternalError"
)

func (t CELEvalRunReason) String() string {
	return string(t)
}

type CELEvalStatus struct {
	// Spec contains the exact spec used to instantiate the Run
	Spec *CELEvalSpec `json:"spec,omitempty"`

	// Results contains the results from evaluating the CELEval expressions given the environment Variables
	Results []v1alpha1.RunResult `json:"results,omitempty"`
}
