package recorder

import (
	"context"
	"fmt"

	"github.com/tektoncd/experimental/metrics-operator/pkg/apis/monitoring/v1alpha1"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"go.opencensus.io/tag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/util/sets"
)

func tagMapFromByStatements(by []v1alpha1.ByStatement, run *v1alpha1.RunDimensions) (*tag.Map, error) {
	mutators := []tag.Mutator{}
	for _, byStatement := range by {
		byKey, err := byStatement.Key()
		if err != nil {
			return nil, err
		}
		byValue, err := byStatement.Value(run)
		if err != nil {
			return nil, err
		}
		// TODO: error handling
		tagKey := tag.MustNewKey(byKey)
		mutators = append(mutators, tag.Upsert(tagKey, byValue))
	}
	ctx, err := tag.New(context.Background(), mutators...)
	if err != nil {
		return nil, err
	}
	return tag.FromContext(ctx), nil
}

func viewTags(by []v1alpha1.ByStatement) []tag.Key {
	keys := []tag.Key{}
	for _, byStatement := range by {
		key, err := byStatement.Key()
		if err != nil {
			continue
		}
		tagKey, err := tag.NewKey(key)
		if err != nil {
			continue
		}
		keys = append(keys, tagKey)
	}
	return keys
}

func match(m *v1alpha1.MetricGaugeMatch, run *v1alpha1.RunDimensions) (bool, error) {
	v, err := m.Key.Value(run)
	if err != nil {
		return false, err
	}

	values := sets.New[string](m.Values...)
	switch m.Operator {
	case metav1.LabelSelectorOpIn:
		return values.Has(v), nil
	case metav1.LabelSelectorOpNotIn:
		return !values.Has(v), nil
	default:
		return false, fmt.Errorf("unsupported operation: %q", m.Operator)
	}
}

func TaskRunDimensions(taskRun *pipelinev1beta1.TaskRun) *v1alpha1.RunDimensions {
	return &v1alpha1.RunDimensions{
		Resource:  "taskrun",
		Name:      taskRun.Name,
		Namespace: taskRun.Namespace,
		IsDeleted: taskRun.DeletionTimestamp != nil,
		Status:    taskRun.Status.Status,
		Labels:    taskRun.Labels,
		Params:    taskRun.Spec.Params,
		Object:    taskRun,
	}
}

func PipelineRunDimensions(pipelineRun *pipelinev1beta1.PipelineRun) *v1alpha1.RunDimensions {
	return &v1alpha1.RunDimensions{
		Resource:  "pipelinerun",
		Name:      pipelineRun.Name,
		Namespace: pipelineRun.Namespace,
		IsDeleted: pipelineRun.DeletionTimestamp != nil,
		Status:    pipelineRun.Status.Status,
		Labels:    pipelineRun.Labels,
		Params:    pipelineRun.Spec.Params,
		Object:    pipelineRun,
	}
}
