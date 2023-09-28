package recorder

import (
	"context"

	"github.com/tektoncd/experimental/metrics-operator/pkg/apis/monitoring/v1alpha1"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"go.opencensus.io/stats"
)

type TaskGauge struct {
	GenericRunGauge
	TaskName string
}

// Filter returns true when the TaskRun should be recorded, independent of value
func (t *TaskGauge) Filter(run *v1alpha1.RunDimensions) bool {
	taskRun, ok := run.Object.(*pipelinev1beta1.TaskRun)
	if !ok {
		return false
	}
	ref := taskRun.Spec.TaskRef
	return ref != nil && ref.Name == t.TaskName
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
		TaskName:        monitor.Spec.TaskName,
	}
	return gauge
}
