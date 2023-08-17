package v1alpha1

import (
	"errors"

	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
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

func (t *TaskRunValueRef) Key() (string, error) {
	if t.Condition != nil {
		if *t.Condition == string(apis.ConditionSucceeded) {
			return "status", nil
		}
		if *t.Condition == string(apis.ConditionReady) {
			return "ready", nil
		}
		return "", errors.New("invalid")
	}
	// TODO: sanatize string
	if t.Param != nil {
		return *t.Param, nil
	}
	// TODO: sanatize string
	if t.Label != nil {
		return *t.Label, nil
	}
	return "", errors.New("invalid")
}

func statusCondition(cond *apis.Condition) string {
	if cond == nil {
		return ""
	}
	if cond.IsTrue() {
		return "success"
	}
	if cond.IsFalse() {
		return "failed"
	}
	return "unknown"
}

// TODO: decouple from tekton.dev/v1beta1
func (t *TaskRunValueRef) Value(taskRun *pipelinev1beta1.TaskRun) (string, error) {
	if t.Condition != nil {
		if *t.Condition == string(apis.ConditionSucceeded) {
			return statusCondition(taskRun.Status.GetCondition(apis.ConditionSucceeded)), nil
		}
		return "INVALID", nil
	}

	if t.Label != nil {
		labelValue, exists := taskRun.Labels[*t.Label]
		if !exists {
			return "MISSING", nil
		}
		return labelValue, nil
	}

	if t.Param != nil {
		for _, param := range taskRun.Spec.Params {
			if param.Name == *t.Param {
				// TODO: support array and objects
				if param.Value.StringVal != "" {
					return param.Value.StringVal, nil
				}
				return "UNSUPPORTED_VALUE", nil
			}
		}
		return "MISSING", nil
	}
	return "", errors.New("invalid value")
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
