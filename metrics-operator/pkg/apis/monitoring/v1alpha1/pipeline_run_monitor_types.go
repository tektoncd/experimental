package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

// +genclient
// +genreconciler:krshapedlogic=false
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PipelineRunMonitor ...
// +k8s:openapi-gen=true
type PipelineRunMonitor struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              PipelineRunMonitorSpec   `json:"spec"`
	Status            PipelineRunMonitorStatus `json:"status"`
}

// PipelineRunMonitorSpec ...
type PipelineRunMonitorSpec struct {
	Selector metav1.LabelSelector `json:"selector"`
	Metrics  []Metric             `json:"metrics"`
}

// PipelineRunMonitorStatus
type PipelineRunMonitorStatus struct {
	duckv1.Status `json:",inline"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PipelineRunMonitorList ...
type PipelineRunMonitorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PipelineRunMonitor `json:"items"`
}
