package recorder

import (
	"context"

	"github.com/tektoncd/experimental/metrics-operator/pkg/apis/monitoring/v1alpha1"
	"go.opencensus.io/stats"
)

type TaskRunHistogram struct {
	GenericRunHistogram
	TaskRunFilter
}

func (t *TaskRunHistogram) Record(ctx context.Context, recorder stats.Recorder, run *v1alpha1.RunDimensions) {
	matched, err := t.Filter(run)
	if err != nil {
		return
	}
	if !matched {
		return
	}
	t.GenericRunHistogram.Record(ctx, recorder, run)
}

func NewTaskRunHistogram(metric *v1alpha1.Metric, monitor *v1alpha1.TaskRunMonitor) *TaskRunHistogram {
	generic := NewGenericRunHistogram(metric, "taskrun", monitor.Name)
	histogram := &TaskRunHistogram{
		GenericRunHistogram: *generic,
		TaskRunFilter: TaskRunFilter{
			Selector: monitor.Spec.Selector.DeepCopy(),
		},
	}
	return histogram
}
