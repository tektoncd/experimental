package recorder

import (
	"context"

	"github.com/tektoncd/experimental/metrics-operator/pkg/apis/monitoring/v1alpha1"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	pipelinev1beta1listers "github.com/tektoncd/pipeline/pkg/client/listers/pipeline/v1beta1"
	"go.opencensus.io/stats"
)

type TaskGauge struct {
	GenericTaskRunGauge
	TaskName string
}

// Filter returns true when the TaskRun should be recorded, independent of value
func (t *TaskGauge) Filter(taskRun *pipelinev1beta1.TaskRun) bool {
	ref := taskRun.Spec.TaskRef
	return ref != nil && ref.Name == t.TaskName
}

func (t *TaskGauge) Record(ctx context.Context, recorder stats.Recorder, taskRun *pipelinev1beta1.TaskRun) {
	if !t.Filter(taskRun) {
		t.Clean(ctx, recorder, taskRun)
		return
	}
	t.GenericTaskRunGauge.Record(ctx, recorder, taskRun)
}

func NewTaskGauge(metric *v1alpha1.TaskMetric, monitor *v1alpha1.TaskMonitor, taskRunLister pipelinev1beta1listers.TaskRunLister) *TaskGauge {
	gauge := &TaskGauge{
		GenericTaskRunGauge: *NewGenericTaskRunGauge(metric, "taskrun", monitor.Name, taskRunLister),
		TaskName:            monitor.Spec.TaskName,
	}
	return gauge
}
