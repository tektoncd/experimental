package v1alpha1

import (
	"context"
	"fmt"
	"strings"

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

var StrategyCancel = Strategy("cancel")

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
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ConcurrencyControlList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ConcurrencyControl `json:"items"`
}

// SetDefaults sets the defaults on the object.
func (t *ConcurrencyControl) SetDefaults(ctx context.Context) {}

// Validate validates a concurrencycontrol
func (t *ConcurrencyControl) Validate(ctx context.Context) *apis.FieldError {
	if strings.ToLower(t.Spec.Strategy) != string(StrategyCancel) {
		return apis.ErrInvalidValue(fmt.Sprintf("got strategy %s but the only supported strategy is 'Cancel'", t.Spec.Strategy), "strategy")
	}
	return nil
}

// GetGroupVersionKind implements kmeta.OwnerRefable
func (cc *ConcurrencyControl) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("ConcurrencyControl")
}
