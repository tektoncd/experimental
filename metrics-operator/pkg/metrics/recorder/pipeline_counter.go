package recorder

import (
	"context"

	"github.com/tektoncd/experimental/metrics-operator/pkg/apis/monitoring/v1alpha1"
	"go.opencensus.io/stats"
)

type PipelineCounter struct {
	GenericRunCounter
	PipelineFilter
}

func (p *PipelineCounter) Record(ctx context.Context, recorder stats.Recorder, run *v1alpha1.RunDimensions) {
	if !p.Filter(run) {
		return
	}
	p.GenericRunCounter.Record(ctx, recorder, run)
}

func NewPipelineCounter(metric *v1alpha1.Metric, monitor *v1alpha1.PipelineMonitor) *PipelineCounter {
	generic := NewGenericRunCounter(metric, "pipeline", monitor.Name)
	counter := &PipelineCounter{
		GenericRunCounter: *generic,
		PipelineFilter: PipelineFilter{
			PipelineName: monitor.Spec.PipelineName,
		},
	}
	return counter
}
