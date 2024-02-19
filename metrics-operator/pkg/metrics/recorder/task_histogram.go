package recorder

import (
	"context"

	"github.com/tektoncd/experimental/metrics-operator/pkg/apis/monitoring/v1alpha1"
	"go.opencensus.io/stats"
)

type TaskHistogram struct {
	GenericRunHistogram
	TaskFilter
}

func (t *TaskHistogram) Record(ctx context.Context, recorder stats.Recorder, run *v1alpha1.RunDimensions) {
	if !t.Filter(run) {
		return
	}
	t.GenericRunHistogram.Record(ctx, recorder, run)
}

func NewTaskHistogram(metric *v1alpha1.Metric, monitor *v1alpha1.TaskMonitor) *TaskHistogram {
	generic := NewGenericRunHistogram(metric, "task", monitor.Name)
	histogram := &TaskHistogram{
		GenericRunHistogram: *generic,
		TaskFilter: TaskFilter{
			TaskName: monitor.Spec.TaskName,
		},
	}
	return histogram
}
