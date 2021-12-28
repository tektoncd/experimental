/*
Copied from pipeline -> taskrun experimental project
*/
package pipelineinpod

import (
	"context"
	"fmt"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/typed/pipeline/v1beta1"
	listers "github.com/tektoncd/pipeline/pkg/client/listers/pipeline/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getTaskRunIfExists(lister listers.TaskRunLister, namespace, name string) (*v1beta1.TaskRun, error) {
	tr, err := lister.TaskRuns(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("couldn't fetch taskrun %v", err)
	}
	return tr, nil
}

func getPipelineSpec(ctx context.Context, tv1beta1 tektonv1beta1.TektonV1beta1Interface, namespace, pipeline string) (*v1beta1.PipelineSpec, error) {
	p, err := tv1beta1.Pipelines(namespace).Get(ctx, pipeline, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return &p.Spec, nil
}

func getTaskSpecs(ctx context.Context, tv1beta1 tektonv1beta1.TektonV1beta1Interface, pSpec *v1beta1.PipelineSpec, namespace string) (map[string]*v1beta1.TaskSpec, error) {
	taskSpecs := map[string]*v1beta1.TaskSpec{}
	for _, ptask := range pSpec.Tasks {
		var taskSpec *v1beta1.TaskSpec
		if ptask.TaskRef == nil {
			taskSpec = &ptask.TaskSpec.TaskSpec
		} else {
			var err error
			taskSpec, err = getTaskSpec(ctx, tv1beta1, namespace, ptask.TaskRef.Name)
			if err != nil {
				return nil, fmt.Errorf("couldn't fetch taskspec for %s: %v", ptask.Name, err)
			}
		}
		taskSpecs[ptask.Name] = taskSpec
	}
	return taskSpecs, nil
}

func getTaskSpec(ctx context.Context, tv1beta1 tektonv1beta1.TektonV1beta1Interface, namespace, task string) (*v1beta1.TaskSpec, error) {
	t, err := tv1beta1.Tasks(namespace).Get(ctx, task, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return &t.Spec, nil
}
