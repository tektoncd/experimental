package recorder

import (
	"context"

	"github.com/tektoncd/experimental/metrics-operator/pkg/apis/monitoring/v1alpha1"
	"go.opencensus.io/stats"
)

type PipelineRunGauge struct {
	GenericRunGauge
	PipelineRunFilter
}

func (p *PipelineRunGauge) Record(ctx context.Context, recorder stats.Recorder, run *v1alpha1.RunDimensions) {
	matched, err := p.Filter(run)
	if err != nil {
		p.Clean(ctx, recorder, run)
		return
	}
	if !matched {
		p.Clean(ctx, recorder, run)
		return
	}
	p.GenericRunGauge.Record(ctx, recorder, run)
}

func NewPipelineRunGauge(metric *v1alpha1.Metric, monitor *v1alpha1.PipelineRunMonitor) *PipelineRunGauge {
	gauge := &PipelineRunGauge{
		GenericRunGauge: *NewGenericRunGauge(metric, "pipelinerun", monitor.Name),
		PipelineRunFilter: PipelineRunFilter{
			Selector: monitor.Spec.Selector.DeepCopy(),
		},
	}
	return gauge
}
