package recorder

import (
	"context"

	"github.com/tektoncd/experimental/metrics-operator/pkg/apis/monitoring/v1alpha1"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"go.opencensus.io/stats"
)

type PipelineHistogram struct {
	GenericRunHistogram
	PipelineName string
}

func (t *PipelineHistogram) Filter(run *v1alpha1.RunDimensions) bool {
	pipelineRun, ok := run.Object.(*pipelinev1beta1.PipelineRun)
	if !ok {
		return false
	}
	ref := pipelineRun.Spec.PipelineRef
	return ref != nil && ref.Name == t.PipelineName
}

func (t *PipelineHistogram) Record(ctx context.Context, recorder stats.Recorder, run *v1alpha1.RunDimensions) {
	if !t.Filter(run) {
		return
	}
	t.GenericRunHistogram.Record(ctx, recorder, run)
}

func NewPipelineHistogram(metric *v1alpha1.Metric, monitor *v1alpha1.PipelineMonitor) *PipelineHistogram {
	generic := NewGenericRunHistogram(metric, "pipeline", monitor.Name)
	histogram := &PipelineHistogram{
		GenericRunHistogram: *generic,
		PipelineName:        monitor.Spec.PipelineName,
	}
	return histogram
}
