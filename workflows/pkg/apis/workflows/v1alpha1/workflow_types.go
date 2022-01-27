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
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	triggersv1beta1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
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

// WorkflowSpec describes the desired state of the Workflow
type WorkflowSpec struct {
	// Repos defines a set of Git repos required for this Workflow
	// Repos   []Repo   `json:"repos,omitempty"`

	// Secrets defines any secrets that this Workflow needs
	// Secrets []Secret `json:"secrets,omitempty"`

	// Triggers is a list of triggers that can trigger this workflow
	Triggers []Trigger `json:"triggers,omitempty"`

	// Params define the default values for params in the Pipeline that can be
	// overridden in a WorkflowRun or (in the future) from an incoming event.
	Params []pipelinev1beta1.ParamSpec `json:"params,omitempty"`

	// Pipeline is a reference to a Pipeline. Currently only an inline
	// pipelineSpec is supported
	Pipeline PipelineRef `json:"pipeline,omitempty"`

	// ServiceAccountName is the K8s service account that pipelineruns
	// generated from this workflow run as
	// +optional
	ServiceAccountName *string `json:"serviceAccountName,omitempty"`

	// Workspaces is a list of workspaces that the Pipeline in this workflow needs
	// TODO: Auto-setup a Workspace across multiple
	Workspaces []WorkflowWorkspaceBinding `json:"workspaces"`

	// TODO: Timeout ?
	Timeout *pipelinev1beta1.TimeoutFields `json:"timeout,omitempty"`
	// TODO: queue_ttl -> pending_timeout
}

// WorkflowStatus describes the observed state of the Workflow
type WorkflowStatus struct {
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WorkflowList contains a list of Workflows
type WorkflowList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Workflow `json:"items"`
}

// PipelineRef describes a pipeline
// Only one of the following must be provided
// TODO: Add validation
type PipelineRef struct {
	// Spec is a PipelineSpec that defines the pipeline to be run for this Workflow
	// Mutually exclusive with Git
	Spec pipelinev1beta1.PipelineSpec `json:"spec,omitempty"`

	// Git defines the location of a Tekton Pipeline inside a git repository
	// Mutually exclusive with Spec
	Git PipelineRefGit `json:"git,omitempty"`
}

// PipelineRefGit refers to the location of a pipeline within a git repository at a particular commit/revision
type PipelineRefGit struct {
	// URL is the URL of the git repo containing the pipeline
	URL string `json:"url"`

	// Revision is the git revision to fetch the pipeline from
	Revision string `json:"revision"`

	// PathInRepo is the path to the pipeline file in the git repo at the revision
	PathInRepo string `json:"pathInRepo"`
}

// WorkflowWorkspaceBinding maps a Pipeline's declared Workspaces
// to a Volume. Unlike a regular WorkspaceBinding, a WorkflowWorkspaceBinding
// will add additional magic to auto-propagate/generate PVCs
// TODO: Fluent Syntax for Binding
type WorkflowWorkspaceBinding struct {
	Name                             string `json:"name"`
	pipelinev1beta1.WorkspaceBinding `json:",inline"`
}

type Secret struct {
	Name string `json:"name"`
	Ref  string `json:"ref"`
}

type Trigger struct {
	// Event describes the incoming event for this Trigger
	Event Event `json:"event"`

	// Bindings are the TriggerBindings used to extract information from this Trigger
	// +listType=atomic
	Bindings []*triggersv1beta1.TriggerSpecBinding `json:"bindings"`

	// Name is the name of this Trigger
	// +optional
	Name string `json:"name,omitempty"`

	// Interceptors are used to define additional filters on this trigger
	// This field is temporary till we decide on how exactly to support simplified filters
	// +listType=atomic
	Interceptors []*triggersv1beta1.TriggerInterceptor `json:"interceptors,omitempty"`
}

type Event struct {
	// Source defines the source of a trigger event
	Source EventSource `json:"source"`

	// Type is a string that defines the type of an event (e.g. a pull_request or a push)
	// At the moment this assumes one of the GitHub event types
	Type string `json:"type"`

	// Secret is the Webhook secret used for this Trigger
	// This field is temporary until we implement a better way to handle secrets for webhook validation
	Secret triggersv1beta1.SecretRef `json:"secret"`
}

// EventSource defines a Trigger EventSource
type EventSource struct {
	// TBD, this struct should contain enough information to identify the source of the events
	// To start with, we'd support push and pull request events from GitHub as well as Cron/scheduled events

}
