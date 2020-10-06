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

package v1alpha1

import (
	"context"
	"strings"

	"github.com/tektoncd/pipeline/pkg/apis/validate"
	"k8s.io/apimachinery/pkg/util/validation"
	"knative.dev/pkg/apis"
)

var _ apis.Validatable = (*TaskLoop)(nil)

// Validate TaskLoop
func (tl *TaskLoop) Validate(ctx context.Context) *apis.FieldError {
	if err := validate.ObjectMetadata(tl.GetObjectMeta()); err != nil {
		return err.ViaField("metadata")
	}
	return tl.Spec.Validate(ctx)
}

// Validate TaskLoopSpec
func (tls *TaskLoopSpec) Validate(ctx context.Context) *apis.FieldError {
	// Validate Task reference or inline task spec.
	if err := validateTask(ctx, tls); err != nil {
		return err
	}
	return nil
}

func validateTask(ctx context.Context, tls *TaskLoopSpec) *apis.FieldError {
	// taskRef and taskSpec are mutually exclusive.
	if (tls.TaskRef != nil && tls.TaskRef.Name != "") && tls.TaskSpec != nil {
		return apis.ErrMultipleOneOf("spec.taskRef", "spec.taskSpec")
	}
	// Check that one of taskRef and taskSpec is present.
	if (tls.TaskRef == nil || tls.TaskRef.Name == "") && tls.TaskSpec == nil {
		return apis.ErrMissingOneOf("spec.taskRef", "spec.taskSpec")
	}
	// Validate TaskSpec if it's present
	if tls.TaskSpec != nil {
		if err := tls.TaskSpec.Validate(ctx); err != nil {
			return err.ViaField("spec.taskSpec")
		}
	}
	if tls.TaskRef != nil && tls.TaskRef.Name != "" {
		// taskRef name must be a valid k8s name
		if errSlice := validation.IsQualifiedName(tls.TaskRef.Name); len(errSlice) != 0 {
			return apis.ErrInvalidValue(strings.Join(errSlice, ","), "spec.taskRef.name")
		}
	}
	return nil
}
