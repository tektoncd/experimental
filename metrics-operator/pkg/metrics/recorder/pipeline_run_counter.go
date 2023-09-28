package recorder

import (
	"context"
	"fmt"

	"github.com/tektoncd/experimental/metrics-operator/pkg/apis/monitoring/v1alpha1"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"go.opencensus.io/stats"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"knative.dev/pkg/logging"
)

type PipelineRunCounter struct {
	GenericRunCounter
	Selector *metav1.LabelSelector
}

// Filter returns true when the PipelineRun should be recorded, independent of value
func (t *PipelineRunCounter) Filter(run *v1alpha1.RunDimensions) (bool, error) {
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
		Selector:          monitor.Spec.Selector.DeepCopy(),
	}
	return counter
}
