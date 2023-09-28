package metrics

import (
	"context"
	"fmt"
	"sync"

	"github.com/tektoncd/experimental/metrics-operator/pkg/apis/monitoring/v1alpha1"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"go.opencensus.io/stats/view"
	"k8s.io/apimachinery/pkg/types"
)

type MetricManager struct {
	Index *MetricIndex
	runs  map[string]*sync.Once
	rw    sync.RWMutex
}

func (m *MetricManager) GetIndex() *MetricIndex {
	return m.Index
}

func (m *MetricManager) clean(ctx context.Context, run *v1alpha1.RunDimensions) {
	m.GetIndex().Clean(ctx, run)
	m.rw.Lock()
	defer m.rw.Unlock()

	var uid types.UID
	switch r := run.Object.(type) {
	case *pipelinev1beta1.PipelineRun:
		uid = r.ObjectMeta.UID
	case *pipelinev1beta1.TaskRun:
		uid = r.ObjectMeta.UID
	}
	key := fmt.Sprintf("%s/%s/%s", run.Namespace, run.Name, uid)
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
