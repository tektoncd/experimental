package recorder

import (
	"context"

	"github.com/tektoncd/experimental/metrics-operator/pkg/apis/monitoring/v1alpha1"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"go.opencensus.io/tag"
)

func tagMapFromByStatements(by []v1alpha1.TaskByStatement, taskRun *pipelinev1beta1.TaskRun) (*tag.Map, error) {
	mutators := []tag.Mutator{}
	for _, byStatement := range by {
		byKey, err := byStatement.Key()
		if err != nil {
			return nil, err
		}
		byValue, err := byStatement.Value(taskRun)
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

func viewTags(by []v1alpha1.TaskByStatement) []tag.Key {
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
