package v1alpha1

import (
	"errors"
	"fmt"

	"knative.dev/pkg/apis"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	duckv1 "knative.dev/pkg/apis/duck/v1"

	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

type RunDimensions struct {
	Resource  string
	Name      string
	Namespace string
	IsDeleted bool
	Status    duckv1.Status
	Labels    map[string]string
	Params    pipelinev1beta1.Params
	Object    runtime.Object
}

func (r *RunDimensions) GetId() string {
	return fmt.Sprintf("%s/%s/%s", r.Resource, r.Name, r.Namespace)
}

type MetricDimensionRef struct {
	Condition *string `json:"condition,omitempty"`
	Param     *string `json:"param,omitempty"`
	Label     *string `json:"label,omitempty"`
}

func (t *MetricDimensionRef) Key() (string, error) {
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

func (t *MetricDimensionRef) Value(runDimentions *RunDimensions) (string, error) {
	if t.Condition != nil {
		if *t.Condition == string(apis.ConditionSucceeded) {
			return statusCondition(runDimentions.Status.GetCondition(apis.ConditionSucceeded)), nil
		}
		return "INVALID", nil
	}

	if t.Label != nil {
		labelValue, exists := runDimentions.Labels[*t.Label]
		if !exists {
			return "MISSING", nil
		}
		return labelValue, nil
	}

	if t.Param != nil {
		for _, param := range runDimentions.Params {
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

type ByStatement struct {
	MetricDimensionRef `json:",inline"`
}

type MetricHistogramDuration struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type MetricGaugeMatch struct {
	Key      MetricDimensionRef           `json:"key"`
	Operator metav1.LabelSelectorOperator `json:"operator"`
	Values   []string                     `json:"values"`
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
	return "running"
}

// Metric represents the specification of a set of metrics.
type Metric struct {
	Type     string                   `json:"type"`
	Name     string                   `json:"name"`
	By       []ByStatement            `json:"by,omitempty"`
	Duration *MetricHistogramDuration `json:"duration,omitempty"`
	Match    *MetricGaugeMatch        `json:"match,omitempty"`
}
