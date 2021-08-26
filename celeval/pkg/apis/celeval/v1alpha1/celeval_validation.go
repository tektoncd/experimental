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
	"context"
	"fmt"
	"github.com/tektoncd/pipeline/pkg/apis/validate"
	"knative.dev/pkg/apis"
)

var _ apis.Validatable = (*CELEval)(nil)

func (c *CELEval) Validate(ctx context.Context) *apis.FieldError {
	if err := validate.ObjectMetadata(c.GetObjectMeta()); err != nil {
		return err.ViaField("metadata")
	}
	return c.Spec.Validate()
}

func (cs *CELEvalSpec) Validate() *apis.FieldError {
	if err := validateCELEval(cs); err != nil {
		return err
	}
	return nil
}

func validateCELEval(cs *CELEvalSpec) (errs *apis.FieldError) {
	errs = errs.Also(validateExpressionsProvided(cs))
	errs = errs.Also(validateExpressionsType(cs))
	errs = errs.Also(validateVariablesType(cs))
	return errs
}

func validateExpressionsProvided(cs *CELEvalSpec) (errs *apis.FieldError) {
	if len(cs.Expressions) == 0 {
		errs = errs.Also(apis.ErrMissingField("expressions"))
	}
	return errs
}

func validateExpressionsType(cs *CELEvalSpec) (errs *apis.FieldError) {
	for _, expression := range cs.Expressions {
		if expression.Value.StringVal == "" {
			errs = errs.Also(apis.ErrInvalidValue(fmt.Sprintf("CEL expression %s must be a string", expression.Name),
				"value").ViaFieldKey("expressions", expression.Name))
		}
	}
	return errs
}

func validateVariablesType(cs *CELEvalSpec) (errs *apis.FieldError) {
	for _, variable := range cs.Variables {
		if variable.Value.StringVal == "" {
			errs = errs.Also(apis.ErrInvalidValue(fmt.Sprintf("CEL environment variable %s must be a string", variable.Name),
				"value").ViaFieldKey("variables", variable.Name))
		}
	}
	return errs
}
