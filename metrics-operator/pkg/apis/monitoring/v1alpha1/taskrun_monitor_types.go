package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

// +genclient
// +genreconciler:krshapedlogic=false
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TaskRunMonitor ...
// +k8s:openapi-gen=true
type TaskRunMonitor struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              TaskRunMonitorSpec   `json:"spec"`
	Status            TaskRunMonitorStatus `json:"status"`
}

// TaskMonitorSpec ...
type TaskRunMonitorSpec struct {
	Selector metav1.LabelSelector `json:"selector"`
	Metrics  []TaskMetric         `json:"metrics"`
}

// TaskMonitorStatus
type TaskRunMonitorStatus struct {
	duckv1.Status `json:",inline"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TaskMonitorList ...
type TaskRunMonitorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TaskMonitor `json:"items"`
}
