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
	TaskName string       `json:"taskName"`
	Metrics  []TaskMetric `json:"metrics"`
}

type TaskMetric struct {
	Type     string                       `json:"type"`
	Name     string                       `json:"name"`
	By       []TaskByStatement            `json:"by,omitempty"`
	Duration *TaskMetricHistogramDuration `json:"duration,omitempty"`
	Match    *TaskMetricGaugeMatch        `json:"match,omitempty"`
}

type TaskRunValueRef struct {
	Condition *string `json:"condition,omitempty"`
	Param     *string `json:"param,omitempty"`
	Label     *string `json:"label,omitempty"`
}

type TaskByStatement struct {
	TaskRunValueRef `json:",inline"`
}

type TaskMetricHistogramDuration struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type TaskMetricGaugeMatch struct {
	Key      TaskRunValueRef              `json:"key"`
	Operator metav1.LabelSelectorOperator `json:"operator"`
	Values   []string                     `json:"values"`
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
