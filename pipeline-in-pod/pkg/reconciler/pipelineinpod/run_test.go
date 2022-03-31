package pipelineinpod

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	cprv1alpha1 "github.com/tektoncd/experimental/pipeline-in-pod/pkg/apis/colocatedpipelinerun/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestToColocatedPipelineRun(t *testing.T) {

	run := v1alpha1.Run{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Run",
			APIVersion: "tekton.dev/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-run",
			Namespace: "default",
		},
		Spec: v1alpha1.RunSpec{
			Spec: &v1alpha1.EmbeddedRunSpec{
				TypeMeta: runtime.TypeMeta{
					Kind:       "ColocatedPipelineRun",
					APIVersion: "tekton.dev/v1alpha1",
				},
				Metadata: v1beta1.PipelineTaskMetadata{
					Labels: map[string]string{
						"foo": "bar",
					},
				},
				Spec: runtime.RawExtension{
					Raw: []byte{},
				},
			},
		},
		Status: v1alpha1.RunStatus{
			RunStatusFields: v1alpha1.RunStatusFields{
				ExtraFields: runtime.RawExtension{
					Raw: []byte{},
				},
			},
		},
	}
	cpr, err := toColocatedPipelineRun(&run)
	expected := cprv1alpha1.ColocatedPipelineRun{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ColocatedPipelineRun",
			APIVersion: "tekton.dev/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"foo": "bar",
			},
			Namespace: "default",
			Name:      "my-run",
		},
		Spec:   cprv1alpha1.ColocatedPipelineRunSpec{},
		Status: cprv1alpha1.ColocatedPipelineRunStatus{},
	}
	if err != nil {
		t.Errorf("got err %s", err)
	}
	if diff := cmp.Diff(cpr, expected); diff != "" {
		t.Errorf("got != want: %s", diff)
	}
}
