package recorder

import (
	"testing"

	"github.com/tektoncd/experimental/metrics-operator/pkg/apis/monitoring/v1alpha1"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

func TestGaugeValue(t *testing.T) {
	v := &GaugeValue{}
	metric := &v1alpha1.TaskMetric{
		Type: "gauge",
		Name: "status",
		By: []v1alpha1.TaskByStatement{
			{
				TaskRunValueRef: v1alpha1.TaskRunValueRef{
					Condition: pointer.String("Succeeded"),
				},
			},
			{
				TaskRunValueRef: v1alpha1.TaskRunValueRef{
					Label: pointer.String("repository"),
				},
			},
		},
	}

	taskRun := &pipelinev1beta1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: "hello-world",
			Labels: map[string]string{
				"repository": "repo0",
			},
		},
		Status: pipelinev1beta1.TaskRunStatus{
			Status: duckv1.Status{
				Conditions: duckv1.Conditions{
					{
						Type:   "Succeeded",
						Status: corev1.ConditionUnknown,
					},
				},
			},
		},
	}
	tagMap, err := tagMapFromByStatements(metric.By, taskRun)
	if err != nil {
		t.Fatal(err)
	}
	var gauge float64
	v.Update(taskRun, tagMap)

	gauge, err = v.ValueFor(tagMap)
	if err != nil {
		t.Fatal(err)
	}
	if gauge != 1. {
		t.Errorf("Expected 1, got %f", gauge)
	}

	taskRun2 := taskRun.DeepCopy()
	taskRun2.Name = "hello-world-run2"
	v.Update(taskRun2, tagMap)

	gauge, err = v.ValueFor(tagMap)
	if err != nil {
		t.Fatal(err)
	}
	if gauge != 2. {
		t.Errorf("Expected 2, got %f", gauge)
	}

	v.Delete(taskRun2)

	gauge, err = v.ValueFor(tagMap)
	if err != nil {
		t.Fatal(err)
	}
	if gauge != 1. {
		t.Errorf("Expected 1, got %f", gauge)
	}

	taskRun.Labels = map[string]string{
		"repository": "repo2",
	}
	tagMap2, err := tagMapFromByStatements(metric.By, taskRun)
	if err != nil {
		t.Fatal(err)
	}

	v.Update(taskRun, tagMap2)

	gauge, err = v.ValueFor(tagMap2)
	if err != nil {
		t.Fatal(err)
	}
	if gauge != 1. {
		t.Errorf("Expected 1, got %f", gauge)
	}
	gauge, err = v.ValueFor(tagMap)
	if err != nil {
		t.Fatal(err)
	}
	if gauge != 0. {
		t.Errorf("Expected 0, got %f", gauge)
	}
}
