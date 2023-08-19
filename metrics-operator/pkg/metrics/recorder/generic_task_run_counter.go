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

type GenericTaskRunCounter struct {
	Resource   string
	Monitor    string
	TaskMetric *v1alpha1.TaskMetric
	view       *view.View
	measure    *stats.Float64Measure
}

func (g *GenericTaskRunCounter) Metric() *v1alpha1.TaskMetric {
	return g.TaskMetric
}

func (t *GenericTaskRunCounter) MetricName() string {
	return naming.CounterMetric(t.Resource, t.Monitor, t.TaskMetric.Name)
}

func (t *GenericTaskRunCounter) MonitorName() string {
	return t.Monitor
}

func (t *GenericTaskRunCounter) View() *view.View {
	return t.view
}

func (t *GenericTaskRunCounter) Record(ctx context.Context, recorder stats.Recorder, taskRun *pipelinev1beta1.TaskRun) {
	logger := logging.FromContext(ctx)
	tagMap, err := tagMapFromByStatements(t.TaskMetric.By, taskRun)
	if err != nil {
		logger.Errorw("error recording value", "resource", t.Resource, "monitor", t.MonitorName(), "metric", t.TaskMetric)
		return
	}
	recorder.Record(tagMap, []stats.Measurement{t.measure.M(1)}, map[string]any{})
}

func (t *GenericTaskRunCounter) Clean(ctx context.Context, recorder stats.Recorder, taskRun *pipelinev1beta1.TaskRun) {
}

func NewGenericTaskRunCounter(metric *v1alpha1.TaskMetric, resource, monitorName string) *GenericTaskRunCounter {
	counter := &GenericTaskRunCounter{
		Resource:   resource,
		Monitor:    monitorName,
		TaskMetric: metric,
	}
	counter.measure = stats.Float64(counter.MetricName(), fmt.Sprintf("count samples for TaskMonitor %s/%s", counter.Monitor, counter.TaskMetric.Name), stats.UnitDimensionless)
	view := &view.View{
		Description: counter.measure.Description(),
		Measure:     counter.measure,
		Aggregation: view.Count(),
		TagKeys:     viewTags(metric.By),
	}
	counter.view = view
	return counter
}
