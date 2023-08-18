package recorder

import (
	"context"
	"fmt"
	"sync"

	"github.com/tektoncd/experimental/metrics-operator/pkg/apis/monitoring/v1alpha1"
	"github.com/tektoncd/experimental/metrics-operator/pkg/naming"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	pipelinev1beta1listers "github.com/tektoncd/pipeline/pkg/client/listers/pipeline/v1beta1"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/sets"
	"knative.dev/pkg/logging"
)

type TaskGauge struct {
	TaskName        string
	TaskMonitorName string
	TaskRunLister   pipelinev1beta1listers.TaskRunLister
	TaskMetric      *v1alpha1.TaskMetric
	value           GaugeValue
	view            *view.View
	measure         *stats.Float64Measure
}

type GaugeTagMapValue struct {
	tagMap       *tag.Map
	taskRunNames sets.Set[string]
}

type GaugeValue struct {
	m  map[string]GaugeTagMapValue
	rw sync.RWMutex
}

func (g *GaugeValue) ValueFor(tagMap *tag.Map) (float64, error) {
	g.rw.Lock()
	defer g.rw.Unlock()
	if g.m == nil {
		g.m = map[string]GaugeTagMapValue{}
	}

	tagMapValue, exists := g.m[tagMap.String()]
	if !exists {
		return 0.0, fmt.Errorf("tag does not exist")
	}

	return float64(tagMapValue.taskRunNames.Len()), nil
}

func (g *GaugeValue) Keys() []*tag.Map {
	g.rw.Lock()
	defer g.rw.Unlock()
	if g.m == nil {
		g.m = map[string]GaugeTagMapValue{}
	}

	result := []*tag.Map{}
	for _, tagMapValue := range g.m {
		result = append(result, tagMapValue.tagMap)
	}
	return result
}

func (g *GaugeValue) deleteTaskRunFromAllTagMapValues(t *pipelinev1beta1.TaskRun, exceptions sets.Set[string]) {
	for key, tagMapValue := range g.m {
		if tagMapValue.taskRunNames.Has(t.Name) && !exceptions.Has(key) {
			tagMapValue.taskRunNames = tagMapValue.taskRunNames.Delete(t.Name)
		}
	}
}

func (g *GaugeValue) Delete(t *pipelinev1beta1.TaskRun) {
	g.rw.Lock()
	defer g.rw.Unlock()
	if g.m == nil {
		g.m = map[string]GaugeTagMapValue{}
	}

	g.deleteTaskRunFromAllTagMapValues(t, sets.New[string]())
}

func (g *GaugeValue) Update(t *pipelinev1beta1.TaskRun, tagMap *tag.Map) {
	g.rw.Lock()
	defer g.rw.Unlock()
	if g.m == nil {
		g.m = map[string]GaugeTagMapValue{}
	}

	g.deleteTaskRunFromAllTagMapValues(t, sets.New[string](tagMap.String()))

	// TODO: namespace
	tagMapValue, exists := g.m[tagMap.String()]
	if !exists {
		g.m[tagMap.String()] = GaugeTagMapValue{
			tagMap:       tagMap,
			taskRunNames: sets.New[string](t.Name),
		}
	} else {
		tagMapValue.taskRunNames = tagMapValue.taskRunNames.Insert(t.Name)
	}

}

func (t *TaskGauge) MetricName() string {
	return naming.GaugeMetric("task", t.TaskMonitorName, t.TaskMetric.Name)
}

func (t *TaskGauge) MetricType() string {
	return "gauge"
}

func (t *TaskGauge) MonitorName() string {
	return t.TaskMonitorName
}

func (t *TaskGauge) View() *view.View {
	return t.view
}

// Filter returns true when the TaskRun should be recorded, independent of value
func (t *TaskGauge) Filter(taskRun *pipelinev1beta1.TaskRun) bool {
	ref := taskRun.Spec.TaskRef
	return ref != nil && ref.Name == t.TaskName
}

