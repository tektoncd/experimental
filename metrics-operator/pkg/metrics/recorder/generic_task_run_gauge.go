package recorder

import (
	"context"
	"fmt"

	"github.com/tektoncd/experimental/metrics-operator/pkg/apis/monitoring/v1alpha1"
	"github.com/tektoncd/experimental/metrics-operator/pkg/naming"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	pipelinev1beta1listers "github.com/tektoncd/pipeline/pkg/client/listers/pipeline/v1beta1"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"k8s.io/apimachinery/pkg/labels"
	"knative.dev/pkg/logging"
)

type GenericTaskRunGauge struct {
	TaskName      string
	Monitor       string
	Resource      string
	TaskRunLister pipelinev1beta1listers.TaskRunLister
	TaskMetric    *v1alpha1.TaskMetric
	value         GaugeValue
	view          *view.View
	measure       *stats.Float64Measure
}

func (g *GenericTaskRunGauge) MetricName() string {
	return naming.GaugeMetric(g.Resource, g.Monitor, g.TaskMetric.Name)
}

func (g *GenericTaskRunGauge) MetricType() string {
	return "gauge"
}

func (g *GenericTaskRunGauge) MonitorName() string {
	return g.Monitor
}

func (g *GenericTaskRunGauge) View() *view.View {
	return g.view
}

// Filter returns true when the TaskRun should be recorded, independent of value
func (g *GenericTaskRunGauge) Filter(taskRun *pipelinev1beta1.TaskRun) bool {
	ref := taskRun.Spec.TaskRef
	return ref != nil && ref.Name == g.TaskName
}

func (g *GenericTaskRunGauge) syncGauge(ctx context.Context, recorder stats.Recorder, taskRun *pipelinev1beta1.TaskRun) {
	logger := logging.FromContext(ctx)
	allTaskRuns, err := g.TaskRunLister.List(labels.Everything())
	if err != nil {
		logger.Errorf("error listing taskruns", "kind", "TaskMonitor", "monitor", g.Monitor, "metric", g.TaskMetric)
		return
	}

	gaugeBy := map[string]int{}
	gaugeMap := map[string]*tag.Map{}
	for _, tr := range allTaskRuns {
		if !g.Filter(tr) {
			continue
		}
		if g.TaskMetric.Match != nil {
			matched, err := match(g.TaskMetric.Match, tr)
			if err != nil {
				logger.Errorf("skipping taskrun, match failed: %w", err)
				continue
			}
			if !matched {
				continue
			}
		}
		tagMap, err := tagMapFromByStatements(g.TaskMetric.By, tr)
		if err != nil {
			logger.Errorf("could not render labels for gauge metric: %w", err)
			continue
		}
		gaugeBy[tagMap.String()] += 1
		gaugeMap[tagMap.String()] = tagMap
	}

	for key, tagMap := range gaugeMap {
		gaugeValue := gaugeBy[key]
		recorder.Record(tagMap, []stats.Measurement{g.measure.M(float64(gaugeValue))}, map[string]any{})
	}
}

func (g *GenericTaskRunGauge) Record(ctx context.Context, recorder stats.Recorder, taskRun *pipelinev1beta1.TaskRun) {
	logger := logging.FromContext(ctx)
	if !g.Filter(taskRun) {
		g.Clean(ctx, recorder, taskRun)
		return
	}
	if g.TaskMetric.Match != nil {
		matched, err := match(g.TaskMetric.Match, taskRun)
		if err != nil {
			logger.Errorf("skipping taskrun, match failed: %w", err)
			g.Clean(ctx, recorder, taskRun)
			return
		}
		if !matched {
			logger.Infof("skipping taskrun, match is false")
			g.Clean(ctx, recorder, taskRun)
			return
		}
	}

	if taskRun.DeletionTimestamp != nil {
		logger.Infof("cleanup taskrun, deleted")
		g.Clean(ctx, recorder, taskRun)
		return
	}

	tagMap, err := tagMapFromByStatements(g.TaskMetric.By, taskRun)
	if err != nil {
		logger.Errorf("unable to render tag map for metric: %w", err)
		return
	}

	g.value.Update(taskRun, tagMap)
	g.reportAll(ctx, recorder, taskRun)
}

func (g *GenericTaskRunGauge) reportAll(ctx context.Context, recorder stats.Recorder, taskRun *pipelinev1beta1.TaskRun) {
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

func (g *GenericTaskRunGauge) Clean(ctx context.Context, recorder stats.Recorder, taskRun *pipelinev1beta1.TaskRun) {
	g.value.Delete(taskRun)
	g.reportAll(ctx, recorder, taskRun)
}

func NewGenericTaskRunGauge(metric *v1alpha1.TaskMetric, resource, monitorName string, taskRunLister pipelinev1beta1listers.TaskRunLister) *GenericTaskRunGauge {
	gauge := &GenericTaskRunGauge{
		Resource:      resource,
		Monitor:       monitorName,
		TaskMetric:    metric,
		TaskRunLister: taskRunLister,
	}
	gauge.measure = stats.Float64(gauge.MetricName(), fmt.Sprintf("gauge samples for TaskMonitor %s/%s", gauge.Monitor, gauge.TaskMetric.Name), stats.UnitDimensionless)
	view := &view.View{
		Description: gauge.measure.Description(),
		Measure:     gauge.measure,
		Aggregation: view.LastValue(),
		TagKeys:     viewTags(metric.By),
	}
	gauge.view = view
	return gauge
}
