package metrics

import (
	"context"
	"fmt"
	"sync"
	"time"

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

	m.runs[key].Do(func() {
		m.GetIndex().Record(ctx, taskRun)
	})
	time.AfterFunc(1*time.Hour, func() {
		m.clean(taskRun)
	})
	return nil
}

func (m *MetricManager) clean(taskRun *pipelinev1beta1.TaskRun) {
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
