package v1alpha1

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/kmeta"
)

const (
	ManagedByLabelKey = "app.kubernetes.io/managed-by"

	// Label used to indicate that a reconciler should start a pending PipelineRun
	LabelToStartPR = "tekton.dev/ok-to-start"
)

type Strategy string

var (
	StrategyCancel                      = Strategy("Cancel")
	StrategyGracefullyCancel            = Strategy("GracefullyCancel")
	StrategyGracefullyStop              = Strategy("GracefullyStop")
	supportedStrategies      []Strategy = []Strategy{StrategyCancel, StrategyGracefullyCancel, StrategyGracefullyStop}
)

// +genclient
// +genreconciler:krshapedlogic=false
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
type ConcurrencyControl struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// +optional
	Spec ConcurrencySpec `json:"spec"`
}

var _ kmeta.OwnerRefable = (*ConcurrencyControl)(nil)
var _ apis.Validatable = (*ConcurrencyControl)(nil)
var _ apis.Defaultable = (*ConcurrencyControl)(nil)

type ConcurrencySpec struct {
	// + optional
	Strategy string `json:"strategy,omitempty"`
	// + optional
	Selector metav1.LabelSelector `json:"selector,omitempty"`
	// Label keys that identify the concurrency group of a PipelineRun.
	// All PipelineRuns with the same value for all of these labels are part of the
	// same concurrency group.
	// If a PipelineRun has no value for a key in GroupBy, other PipelineRuns that
	// also have no value for that key will be part of the same concurrency group.
	// + optional
	GroupBy []string `json:"groupBy,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ConcurrencyControlList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ConcurrencyControl `json:"items"`
}

// SetDefaults sets the defaults on the object.
func (t *ConcurrencyControl) SetDefaults(ctx context.Context) {
	if t.Spec.Strategy == "" {
		t.Spec.Strategy = string(StrategyGracefullyCancel)
	}
}

// Validate validates a concurrencycontrol
func (t *ConcurrencyControl) Validate(ctx context.Context) *apis.FieldError {
	return validateStrategy(t.Spec.Strategy)
}

func validateStrategy(s string) *apis.FieldError {
	for _, supported := range supportedStrategies {
		if s == string(supported) {
			return nil
		}
	}
	return apis.ErrInvalidValue(fmt.Sprintf("got unsupported strategy %s", s), "strategy")
}

// GetGroupVersionKind implements kmeta.OwnerRefable
func (cc *ConcurrencyControl) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("ConcurrencyControl")
}
