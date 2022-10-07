package v1alpha1

import (
	"context"
	"fmt"

	"knative.dev/pkg/apis"
)

var _ apis.Defaultable = (*Workflow)(nil)

func (w *Workflow) SetDefaults(ctx context.Context) {
	for i, t := range w.Spec.Triggers {
		if t.Name == "" {
			w.Spec.Triggers[i].Name = fmt.Sprintf("%d", i)
		}
	}
}
