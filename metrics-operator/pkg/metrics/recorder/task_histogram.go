package recorder

import (
	"context"

	"github.com/tektoncd/experimental/metrics-operator/pkg/apis/monitoring/v1alpha1"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"go.opencensus.io/stats"
)

type TaskHistogram struct {
	GenericRunHistogram
	TaskName string
}

func (t *TaskHistogram) Filter(run *v1alpha1.RunDimensions) bool {
	taskRun, ok := run.Object.(*pipelinev1beta1.TaskRun)
	if !ok {
		return false
	}
	ref := taskRun.Spec.TaskRef
	return ref != nil && ref.Name == t.TaskName
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
		TaskName:            monitor.Spec.TaskName,
	}
	return histogram
}