func (t *TaskGauge) syncGauge(ctx context.Context, recorder stats.Recorder, taskRun *pipelinev1beta1.TaskRun) {
	logger := logging.FromContext(ctx)
	allTaskRuns, err := t.TaskRunLister.List(labels.Everything())
	if err != nil {
		logger.Errorf("error listing taskruns", "kind", "TaskMonitor", "monitor", t.TaskMonitorName, "metric", t.TaskMetric)
		return
	}

	gaugeBy := map[string]int{}
	gaugeMap := map[string]*tag.Map{}
	for _, tr := range allTaskRuns {
		if !t.Filter(tr) {
			continue
		}
		if t.TaskMetric.Match != nil {
			matched, err := match(t.TaskMetric.Match, tr)
			if err != nil {
				logger.Errorf("skipping taskrun, match failed: %w", err)
				continue
			}
			if !matched {
				continue
			}
		}
		tagMap, err := tagMapFromByStatements(t.TaskMetric.By, tr)
		if err != nil {
			logger.Errorf("could not render labels for gauge metric: %w", err)
			continue
		}
		gaugeBy[tagMap.String()] += 1
		gaugeMap[tagMap.String()] = tagMap
	}

	for key, tagMap := range gaugeMap {
		gaugeValue := gaugeBy[key]
		recorder.Record(tagMap, []stats.Measurement{t.measure.M(float64(gaugeValue))}, map[string]any{})
	}
}

func (t *TaskGauge) Record(ctx context.Context, recorder stats.Recorder, taskRun *pipelinev1beta1.TaskRun) {
	logger := logging.FromContext(ctx)
	if !t.Filter(taskRun) {
		t.Clean(ctx, recorder, taskRun)
		return
	}
	if t.TaskMetric.Match != nil {
		matched, err := match(t.TaskMetric.Match, taskRun)
		if err != nil {
			logger.Errorf("skipping taskrun, match failed: %w", err)
			t.Clean(ctx, recorder, taskRun)
			return
		}
		if !matched {
			logger.Infof("skipping taskrun, match is false")
			t.Clean(ctx, recorder, taskRun)
			return
		}
	}

	if taskRun.DeletionTimestamp != nil {
		t.Clean(ctx, recorder, taskRun)
		return
	}
	tagMap, err := tagMapFromByStatements(t.TaskMetric.By, taskRun)
	if err != nil {
		logger.Errorf("unable to render tag map for metric: %w", err)
		return
	}
	t.value.Update(taskRun, tagMap)

	t.reportAll(ctx, recorder, taskRun)
}

func (t *TaskGauge) reportAll(ctx context.Context, recorder stats.Recorder, taskRun *pipelinev1beta1.TaskRun) {
	logger := logging.FromContext(ctx)
	for _, existingTagMap := range t.value.Keys() {
		gaugeMeasurement, err := t.value.ValueFor(existingTagMap)
		if err != nil {
			logger.Errorf("unable to render value for metric: %w", err)
			continue
		}
		recorder.Record(existingTagMap, []stats.Measurement{t.measure.M(float64(gaugeMeasurement))}, map[string]any{})
	}
}

func (t *TaskGauge) Clean(ctx context.Context, recorder stats.Recorder, taskRun *pipelinev1beta1.TaskRun) {
	t.value.Delete(taskRun)
	t.reportAll(ctx, recorder, taskRun)
}

func NewTaskGauge(metric *v1alpha1.TaskMetric, monitor *v1alpha1.TaskMonitor, taskRunLister pipelinev1beta1listers.TaskRunLister) *TaskGauge {
	gauge := &TaskGauge{
		TaskName:        monitor.Spec.TaskName,
		TaskMonitorName: monitor.Name,
		TaskMetric:      metric,
		TaskRunLister:   taskRunLister,
	}
	gauge.measure = stats.Float64(gauge.MetricName(), fmt.Sprintf("gauge samples for TaskMonitor %s/%s", gauge.TaskMonitorName, gauge.TaskMetric.Name), stats.UnitDimensionless)
	view := &view.View{
		Description: gauge.measure.Description(),
		Measure:     gauge.measure,
		Aggregation: view.LastValue(),
		TagKeys:     viewTags(metric.By),
	}
	gauge.view = view
	return gauge
}
