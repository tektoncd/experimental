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

type TaskRunCounter struct {
	GenericRunCounter
	Selector *metav1.LabelSelector
}

// Filter returns true when the TaskRun should be recorded, independent of value
func (t *TaskRunCounter) Filter(run *v1alpha1.RunDimensions) (bool, error) {
	taskRun, ok := run.Object.(*pipelinev1beta1.TaskRun)
	if !ok {
		return false, fmt.Errorf("expected taskRun, but got %T", run.Object)
	}
	if t.Selector == nil {
		return true, nil
	}
	selector, err := metav1.LabelSelectorAsSelector(t.Selector)
	if err != nil {
		return false, err
	}
	return selector.Matches(labels.Set(taskRun.Labels)), nil
}

func (t *TaskRunCounter) Record(ctx context.Context, recorder stats.Recorder, run *v1alpha1.RunDimensions) {
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

func NewTaskRunCounter(metric *v1alpha1.Metric, monitor *v1alpha1.TaskRunMonitor) *TaskRunCounter {
	generic := NewGenericRunCounter(metric, "taskrun", monitor.Name)
	counter := &TaskRunCounter{
		GenericRunCounter: *generic,
		Selector:          monitor.Spec.Selector.DeepCopy(),
	}
	return counter
}
