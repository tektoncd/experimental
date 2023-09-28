package recorder

import (
	"context"

	"github.com/tektoncd/experimental/metrics-operator/pkg/apis/monitoring/v1alpha1"
	"go.opencensus.io/stats"
)

type PipelineRunHistogram struct {
	GenericRunHistogram
	PipelineRunFilter
}

func (p *PipelineRunHistogram) Record(ctx context.Context, recorder stats.Recorder, run *v1alpha1.RunDimensions) {
	matched, err := p.Filter(run)
	if err != nil {
		return
	}
	if !matched {
		return
	}
	p.GenericRunHistogram.Record(ctx, recorder, run)
}

func NewPipelineRunHistogram(metric *v1alpha1.Metric, monitor *v1alpha1.PipelineRunMonitor) *PipelineRunHistogram {
	generic := NewGenericRunHistogram(metric, "pipelinerun", monitor.Name)
	histogram := &PipelineRunHistogram{
		GenericRunHistogram: *generic,
		PipelineRunFilter: PipelineRunFilter{
			Selector: monitor.Spec.Selector.DeepCopy(),
		},
	}
	return histogram
}
