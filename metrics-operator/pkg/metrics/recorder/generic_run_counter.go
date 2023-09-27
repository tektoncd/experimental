package recorder

import (
	"context"
	"fmt"

	"github.com/tektoncd/experimental/metrics-operator/pkg/apis/monitoring/v1alpha1"
	"github.com/tektoncd/experimental/metrics-operator/pkg/naming"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"knative.dev/pkg/logging"
)

type GenericRunCounter struct {
	Resource  string
	Monitor   string
	RunMetric *v1alpha1.Metric
	view      *view.View
	measure   *stats.Float64Measure
}

func (g *GenericRunCounter) Metric() *v1alpha1.Metric {
	return g.RunMetric
}

func (t *GenericRunCounter) MetricName() string {
	return naming.CounterMetric(t.Resource, t.Monitor, t.RunMetric.Name)
}

func (t *GenericRunCounter) MonitorName() string {
	return t.Monitor
}

func (t *GenericRunCounter) View() *view.View {
	return t.view
}

func (t *GenericRunCounter) Record(ctx context.Context, recorder stats.Recorder, run *v1alpha1.RunDimensions) {
	logger := logging.FromContext(ctx)
	tagMap, err := tagMapFromByStatements(t.RunMetric.By, run)
	if err != nil {
		logger.Errorw("error recording value", "resource", t.Resource, "monitor", t.MonitorName(), "metric", t.RunMetric)
		return
	}
	recorder.Record(tagMap, []stats.Measurement{t.measure.M(1)}, map[string]any{})
}

func (t *GenericRunCounter) Clean(ctx context.Context, recorder stats.Recorder, run *v1alpha1.RunDimensions) {
}

func NewGenericRunCounter(metric *v1alpha1.Metric, resource, monitorName string) *GenericRunCounter {
	counter := &GenericRunCounter{
		Resource:  resource,
		Monitor:   monitorName,
		RunMetric: metric,
	}
	counter.measure = stats.Float64(counter.MetricName(), fmt.Sprintf("count samples for %s %s/%s", counter.Resource, counter.Monitor, counter.RunMetric.Name), stats.UnitDimensionless)
	view := &view.View{
		Description: counter.measure.Description(),
		Measure:     counter.measure,
		Aggregation: view.Count(),
		TagKeys:     viewTags(metric.By),
	}
	counter.view = view
	return counter
}
