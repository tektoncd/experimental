package recorder

import (
	"context"
	"fmt"

	"github.com/tektoncd/experimental/metrics-operator/pkg/apis/monitoring/v1alpha1"
	"github.com/tektoncd/experimental/metrics-operator/pkg/naming"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"knative.dev/pkg/logging"
)

type TaskCounter struct {
	TaskName        string
	TaskMonitorName string
	TaskMetric      *v1alpha1.TaskMetric
	view            *view.View
	measure         *stats.Float64Measure
}

func (t *TaskCounter) MetricName() string {
	return naming.CounterMetric("task", t.TaskMonitorName, t.TaskMetric.Name)
}

func (t *TaskCounter) MonitorName() string {
	return t.TaskMonitorName
}

func (t *TaskCounter) View() *view.View {
	return t.view
}

func (t *TaskCounter) Record(ctx context.Context, recorder stats.Recorder, taskRun *pipelinev1beta1.TaskRun) {
	logger := logging.FromContext(ctx)
	tagMap, err := tagMapFromByStatements(t.TaskMetric.By, taskRun)
	if err != nil {
		logger.Errorw("error recording value", "kind", "TaskMonitor", "monitor", t.TaskMonitorName, "metric", t.TaskMetric)
		return
	}
	recorder.Record(tagMap, []stats.Measurement{t.measure.M(1)}, map[string]any{})
}

func NewTaskCounter(metric *v1alpha1.TaskMetric, monitor *v1alpha1.TaskMonitor) *TaskCounter {
	counter := &TaskCounter{
		TaskName:        monitor.Spec.TaskName,
		TaskMonitorName: monitor.Name,
		TaskMetric:      metric,
	}
	counter.measure = stats.Float64(counter.MetricName(), fmt.Sprintf("count samples for TaskMonitor %s/%s", counter.TaskMonitorName, counter.TaskMetric.Name), stats.UnitDimensionless)
	view := &view.View{
		Description: counter.measure.Description(),
		Measure:     counter.measure,
		Aggregation: view.Count(),
		TagKeys:     viewTags(metric.By),
	}
	counter.view = view
	return counter
}
