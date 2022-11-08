package v1alpha1

import (
	"context"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/webhook/resourcesemantics"
)

var _ apis.Validatable = (*GitRepository)(nil)
var _ resourcesemantics.VerbLimited = (*GitRepository)(nil)
var _ apis.Defaultable = (*GitRepository)(nil)

// +genclient
// +genreconciler:krshapedlogic=false
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true

// GitRepository represents a connection to a Git Repository
type GitRepository struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// +optional
	Spec RepoSpec `json:"spec,omitempty"`
	// +optional
	Status RepoStatus `json:"status,omitempty"`
}

type RepoSpec struct {
	URL string `json:"url"`
	// A reference to a secret containing a GitHub personal access token
	// with permission to create webhooks
	// +optional
	AccessToken Secret `json:"accessToken"`
	// A reference to a secret containing a random string to secure a webhook
	WebhookSecret Secret `json:"webhookSecret"`
	// +optional
	EventTypes []string `json:"eventTypes,omitempty"`
}

type RepoStatus struct {
	duckv1.Status `json:",inline"`
}

// GetGroupVersionKind implements kmeta.OwnerRefable.
func (*GitRepository) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("GitRepository")
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GitRepositoryList contains a list of  GitRepositories
type GitRepositoryList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GitRepository `json:"items"`
}

// SupportedVerbs returns the operations that validation should be called for
func (g *GitRepository) SupportedVerbs() []admissionregistrationv1.OperationType {
	return []admissionregistrationv1.OperationType{admissionregistrationv1.Create, admissionregistrationv1.Update}
}

// Validate performs validation of the metadata and spec of this ClusterTask.
func (g *GitRepository) Validate(ctx context.Context) *apis.FieldError {
	return nil
}

func (g *GitRepository) SetDefaults(ctx context.Context) {}
