package metrics

import (
	"context"
	"fmt"
	"sync"

	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"knative.dev/pkg/kmp"
)

type RunMetric interface {
	MetricName() string
	MonitorName() string
	View() *view.View
	Record(ctx context.Context, recorder stats.Recorder, taskRun *pipelinev1beta1.TaskRun)
}

type MetricIndex struct {
	external view.Meter
	store    map[string]RunMetric
	rw       sync.RWMutex
}

func (m *MetricIndex) Record(ctx context.Context, taskRun *pipelinev1beta1.TaskRun) {
	for _, metric := range m.store {
		metric.Record(ctx, m.external, taskRun)
	}
}

func (m *MetricIndex) RegisterRunMetric(ctx context.Context, runMetric RunMetric) error {

	isRegistered, isModified, err := m.IsRegistered(runMetric)
	if err != nil {
		return fmt.Errorf("error verifying run metric registration: %w", err)
	}
	if isRegistered && isModified {
		err := m.UnregisterRunMetric(runMetric)
		if err != nil {
			return err
		}
	}
	if isRegistered && !isModified {
		return nil
	}

	m.rw.Lock()
	defer m.rw.Unlock()

	m.store[runMetric.MetricName()] = runMetric
	err = m.external.Register(runMetric.View())
	if err != nil {
		return err
	}
	return nil
}

// IsRegistered returns two booleans: if it's registered, and if it's modified
func (m *MetricIndex) IsRegistered(runMetric RunMetric) (bool, bool, error) {
	m.rw.Lock()
	defer m.rw.Unlock()

	viewFound := m.external.Find(runMetric.MetricName())
	if viewFound != nil {
		lastSeen := m.store[runMetric.MetricName()]
		isModified := false
		equals, err := kmp.SafeEqual(lastSeen, runMetric)
		if err != nil {
			return true, false, err
		}
		if !equals {
			isModified = true
		}
		return true, isModified, nil
	}
	return false, false, nil
}

func (m *MetricIndex) UnregisterRunMetricByName(runMetricName string) error {
	m.rw.Lock()
	defer m.rw.Unlock()

	existingView := m.external.Find(runMetricName)
	if existingView != nil {
		m.external.Unregister(existingView)
	}
	delete(m.store, runMetricName)
	return nil
}

func (m *MetricIndex) UnregisterRunMetric(runMetric RunMetric) error {
	return m.UnregisterRunMetricByName(runMetric.MetricName())
}

func (m *MetricIndex) GetAllMetricNamesFromMonitor(monitor string) []string {
	metrics := []string{}
	m.rw.Lock()
	defer m.rw.Unlock()
	for _, runMetric := range m.store {
		if monitor == runMetric.MonitorName() {
			metrics = append(metrics, runMetric.MetricName())
		}
	}
	return metrics
}

func (m *MetricIndex) UnregisterAllMetricsMonitor(monitor string) error {
	for _, metricName := range m.GetAllMetricNamesFromMonitor(monitor) {
		err := m.UnregisterRunMetricByName(metricName)
		if err != nil {
			return err
		}
	}
	return nil
}