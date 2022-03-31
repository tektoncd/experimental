package pipelineinpod

import (
	"encoding/json"

	cprv1alpha1 "github.com/tektoncd/experimental/pipeline-in-pod/pkg/apis/colocatedpipelinerun/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func DecodeExtraFields(run v1alpha1.RunSpec, into interface{}) error {
	if run.Spec == nil || len(run.Spec.Spec.Raw) == 0 {
		return nil
	}
	return json.Unmarshal(run.Spec.Spec.Raw, into)
}

func EncodeExtraFields(run *v1alpha1.RunSpec, from interface{}) error {
	data, err := json.Marshal(from)
	if err != nil {
		return err
	}
	run.Spec.Spec = runtime.RawExtension{
		Raw: data,
	}
	return nil
}

func toColocatedPipelineRun(run *v1alpha1.Run) (cprv1alpha1.ColocatedPipelineRun, error) {
	var cpr cprv1alpha1.ColocatedPipelineRun

	spec := &cprv1alpha1.ColocatedPipelineRunSpec{}
	if err := DecodeExtraFields(run.Spec, spec); err != nil {
		return cpr, err
	}
	status := &cprv1alpha1.ColocatedPipelineRunStatus{}
	if err := run.Status.DecodeExtraFields(status); err != nil {
		return cpr, err
	}
	cpr.Spec = *spec
	cpr.Status = *status
	cpr.TypeMeta = metav1.TypeMeta{
		Kind:       run.Spec.Spec.Kind,
		APIVersion: run.Spec.Spec.APIVersion,
	}
	cpr.ObjectMeta = metav1.ObjectMeta{
		Labels:      run.Spec.Spec.Metadata.Labels,
		Annotations: run.Spec.Spec.Metadata.Annotations,
		Namespace:   run.Namespace,
	}
	if cpr.Name == "" {
		cpr.Name = run.Name
	}
	if cpr.UID == "" && run.UID != "" {
		cpr.UID = run.UID
	}
	return cpr, nil
}

func updateRunFromColocatedPipelineRun(run *v1alpha1.Run, cpr cprv1alpha1.ColocatedPipelineRun) error {
	if err := run.Status.EncodeExtraFields(cpr.Status); err != nil {
		return err
	}
	// TODO: smarter translation e.g. translating reasons
	run.Status.Status = cpr.Status.Status
	return nil
}
