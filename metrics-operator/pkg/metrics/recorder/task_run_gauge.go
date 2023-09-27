package recorder

import (
	"context"

	"github.com/tektoncd/experimental/metrics-operator/pkg/apis/monitoring/v1alpha1"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	pipelinev1beta1listers "github.com/tektoncd/pipeline/pkg/client/listers/pipeline/v1beta1"
	"go.opencensus.io/stats"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

type TaskRunGauge struct {
	GenericTaskRunGauge
	Selector *metav1.LabelSelector
}

// Filter returns true when the TaskRun should be recorded, independent of value
func (t *TaskRunGauge) Filter(taskRun *pipelinev1beta1.TaskRun) (bool, error) {
	if t.Selector == nil {
		return true, nil
	}
	selector, err := metav1.LabelSelectorAsSelector(t.Selector)
	if err != nil {
		return false, err
	}
	return selector.Matches(labels.Set(taskRun.Labels)), nil
}

func (t *TaskRunGauge) Record(ctx context.Context, recorder stats.Recorder, taskRun *pipelinev1beta1.TaskRun) {
	matched, err := t.Filter(taskRun)
	if err != nil {
		t.Clean(ctx, recorder, taskRun)
		return
	}
	if !matched {
		t.Clean(ctx, recorder, taskRun)
		return
	}
	t.GenericTaskRunGauge.Record(ctx, recorder, taskRun)
}

func NewTaskRunGauge(metric *v1alpha1.Metric, monitor *v1alpha1.TaskRunMonitor, taskRunLister pipelinev1beta1listers.TaskRunLister) *TaskRunGauge {
	gauge := &TaskRunGauge{
		GenericTaskRunGauge: *NewGenericTaskRunGauge(metric, "taskrun", monitor.Name, taskRunLister),
		Selector:            monitor.Spec.Selector.DeepCopy(),
	}
	return gauge
}
