package recorder

import (
	"context"
	"fmt"

	"github.com/tektoncd/experimental/metrics-operator/pkg/apis/monitoring/v1alpha1"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"go.opencensus.io/stats"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

type TaskRunHistogram struct {
	GenericTaskRunHistogram
	Selector *metav1.LabelSelector
}

func (t *TaskRunHistogram) Filter(run *v1alpha1.RunDimensions) (bool, error) {
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

func (t *TaskRunHistogram) Record(ctx context.Context, recorder stats.Recorder, run *v1alpha1.RunDimensions) {
	matched, err := t.Filter(run)
	if err != nil {
		return
	}
	if !matched {
		return
	}
	t.GenericTaskRunHistogram.Record(ctx, recorder, run)
}

func NewTaskRunHistogram(metric *v1alpha1.Metric, monitor *v1alpha1.TaskRunMonitor) *TaskRunHistogram {
	generic := NewGenericTaskRunHistogram(metric, "taskrun", monitor.Name)
	histogram := &TaskRunHistogram{
		GenericTaskRunHistogram: *generic,
		Selector:                monitor.Spec.Selector.DeepCopy(),
	}
	return histogram
}
