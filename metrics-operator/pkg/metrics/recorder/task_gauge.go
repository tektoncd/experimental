package recorder

import (
	"context"

	"github.com/tektoncd/experimental/metrics-operator/pkg/apis/monitoring/v1alpha1"
	"go.opencensus.io/stats"
)

type TaskGauge struct {
	GenericRunGauge
	TaskFilter
}

func (t *TaskGauge) Record(ctx context.Context, recorder stats.Recorder, run *v1alpha1.RunDimensions) {
	if !t.Filter(run) {
		t.Clean(ctx, recorder, run)
		return
	}
	t.GenericRunGauge.Record(ctx, recorder, run)
}

func NewTaskGauge(metric *v1alpha1.Metric, monitor *v1alpha1.TaskMonitor) *TaskGauge {
	gauge := &TaskGauge{
		GenericRunGauge: *NewGenericRunGauge(metric, "task", monitor.Name),
		TaskFilter: TaskFilter{
			TaskName: monitor.Spec.TaskName,
		},
	}
	return gauge
}
