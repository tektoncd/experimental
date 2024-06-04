package recorder

import (
	"context"

	"github.com/tektoncd/experimental/metrics-operator/pkg/apis/monitoring/v1alpha1"
	"go.opencensus.io/stats"
)

type TaskRunGauge struct {
	GenericRunGauge
	TaskRunFilter
}

func (t *TaskRunGauge) Record(ctx context.Context, recorder stats.Recorder, run *v1alpha1.RunDimensions) {
	matched, err := t.Filter(run)
	if err != nil {
		t.Clean(ctx, recorder, run)
		return
	}
	if !matched {
		t.Clean(ctx, recorder, run)
		return
	}
	t.GenericRunGauge.Record(ctx, recorder, run)
}

func NewTaskRunGauge(metric *v1alpha1.Metric, monitor *v1alpha1.TaskRunMonitor) *TaskRunGauge {
	gauge := &TaskRunGauge{
		GenericRunGauge: *NewGenericRunGauge(metric, "taskrun", monitor.Name),
		TaskRunFilter: TaskRunFilter{
			Selector: monitor.Spec.Selector.DeepCopy(),
		},
	}
	return gauge
}
