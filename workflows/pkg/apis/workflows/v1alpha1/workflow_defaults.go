package v1alpha1

import (
	"context"

	"knative.dev/pkg/apis"
)

var _ apis.Defaultable = (*Workflow)(nil)

func (w *Workflow) SetDefaults(ctx context.Context) {}
