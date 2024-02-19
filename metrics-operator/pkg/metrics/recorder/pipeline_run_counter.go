package recorder

import (
	"context"

	"github.com/tektoncd/experimental/metrics-operator/pkg/apis/monitoring/v1alpha1"
	"go.opencensus.io/stats"
	"knative.dev/pkg/logging"
)

type PipelineRunCounter struct {
	GenericRunCounter
	PipelineRunFilter
}

func (t *PipelineRunCounter) Record(ctx context.Context, recorder stats.Recorder, run *v1alpha1.RunDimensions) {
	logger := logging.FromContext(ctx)
	matched, err := t.Filter(run)
	if err != nil {
		logger.Errorf("could not record metric")
		return
	}
	if !matched {
		return
	}
	t.GenericRunCounter.Record(ctx, recorder, run)
}

func NewPipelineRunCounter(metric *v1alpha1.Metric, monitor *v1alpha1.PipelineRunMonitor) *PipelineRunCounter {
	generic := NewGenericRunCounter(metric, "pipelinerun", monitor.Name)
	counter := &PipelineRunCounter{
		GenericRunCounter: *generic,
		PipelineRunFilter: PipelineRunFilter{
			Selector: monitor.Spec.Selector.DeepCopy(),
		},
	}
	return counter
}
