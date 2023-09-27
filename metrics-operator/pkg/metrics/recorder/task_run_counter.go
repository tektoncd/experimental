package recorder

import (
	"context"

	"github.com/tektoncd/experimental/metrics-operator/pkg/apis/monitoring/v1alpha1"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"go.opencensus.io/stats"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"knative.dev/pkg/logging"
)

type TaskRunCounter struct {
	GenericTaskRunCounter
	Selector *metav1.LabelSelector
}

// Filter returns true when the TaskRun should be recorded, independent of value
func (t *TaskRunCounter) Filter(taskRun *pipelinev1beta1.TaskRun) (bool, error) {
	if t.Selector == nil {
		return true, nil
	}
	selector, err := metav1.LabelSelectorAsSelector(t.Selector)
	if err != nil {
		return false, err
	}
	return selector.Matches(labels.Set(taskRun.Labels)), nil
}

func (t *TaskRunCounter) Record(ctx context.Context, recorder stats.Recorder, taskRun *pipelinev1beta1.TaskRun) {
	logger := logging.FromContext(ctx)
	matched, err := t.Filter(taskRun)
	if err != nil {
		logger.Errorf("could not record metric")
		return
	}
	if !matched {
		return
	}
	t.GenericTaskRunCounter.Record(ctx, recorder, taskRun)
}

func NewTaskRunCounter(metric *v1alpha1.Metric, monitor *v1alpha1.TaskRunMonitor) *TaskRunCounter {
	generic := NewGenericTaskRunCounter(metric, "taskrun", monitor.Name)
	counter := &TaskRunCounter{
		GenericTaskRunCounter: *generic,
		Selector:              monitor.Spec.Selector.DeepCopy(),
	}
	return counter
}
