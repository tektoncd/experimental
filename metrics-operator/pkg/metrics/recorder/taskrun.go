package recorder

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/tektoncd/experimental/metrics-operator/pkg/apis/monitoring/v1alpha1"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

type TaskRunRecorder struct{}

func (t *TaskRunRecorder) RecordTaskRunDone(taskRun *pipelinev1beta1.TaskRun) {

}

type MetricIndex struct {
	external view.Meter
	// TODO: generalize and synchronize
	store map[string]TaskRunCounter
}

func (m *MetricIndex) Record(ctx context.Context, taskRun *pipelinev1beta1.TaskRun) {
	for _, metric := range m.store {
		metric.Record(ctx, m.external, taskRun)
	}
}

func (m *MetricIndex) RegisterTaskMetric(ctx context.Context, taskMonitor *v1alpha1.TaskMonitor, taskMetric *v1alpha1.TaskMetric) error {
	if taskMetric.Type == "counter" {
		counter := &TaskRunCounter{
			TaskName:        taskMonitor.Spec.TaskName,
			TaskMonitorName: taskMetric.Name,
			TaskMetric:      taskMetric,
		}
		isRegistered, isModified := m.IsTaskRunCounterRegistered(counter)
		if isRegistered && isModified {
			err := m.UnregisterTaskRunCounter(counter)
			if err != nil {
				return err
			}
		}
		if isRegistered && !isModified {
			return nil
		}
		m.store[counter.MetricName()] = *counter
		err := m.external.Register(counter.View())
		if err != nil {
			return err
		}
		return nil
	}
	return errors.New("Not implemented")
}

// IsTaskRunCounterRegistered returns two booleans: if it's registered, and if it's modified
func (m *MetricIndex) IsTaskRunCounterRegistered(counter *TaskRunCounter) (bool, bool) {
	// TODO
	viewFound := m.external.Find(counter.MetricName())
	if viewFound != nil {
		lastSeenCounter := m.store[counter.MetricName()]
		isModified := false
		// TODO: improve isModified to use some kind of diff or Equals
		if lastSeenCounter.TaskName != counter.TaskName {
			isModified = true
		}
		return true, isModified
	}
	return false, false
}

func (m *MetricIndex) UnregisterTaskRunCounter(counter *TaskRunCounter) error {
	//TODO
	return nil
}

type TaskRunCounter struct {
	TaskName        string
	TaskMonitorName string
	TaskMetric      *v1alpha1.TaskMetric
}

func (t *TaskRunCounter) MetricName() string {
	resource := "taskrun"
	return fmt.Sprintf("tekton_metrics_%s_%s_%s_total", resource, strings.ReplaceAll(t.TaskMonitorName, "-", "_"), t.TaskMetric.Name)
}

func (t *TaskRunCounter) Measure() *stats.Float64Measure {
	return stats.Float64(t.MetricName(), fmt.Sprintf("count samples for TaskMonitor %s", t.TaskMonitorName), stats.UnitDimensionless)
}

func (t *TaskRunCounter) View() *view.View {
	countMeasure := t.Measure()
	return &view.View{
		Description: countMeasure.Description(),
		Measure:     countMeasure,
		Aggregation: view.Count(),
		// TODO: tags
	}
}

func (t *TaskRunCounter) Record(ctx context.Context, recorder stats.Recorder, taskRun *pipelinev1beta1.TaskRun) {
	recorder.Record(&tag.Map{}, []stats.Measurement{t.Measure().M(1)}, map[string]any{})
}
