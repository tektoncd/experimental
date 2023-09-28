package recorder

import (
	"context"

	"github.com/tektoncd/experimental/metrics-operator/pkg/apis/monitoring/v1alpha1"
	"go.opencensus.io/stats"
	"knative.dev/pkg/logging"
)

type TaskRunCounter struct {
	GenericRunCounter
	TaskRunFilter
}

func (t *TaskRunCounter) Record(ctx context.Context, recorder stats.Recorder, run *v1alpha1.RunDimensions) {
	logger := logging.FromContext(ctx)
	matched, err := t.Filter(run)
	if err != nil {
		logger.Errorf("could not record metric")
		return
	}
	if !matched {
		return
	}
	t.GenericRunCounter.Record(ctx, recorder, run)
}

func NewTaskRunCounter(metric *v1alpha1.Metric, monitor *v1alpha1.TaskRunMonitor) *TaskRunCounter {
	generic := NewGenericRunCounter(metric, "taskrun", monitor.Name)
	counter := &TaskRunCounter{
		GenericRunCounter: *generic,
		TaskRunFilter: TaskRunFilter{
			Selector: monitor.Spec.Selector.DeepCopy(),
		},
	}
	return counter
}
