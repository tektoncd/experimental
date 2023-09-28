package recorder

import (
	"context"

	"github.com/tektoncd/experimental/metrics-operator/pkg/apis/monitoring/v1alpha1"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"go.opencensus.io/stats"
)

type PipelineCounter struct {
	GenericRunCounter
	PipelineName string
}

// Filter returns true when the PipelineRun should be recorded, independent of value
func (t *PipelineCounter) Filter(run *v1alpha1.RunDimensions) bool {
	pipelineRun, ok := run.Object.(*pipelinev1beta1.PipelineRun)
	if !ok {
		return false
	}
	ref := pipelineRun.Spec.PipelineRef
	return ref != nil && ref.Name == t.PipelineName
}

func (t *PipelineCounter) Record(ctx context.Context, recorder stats.Recorder, run *v1alpha1.RunDimensions) {
	if !t.Filter(run) {
		return
	}
	t.GenericRunCounter.Record(ctx, recorder, run)
}

func NewPipelineCounter(metric *v1alpha1.Metric, monitor *v1alpha1.PipelineMonitor) *PipelineCounter {
	generic := NewGenericRunCounter(metric, "pipeline", monitor.Name)
	counter := &PipelineCounter{
		GenericRunCounter: *generic,
		PipelineName:      monitor.Spec.PipelineName,
	}
	return counter
}
