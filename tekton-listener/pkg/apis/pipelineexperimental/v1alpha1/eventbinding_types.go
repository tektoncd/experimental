/*
Copyright 2018 The Knative Authors

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
	"context"

	duckv1alpha1 "github.com/knative/pkg/apis/duck/v1alpha1"
	pipelinev1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EventBindingSpec defines the desired state of the EventBinding
// +k8s:deepcopy-gen=true
type EventBindingSpec struct {
	// The tekton pipeline this will reference
	PipelineRef pipelinev1alpha1.PipelineRef `json:"pipelineRef"`
	// The source we are creating this binding to handle
	SourceRef SourceRef `json:"sourceref"`
	// The resources that will be created/deleted for this binding
	ResourceTemplates []pipelinev1alpha1.PipelineResource `json:"resourceTemplates"`
	// The resources to bind the PipelineRun to
	Resources []pipelinev1alpha1.PipelineResourceBinding `json:"resources"`
	// Params is a list of parameter names and values for use with the PipelineRun
	Params []pipelinev1alpha1.Param `json:"params"`
	// Time after which the Pipeline times out. Defaults to never
	Timeout *metav1.Duration `json:"timeout,omitempty"`
	// Reference to the specific type of event we want to handle
	EventRef EventRef `json:"eventref,omitempty"`
	// +optional
	ServiceAccount string `json:"serviceAccount"`
}

type SourceRef struct {
	Name       string `json:"name,omitempty"`
	APIVersion string `json:"apiversion,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// EventBinding binds an event source to a pipeline in order to produce PipelineRuns
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=true
type EventBinding struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// +optional
	Spec EventBindingSpec `json:"spec,omitempty"`
	// +optional
	Status EventBindingStatus `json:"status,omitempty"`
}

// EventBindingSpecStatus defines the pipelinerun spec status the user can provide
type EventBindingSpecStatus string

// EventBindingStatus defines the observed state of the EventBinding
// +k8s:deepcopy-gen=true
type EventBindingStatus struct {
	duckv1alpha1.Status `json:",inline"`
	// namespace of the listener
	Namespace string `json:"namespace"`
	// name of the tekton listeners resource
	ListenerName string `json:"listenername"`
	// The listener is addressable
	// +optional
	Address duckv1alpha1.Addressable `json:"address,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// EventBindingList contains a list of EventBindings
// +k8s:deepcopy-gen=true
type EventBindingList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EventBinding `json:"items"`
}

type EventRef struct {
	EventName  string `json:"eventname,inline"`
	EventType  string `json:"eventtype,inline"`
	APIVersion string `json:"apiversion,omitempty"`
}

// SetDefaults for pipelinerun
func (e *EventBinding) SetDefaults(ctx context.Context) {}
