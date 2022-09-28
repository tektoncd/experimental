package v1alpha1

import (
	"context"

	"github.com/tektoncd/pipeline/pkg/apis/validate"
	"knative.dev/pkg/apis"
)

var _ apis.Validatable = (*Workflow)(nil)

// Validate performs validation of the metadata and spec of this ClusterTask.
func (t *Workflow) Validate(ctx context.Context) *apis.FieldError {
	errs := validate.ObjectMetadata(t.GetObjectMeta()).ViaField("metadata")
	if apis.IsInDelete(ctx) {
		return nil
	}
	return errs.Also(t.Spec.Validate(apis.WithinSpec(ctx)).ViaField("spec"))
}

func (s *WorkflowSpec) Validate(ctx context.Context) (errs *apis.FieldError) {
	return nil
}
