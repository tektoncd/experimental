package recorder

import (
	"context"

	"github.com/tektoncd/experimental/metrics-operator/pkg/apis/monitoring/v1alpha1"
	"go.opencensus.io/stats"
)

type PipelineGauge struct {
	GenericRunGauge
	PipelineFilter
}

func (p *PipelineGauge) Record(ctx context.Context, recorder stats.Recorder, run *v1alpha1.RunDimensions) {
	if !p.Filter(run) {
		p.Clean(ctx, recorder, run)
		return
	}
	p.GenericRunGauge.Record(ctx, recorder, run)
}

func NewPipelineGauge(metric *v1alpha1.Metric, monitor *v1alpha1.PipelineMonitor) *PipelineGauge {
	gauge := &PipelineGauge{
		GenericRunGauge: *NewGenericRunGauge(metric, "pipeline", monitor.Name),
		PipelineFilter: PipelineFilter{
			PipelineName: monitor.Spec.PipelineName,
		},
	}
	return gauge
}
