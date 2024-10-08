package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

// +genclient
// +genreconciler:krshapedlogic=false
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TaskMonitor ...
// +k8s:openapi-gen=true
type TaskMonitor struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              TaskMonitorSpec   `json:"spec"`
	Status            TaskMonitorStatus `json:"status"`
}

// TaskMonitorSpec ...
type TaskMonitorSpec struct {
	TaskName string   `json:"taskName"`
	Metrics  []Metric `json:"metrics"`
}

// TaskMonitorStatus
type TaskMonitorStatus struct {
	duckv1.Status `json:",inline"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TaskMonitorList ...
type TaskMonitorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TaskMonitor `json:"items"`
}
