package metrics

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/tektoncd/experimental/metrics-operator/pkg/metrics/recorder"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"go.opencensus.io/stats/view"
)

type MetricManager struct {
	Index *MetricIndex
	runs  map[string]*sync.Once
	rw    sync.RWMutex
}

func (m *MetricManager) GetIndex() *MetricIndex {
	return m.Index
}

func (m *MetricManager) RecordTaskRunDone(ctx context.Context, taskRun *pipelinev1beta1.TaskRun) error {
	if !taskRun.IsDone() {
		return fmt.Errorf("record task run done called with a running TaskRun")
	}
	m.rw.Lock()
	defer m.rw.Unlock()
	key := fmt.Sprintf("%s/%s/%s", taskRun.GetNamespace(), taskRun.GetName(), taskRun.GetUID())
	_, exists := m.runs[key]
	if !exists {
		m.runs[key] = &sync.Once{}
	}

	run := recorder.TaskRunDimensions(taskRun)

	m.runs[key].Do(func() {
		m.GetIndex().Record(ctx, run, "histogram")
		m.GetIndex().Record(ctx, run, "counter")
		m.GetIndex().Record(ctx, run, "gauge")
	})
	time.AfterFunc(1*time.Hour, func() {
		m.clean(ctx, taskRun)
	})
	return nil
}

func (m *MetricManager) RecordTaskRunRunning(ctx context.Context, taskRun *pipelinev1beta1.TaskRun) error {
	if taskRun.IsDone() {
		return fmt.Errorf("record task run running called with a done TaskRun")
	}
	run := recorder.TaskRunDimensions(taskRun)
	m.GetIndex().Record(ctx, run, "gauge")
	return nil
}

func (m *MetricManager) clean(ctx context.Context, taskRun *pipelinev1beta1.TaskRun) {

	run := recorder.TaskRunDimensions(taskRun)
	m.GetIndex().Clean(ctx, run)
	m.rw.Lock()
	defer m.rw.Unlock()
	key := fmt.Sprintf("%s/%s/%s", taskRun.GetNamespace(), taskRun.GetName(), taskRun.GetUID())
	_, exists := m.runs[key]
	if exists {
		delete(m.runs, key)
	}
}

func NewManager(external view.Meter) *MetricManager {
	return &MetricManager{
		Index: &MetricIndex{
			external: external,
			store:    map[string]RunMetric{},
		},
		runs: map[string]*sync.Once{},
	}
}
