package recorder

import (
	"context"

	"github.com/tektoncd/experimental/metrics-operator/pkg/apis/monitoring/v1alpha1"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"go.opencensus.io/stats"
)

type TaskHistogram struct {
	GenericTaskRunHistogram
	TaskName string
}

func (t *TaskHistogram) Filter(taskRun *pipelinev1beta1.TaskRun) bool {
	ref := taskRun.Spec.TaskRef
	return ref != nil && ref.Name == t.TaskName
}

func (t *TaskHistogram) Record(ctx context.Context, recorder stats.Recorder, taskRun *pipelinev1beta1.TaskRun) {
	if !t.Filter(taskRun) {
		return
	}
	t.GenericTaskRunHistogram.Record(ctx, recorder, taskRun)
}

func NewTaskHistogram(metric *v1alpha1.Metric, monitor *v1alpha1.TaskMonitor) *TaskHistogram {
	generic := NewGenericTaskRunHistogram(metric, "task", monitor.Name)
	histogram := &TaskHistogram{
		GenericTaskRunHistogram: *generic,
		TaskName:                monitor.Spec.TaskName,
	}
	return histogram
}
