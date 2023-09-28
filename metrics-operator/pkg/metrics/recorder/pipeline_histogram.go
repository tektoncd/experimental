package recorder

import (
	"context"

	"github.com/tektoncd/experimental/metrics-operator/pkg/apis/monitoring/v1alpha1"
	"go.opencensus.io/stats"
)

type PipelineHistogram struct {
	GenericRunHistogram
	PipelineFilter
}

func (p *PipelineHistogram) Record(ctx context.Context, recorder stats.Recorder, run *v1alpha1.RunDimensions) {
	if !p.Filter(run) {
		return
	}
	p.GenericRunHistogram.Record(ctx, recorder, run)
}

func NewPipelineHistogram(metric *v1alpha1.Metric, monitor *v1alpha1.PipelineMonitor) *PipelineHistogram {
	generic := NewGenericRunHistogram(metric, "pipeline", monitor.Name)
	histogram := &PipelineHistogram{
		GenericRunHistogram: *generic,
		PipelineFilter: PipelineFilter{
			PipelineName: monitor.Spec.PipelineName,
		},
	}
	return histogram
}
