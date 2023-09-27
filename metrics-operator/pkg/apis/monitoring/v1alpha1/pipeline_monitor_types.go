package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

// +genclient
// +genreconciler:krshapedlogic=false
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PipelineMonitor ...
// +k8s:openapi-gen=true
type PipelineMonitor struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              PipelineMonitorSpec   `json:"spec"`
	Status            PipelineMonitorStatus `json:"status"`
}

// PipelineMonitorSpec ...
type PipelineMonitorSpec struct {
	PipelineName string   `json:"pipelineName"`
	Metrics      []Metric `json:"metrics"`
}

// PipelineMonitorStatus
type PipelineMonitorStatus struct {
	duckv1.Status `json:",inline"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PipelineMonitorList ...
type PipelineMonitorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PipelineMonitor `json:"items"`
}
