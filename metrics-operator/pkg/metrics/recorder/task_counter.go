package recorder

import (
	"context"

	"github.com/tektoncd/experimental/metrics-operator/pkg/apis/monitoring/v1alpha1"
	"go.opencensus.io/stats"
)

type TaskCounter struct {
	GenericRunCounter
	TaskFilter
}

func (t *TaskCounter) Record(ctx context.Context, recorder stats.Recorder, run *v1alpha1.RunDimensions) {
	if !t.Filter(run) {
		return
	}
	t.GenericRunCounter.Record(ctx, recorder, run)
}

func NewTaskCounter(metric *v1alpha1.Metric, monitor *v1alpha1.TaskMonitor) *TaskCounter {
	generic := NewGenericRunCounter(metric, "task", monitor.Name)
	counter := &TaskCounter{
		GenericRunCounter: *generic,
		TaskFilter: TaskFilter{
			TaskName: monitor.Spec.TaskName,
		},
	}
	return counter
}
