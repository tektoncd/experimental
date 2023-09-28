package recorder

import (
	"context"
	"fmt"

	"github.com/tektoncd/experimental/metrics-operator/pkg/apis/monitoring/v1alpha1"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"go.opencensus.io/stats"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

type PipelineRunHistogram struct {
	GenericRunHistogram
	Selector *metav1.LabelSelector
}

func (t *PipelineRunHistogram) Filter(run *v1alpha1.RunDimensions) (bool, error) {
	pipelineRun, ok := run.Object.(*pipelinev1beta1.PipelineRun)
	if !ok {
		return false, fmt.Errorf("expected PipelineRun, but got %T", run.Object)
	}
	if t.Selector == nil {
		return true, nil
	}
	selector, err := metav1.LabelSelectorAsSelector(t.Selector)
	if err != nil {
		return false, err
	}
	return selector.Matches(labels.Set(pipelineRun.Labels)), nil
}

func (t *PipelineRunHistogram) Record(ctx context.Context, recorder stats.Recorder, run *v1alpha1.RunDimensions) {
	matched, err := t.Filter(run)
	if err != nil {
		return
	}
	if !matched {
		return
	}
	t.GenericRunHistogram.Record(ctx, recorder, run)
}

func NewPipelineRunHistogram(metric *v1alpha1.Metric, monitor *v1alpha1.PipelineRunMonitor) *PipelineRunHistogram {
	generic := NewGenericRunHistogram(metric, "pipelinerun", monitor.Name)
	histogram := &PipelineRunHistogram{
		GenericRunHistogram: *generic,
		Selector:            monitor.Spec.Selector.DeepCopy(),
	}
	return histogram
}
