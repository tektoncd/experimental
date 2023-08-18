package recorder

import (
	"context"

	"github.com/tektoncd/experimental/metrics-operator/pkg/apis/monitoring/v1alpha1"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"go.opencensus.io/stats"
)

type TaskCounter struct {
	GenericTaskRunCounter
	TaskName string
}

// Filter returns true when the TaskRun should be recorded, independent of value
func (t *TaskCounter) Filter(taskRun *pipelinev1beta1.TaskRun) bool {
	ref := taskRun.Spec.TaskRef
	return ref != nil && ref.Name == t.TaskName
}

func (t *TaskCounter) Record(ctx context.Context, recorder stats.Recorder, taskRun *pipelinev1beta1.TaskRun) {
	if !t.Filter(taskRun) {
		return
	}
	t.GenericTaskRunCounter.Record(ctx, recorder, taskRun)
}

func NewTaskCounter(metric *v1alpha1.TaskMetric, monitor *v1alpha1.TaskMonitor) *TaskCounter {
	generic := NewGenericTaskRunCounter(metric, "task", monitor.Name)
	counter := &TaskCounter{
		GenericTaskRunCounter: *generic,
		TaskName:              monitor.Spec.TaskName,
	}
	return counter
}
