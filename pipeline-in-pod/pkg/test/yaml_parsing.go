package parse

import (
	"testing"

	"github.com/tektoncd/experimental/pipeline-in-pod/pkg/client/clientset/versioned/scheme"
	"k8s.io/apimachinery/pkg/runtime"

	cprv1alpha1 "github.com/tektoncd/experimental/pipeline-in-pod/pkg/apis/colocatedpipelinerun/v1alpha1"
)

func MustParseColocatedPipelineRun(t *testing.T, yaml string) cprv1alpha1.ColocatedPipelineRun {
	var cpr cprv1alpha1.ColocatedPipelineRun
	yaml = `apiVersion: tekton.dev/v1alpha1
	kind: ColocatedPipelineRun
	` + yaml
	MustParseYAML(t, yaml, &cpr)
	return cpr
}

func MustParseYAML(t *testing.T, yaml string, i runtime.Object) {
	if _, _, err := scheme.Codecs.UniversalDeserializer().Decode([]byte(yaml), nil, i); err != nil {
		t.Fatalf("mustParseYAML (%s): %v", yaml, err)
	}
}
