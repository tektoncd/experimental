package pipelineinpod

import (
	"context"
	"fmt"

	cprv1alpha1 "github.com/tektoncd/experimental/pipeline-in-pod/pkg/apis/colocatedpipelinerun/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	clientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"github.com/tektoncd/pipeline/pkg/reconciler/pipelinerun/resources"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// GetPipeline is a function used to retrieve Pipelines.
type GetPipeline func(context.Context, string) (v1beta1.PipelineObject, error)

// GetPipelineData will retrieve the Pipeline metadata and Spec associated with the
// provided Run. This can come from a reference Pipeline or from the Run's
// metadata and embedded PipelineSpec.
func GetPipelineData(ctx context.Context, run *cprv1alpha1.ColocatedPipelineRun, getPipeline GetPipeline) (*metav1.ObjectMeta, *v1beta1.PipelineSpec, error) {
	pipelineMeta := metav1.ObjectMeta{}
	pipelineSpec := v1beta1.PipelineSpec{}
	switch {
	case run.Spec.PipelineRef != nil && run.Spec.PipelineRef.Name != "":
		// Get related pipeline for run
		t, err := getPipeline(ctx, run.Spec.PipelineRef.Name)
		if err != nil {
			return nil, nil, fmt.Errorf("error when listing pipelines for run %s: %w", run.Name, err)
		}
		pipelineMeta = t.PipelineMetadata()
		pipelineSpec = t.PipelineSpec()
	case run.Spec.PipelineSpec != nil:
		pipelineMeta = run.ObjectMeta
		pipelineSpec = *run.Spec.PipelineSpec
	default:
		return nil, nil, fmt.Errorf("run %s not providing PipelineRef or PipelineSpec", run.Name)
	}
	return &pipelineMeta, &pipelineSpec, nil
}

func GetPipelineFunc(ctx context.Context, k8s kubernetes.Interface, tekton clientset.Interface, cpr *cprv1alpha1.ColocatedPipelineRun) (GetPipeline, error) {
	// if the spec is already in the status, do not try to fetch it again, just use it as source of truth
	if cpr.Status.PipelineSpec != nil {
		return func(_ context.Context, name string) (v1beta1.PipelineObject, error) {
			return &v1beta1.Pipeline{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: cpr.Namespace,
				},
				Spec: *cpr.Status.PipelineSpec,
			}, nil
		}, nil
	}
	local := &resources.LocalPipelineRefResolver{
		Namespace:    cpr.Namespace,
		Tektonclient: tekton,
	}
	return local.GetPipeline, nil
}

func storePipelineSpecAndMergeMeta(cpr *cprv1alpha1.ColocatedPipelineRun, ps *v1beta1.PipelineSpec, meta *metav1.ObjectMeta) error {
	// Only store the PipelineSpec once, if it has never been set before.
	if cpr.Status.PipelineSpec != nil {
		return nil
	}
	cpr.Status.PipelineSpec = ps

	// Propagate labels from Pipeline to PipelineRun.
	if cpr.ObjectMeta.Labels == nil {
		cpr.ObjectMeta.Labels = make(map[string]string, len(meta.Labels)+1)
	}
	for key, value := range meta.Labels {
		cpr.ObjectMeta.Labels[key] = value
	}
	cpr.ObjectMeta.Labels[pipeline.PipelineLabelKey] = meta.Name

	// Propagate annotations from Pipeline to ColocatedPipelineRun.
	if cpr.ObjectMeta.Annotations == nil {
		cpr.ObjectMeta.Annotations = make(map[string]string, len(meta.Annotations))
	}
	for key, value := range meta.Annotations {
		cpr.ObjectMeta.Annotations[key] = value
	}
	return nil
}
