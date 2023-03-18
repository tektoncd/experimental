package recorder

import (
	"context"
	"fmt"

	"github.com/tektoncd/experimental/metrics-operator/pkg/apis/monitoring/v1alpha1"
	"github.com/tektoncd/experimental/metrics-operator/pkg/naming"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
)

type taskCounter struct {
	taskName        string
	taskMonitorName string
	taskMetric      *v1alpha1.TaskMetric
	view            *view.View
	measure         *stats.Float64Measure
}

func (t *taskCounter) MetricName() string {
	return naming.CounterMetric("task", t.taskMonitorName, t.taskMetric.Name)
}

func (t *taskCounter) MonitorName() string {
	return t.taskMonitorName
}

func (t *taskCounter) View() *view.View {
	return t.view
}

func (t *taskCounter) Record(ctx context.Context, recorder stats.Recorder, taskRun *pipelinev1beta1.TaskRun) {
	tagMap, err := tagMapFromByStatements(t.taskMetric.By, taskRun)
	if err != nil {
		return
	}
	recorder.Record(tagMap, []stats.Measurement{t.measure.M(1)}, map[string]any{})
}

func NewTaskCounter(metric *v1alpha1.TaskMetric, monitor *v1alpha1.TaskMonitor) *taskCounter {
	counter := &taskCounter{
		taskName:        monitor.Spec.TaskName,
		taskMonitorName: monitor.Name,
		taskMetric:      metric,
	}
	counter.measure = stats.Float64(counter.MetricName(), fmt.Sprintf("count samples for TaskMonitor %s/%s", counter.taskMonitorName, counter.taskMetric.Name), stats.UnitDimensionless)
	view := &view.View{
		Description: counter.measure.Description(),
		Measure:     counter.measure,
		Aggregation: view.Count(),
		TagKeys:     viewTags(metric.By),
	}
	counter.view = view
	return counter
}
