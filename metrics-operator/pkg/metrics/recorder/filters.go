package recorder

import (
	"fmt"

	"github.com/tektoncd/experimental/metrics-operator/pkg/apis/monitoring/v1alpha1"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

type PipelineFilter struct {
	PipelineName string
}

// Filter returns true when the PipelineRun should be recorded, independent of value
func (p *PipelineFilter) Filter(run *v1alpha1.RunDimensions) bool {
	pipelineRun, ok := run.Object.(*pipelinev1beta1.PipelineRun)
	if !ok {
		return false
	}
	ref := pipelineRun.Spec.PipelineRef
	return ref != nil && ref.Name == p.PipelineName
}

type PipelineRunFilter struct {
	Selector *metav1.LabelSelector
}

// Filter returns true when the PipelineRun should be recorded, independent of value
func (p *PipelineRunFilter) Filter(run *v1alpha1.RunDimensions) (bool, error) {
	pipelineRun, ok := run.Object.(*pipelinev1beta1.PipelineRun)
	if !ok {
		return false, fmt.Errorf("expected PipelineRun, but got %T", run.Object)
	}
	if p.Selector == nil {
		return true, nil
	}
	selector, err := metav1.LabelSelectorAsSelector(p.Selector)
	if err != nil {
		return false, err
	}
	return selector.Matches(labels.Set(pipelineRun.Labels)), nil
}

type TaskFilter struct {
	TaskName string
}

// Filter returns true when the TaskRun should be recorded, independent of value
func (t *TaskFilter) Filter(run *v1alpha1.RunDimensions) bool {
	taskRun, ok := run.Object.(*pipelinev1beta1.TaskRun)
	if !ok {
		return false
	}
	ref := taskRun.Spec.TaskRef
	return ref != nil && ref.Name == t.TaskName
}

type TaskRunFilter struct {
	Selector *metav1.LabelSelector
}

// Filter returns true when the TaskRun should be recorded, independent of value
func (t *TaskRunFilter) Filter(run *v1alpha1.RunDimensions) (bool, error) {
	taskRun, ok := run.Object.(*pipelinev1beta1.TaskRun)
	if !ok {
		return false, fmt.Errorf("expected taskRun, but got %T", run.Object)
	}
	if t.Selector == nil {
		return true, nil
	}
	selector, err := metav1.LabelSelectorAsSelector(t.Selector)
	if err != nil {
		return false, err
	}
	return selector.Matches(labels.Set(taskRun.Labels)), nil
}
