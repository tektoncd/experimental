package v1alpha1

import (
	"context"

	"github.com/tektoncd/pipeline/pkg/apis/validate"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/webhook/resourcesemantics"
)

var _ apis.Validatable = (*Workflow)(nil)
var _ resourcesemantics.VerbLimited = (*Workflow)(nil)

// SupportedVerbs returns the operations that validation should be called for
func (w *Workflow) SupportedVerbs() []admissionregistrationv1.OperationType {
	return []admissionregistrationv1.OperationType{admissionregistrationv1.Create, admissionregistrationv1.Update}
}

// Validate performs validation of the metadata and spec of this ClusterTask.
func (w *Workflow) Validate(ctx context.Context) *apis.FieldError {
	errs := validate.ObjectMetadata(w.GetObjectMeta()).ViaField("metadata")
	return errs.Also(w.Spec.Validate(apis.WithinSpec(ctx)).ViaField("spec"))
}

func (s *WorkflowSpec) Validate(ctx context.Context) (errs *apis.FieldError) {
	return nil
}
