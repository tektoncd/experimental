package recorder

import (
	"context"

	"github.com/tektoncd/experimental/metrics-operator/pkg/apis/monitoring/v1alpha1"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"go.opencensus.io/stats"
)

type TaskCounter struct {
	GenericRunCounter
	TaskName string
}

// Filter returns true when the TaskRun should be recorded, independent of value
func (t *TaskCounter) Filter(run *v1alpha1.RunDimensions) bool {
	taskRun, ok := run.Object.(*pipelinev1beta1.TaskRun)
	if !ok {
		return false
	}
	ref := taskRun.Spec.TaskRef
	return ref != nil && ref.Name == t.TaskName
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
		TaskName:          monitor.Spec.TaskName,
	}
	return counter
}
