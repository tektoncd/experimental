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
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	triggersv1beta1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

const (
	WorkflowLabelKey   = "workflows.tekton.dev/workflow"
	WorkflowsNamespace = "tekton-workflows"
)

// +genclient
// +genreconciler:krshapedlogic=false
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true

// Workflow represents a Workflow Custom Resource
type Workflow struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Spec holds the desired state of the Workflow from the client
	// +optional
	Spec WorkflowSpec `json:"spec,omitempty"`

	// +optional
	Status WorkflowStatus `json:"status,omitempty"`
}

// GetGroupVersionKind implements kmeta.OwnerRefable.
func (*Workflow) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("Workflow")
}

// WorkflowSpec describes the desired state of the Workflow
type WorkflowSpec struct {
	// Repos defines a set of Git repos required for this Workflow
	Repos []RepoRef `json:"repos,omitempty"`

	// Secrets defines any secrets that this Workflow needs
	// Secrets []Secret `json:"secrets,omitempty"`

	// Triggers is a list of triggers that can trigger this workflow
	Triggers []Trigger `json:"triggers,omitempty"`

	// Params define the default values for params in the Pipeline that can be
	// overridden in a WorkflowRun or (in the future) from an incoming event.
	Params []v1beta1.ParamSpec `json:"params,omitempty"`

	// PipelineSpec is an inline pipeline spec.
	// Cannot specify both PipelineSpec and PipelineRef.
	PipelineSpec *v1beta1.PipelineSpec `json:"pipelineSpec,omitempty"`

	// PipelineRef is a reference to a Pipeline
	PipelineRef *v1beta1.PipelineRef `json:"pipelineRef,omitempty"`

	// ServiceAccountName is the K8s service account that pipelineruns
	// generated from this workflow run as
	// +optional
	ServiceAccountName *string `json:"serviceAccountName,omitempty"`

	// Workspaces is a list of workspaces that the Pipeline in this workflow needs
	// TODO: Auto-setup a Workspace across multiple
	Workspaces []WorkflowWorkspaceBinding `json:"workspaces"`

	// TODO: Timeout ?
	Timeout *v1beta1.TimeoutFields `json:"timeout,omitempty"`
	// TODO: queue_ttl -> pending_timeout
}

// WorkflowStatus describes the observed state of the Workflow
type WorkflowStatus struct {
	duckv1.Status `json:",inline"`
}

type RepoRef struct {
	// TODO: Support cross-namespace repo references
	Name string `json:"name"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WorkflowList contains a list of Workflows
type WorkflowList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Workflow `json:"items"`
}

// WorkflowWorkspaceBinding maps a Pipeline's declared Workspaces
// to a Volume. Unlike a regular WorkspaceBinding, a WorkflowWorkspaceBinding
// will add additional magic to auto-propagate/generate PVCs
// TODO: Fluent Syntax for Binding
type WorkflowWorkspaceBinding struct {
	Name                     string `json:"name"`
	v1beta1.WorkspaceBinding `json:",inline"`
}

type Secret struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

type Trigger struct {
	// Event describes the incoming event for this Trigger
	Event *Event `json:"event"`

	// Bindings are the TriggerBindings used to extract information from this Trigger
	// +listType=atomic
	Bindings []*triggersv1beta1.TriggerSpecBinding `json:"bindings"`

	// Name is the name of this Trigger
	// +optional
	Name string `json:"name,omitempty"`

	// +optional
	Filters *Filters `json:"filters,omitempty"`
}

type Event struct {
	// Source defines the source of a trigger event
	Source EventSource `json:"source"`

	// Types is a slice of string that defines the type of an event (e.g. a pull_request or a push)
	// At the moment this assumes one of the GitHub event types
	Types []EventType `json:"types"`
}

type EventType string

const (
	EventTypePush        = EventType("push")
	EventTypePullRequest = EventType("pull_request")
)

// EventSource defines a Trigger EventSource
type EventSource struct {
	// TODO: support cron jobs
	// +optional
	Repo string `json:"repo,omitempty"`
}

type Filters struct {
	// GitRef filters events to those affecting the specified git branch or tag
	// Valid only for "pull_request" or "push" event types
	// For "pull_request" events, this is the base branch
	// +optional
	GitRef *GitRef `json:"gitRef,omitempty"`
	// +optional
	Custom []Custom `json:"custom,omitempty"`
}

type GitRef struct {
	// Regex matches a git branch or tag
	// +optional
	Regex string `json:"regex,omitempty"`
}

type Custom struct {
	// +optional
	CEL string `json:"cel,omitempty"`
}

var workflowCondSet = apis.NewBatchConditionSet()

// MarkFailed changes the Succeeded condition to False with the provided reason and message.
func (w *WorkflowStatus) MarkFailed(reason, messageFormat string, messageA ...interface{}) {
	workflowCondSet.Manage(w).MarkFalse(apis.ConditionSucceeded, reason, messageFormat, messageA...)
}

func GetEventTypes(ets []EventType) []string {
	var out []string
	for _, et := range ets {
		out = append(out, string(et))
	}
	return out
}
