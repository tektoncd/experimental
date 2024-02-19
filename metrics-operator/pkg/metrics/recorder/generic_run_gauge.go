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

type GenericRunGauge struct {
	Monitor   string
	Resource  string
	RunMetric *v1alpha1.Metric
	value     GaugeValue
	view      *view.View
	measure   *stats.Float64Measure
}

func (g *GenericRunGauge) Metric() *v1alpha1.Metric {
	return g.RunMetric
}
func (g *GenericRunGauge) MetricName() string {
	return naming.GaugeMetric(g.Resource, g.Monitor, g.RunMetric.Name)
}

func (g *GenericRunGauge) MonitorId() string {
	return naming.MonitorId(g.Resource, g.Monitor)
}

func (g *GenericRunGauge) View() *view.View {
	return g.view
}

func (g *GenericRunGauge) Record(ctx context.Context, recorder stats.Recorder, run *v1alpha1.RunDimensions) {
	logger := logging.FromContext(ctx)
	if g.RunMetric.Match != nil {
		matched, err := match(g.RunMetric.Match, run)
		if err != nil {
			logger.Errorf("skipping run, match failed: %w", err)
			g.Clean(ctx, recorder, run)
			return
		}
		if !matched {
			logger.Infof("skipping run, match is false")
			g.Clean(ctx, recorder, run)
			return
		}
	}

	if run.IsDeleted {
		logger.Infof("cleanup run, deleted")
		g.Clean(ctx, recorder, run)
		return
	}

	tagMap, err := tagMapFromByStatements(g.RunMetric.By, run)
	if err != nil {
		logger.Errorf("unable to render tag map for metric: %w", err)
		return
	}

	g.value.Update(run, tagMap)
	g.reportAll(ctx, recorder, run)
}

func (g *GenericRunGauge) reportAll(ctx context.Context, recorder stats.Recorder, run *v1alpha1.RunDimensions) {
	logger := logging.FromContext(ctx)
	for _, existingTagMap := range g.value.Keys() {
		gaugeMeasurement, err := g.value.ValueFor(existingTagMap)
		if err != nil {
			logger.Errorf("unable to render value for metric: %w", err)
			continue
		}
		recorder.Record(existingTagMap, []stats.Measurement{g.measure.M(float64(gaugeMeasurement))}, map[string]any{})
	}
}

func (g *GenericRunGauge) Clean(ctx context.Context, recorder stats.Recorder, run *v1alpha1.RunDimensions) {
	g.value.Delete(run)
	g.reportAll(ctx, recorder, run)
}

func NewGenericRunGauge(metric *v1alpha1.Metric, resource, monitorName string) *GenericRunGauge {
	gauge := &GenericRunGauge{
		Resource:  resource,
		Monitor:   monitorName,
		RunMetric: metric,
	}
	gauge.measure = stats.Float64(gauge.MetricName(), fmt.Sprintf("gauge samples for %s %s/%s", gauge.Resource, gauge.Monitor, gauge.RunMetric.Name), stats.UnitDimensionless)
	view := &view.View{
		Description: gauge.measure.Description(),
		Measure:     gauge.measure,
		Aggregation: view.LastValue(),
		TagKeys:     viewTags(metric.By),
	}
	gauge.view = view
	return gauge
}
