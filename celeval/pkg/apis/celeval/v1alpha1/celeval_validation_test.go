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

package v1alpha1_test

import (
	"context"
	celevalv1alpha1 "github.com/tektoncd/experimental/celeval/pkg/apis/celeval/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/tektoncd/pipeline/test/diff"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
)

func Test_CELEval_Validate_Valid(t *testing.T) {
	tests := []struct {
		name    string
		celEval *celevalv1alpha1.CELEval
	}{{
		name: "expressions only",
		celEval: &celevalv1alpha1.CELEval{
			ObjectMeta: metav1.ObjectMeta{Name: "celevaleval"},
			Spec: celevalv1alpha1.CELEvalSpec{
				Expressions: []*v1beta1.Param{{
					Name: "expr1",
					Value: v1beta1.ArrayOrString{
						StringVal: "foo",
					},
				}},
			},
		},
	}, {
		name: "expressions and variables",
		celEval: &celevalv1alpha1.CELEval{
			ObjectMeta: metav1.ObjectMeta{Name: "celeval"},
			Spec: celevalv1alpha1.CELEvalSpec{
				Expressions: []*v1beta1.Param{{
					Name: "expr1",
					Value: v1beta1.ArrayOrString{
						StringVal: "foo",
					},
				}},
				Variables: []*v1beta1.Param{{
					Name: "var1",
					Value: v1beta1.ArrayOrString{
						StringVal: "bar",
					},
				}},
			},
		},
	}}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.celEval.Validate(context.Background())
			if err != nil {
				t.Errorf("Unexpected error for %s: %s", tc.name, err)
			}
		})
	}
}

func Test_CELEval_Validate_Invalid(t *testing.T) {
	tests := []struct {
		name          string
		celEval       *celevalv1alpha1.CELEval
		expectedError apis.FieldError
	}{{
		name: "no expressions",
		celEval: &celevalv1alpha1.CELEval{
			ObjectMeta: metav1.ObjectMeta{Name: "celeval"},
			Spec:       celevalv1alpha1.CELEvalSpec{},
		},
		expectedError: apis.FieldError{
			Message: "missing field(s)",
			Paths:   []string{"expressions"},
		},
	}, {
		name: "array expressions",
		celEval: &celevalv1alpha1.CELEval{
			ObjectMeta: metav1.ObjectMeta{Name: "celeval"},
			Spec: celevalv1alpha1.CELEvalSpec{
				Expressions: []*v1beta1.Param{{
					Name: "expr1",
					Value: v1beta1.ArrayOrString{
						ArrayVal: []string{"foo", "bar"},
					},
				}},
			},
		},
		expectedError: apis.FieldError{
			Message: "invalid value: CEL expression expr1 must be a string",
			Paths:   []string{"expressions[expr1].value"},
		},
	}, {
		name: "array variables",
		celEval: &celevalv1alpha1.CELEval{
			ObjectMeta: metav1.ObjectMeta{Name: "celeval"},
			Spec: celevalv1alpha1.CELEvalSpec{
				Expressions: []*v1beta1.Param{{
					Name: "expr1",
					Value: v1beta1.ArrayOrString{
						StringVal: "foo",
					},
				}},
				Variables: []*v1beta1.Param{{
					Name: "var1",
					Value: v1beta1.ArrayOrString{
						ArrayVal: []string{"foo", "bar"},
					},
				}},
			},
		},
		expectedError: apis.FieldError{
			Message: "invalid value: CEL environment variable var1 must be a string",
			Paths:   []string{"variables[var1].value"},
		},
	}}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.celEval.Validate(context.Background())
			if err == nil {
				t.Errorf("Expected an Error but did not get one for %s", tc.name)
			} else {
				if d := cmp.Diff(tc.expectedError.Error(), err.Error(), cmpopts.IgnoreUnexported(apis.FieldError{})); d != "" {
					t.Errorf("Error is different from expected for %s. diff %s", tc.name, diff.PrintWantGot(d))
				}
			}
		})
	}
}
