package v1alpha1

import (
	"context"

	"github.com/knative/pkg/apis"
)

func (ps *EventBindingSpec) Validate(ctx context.Context) *apis.FieldError {
	return nil
}

func (ps *EventBinding) Validate(ctx context.Context) *apis.FieldError {
	return nil
}
