package parser

import (
	"generators/pkg/generator"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	v1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
)

func TestParse(t *testing.T) {
	file, err := os.Open("testdata/spec.yaml")
	if err != nil {
		t.Fatalf("fail to open file 'testdata/spec.yaml': %v", err)
	}

	got, err := Parse(file)
	if err != nil {
		t.Fatalf("error from 'Parse': %v", err)
	}

	want := generator.GitHubSpec{
		URL: "https://github.com/wlynch/test",
		Steps: []v1beta1.Step{{Container: corev1.Container{
			Name:    "build",
			Image:   "gcr.io/kaniko-project/executor:latest",
			Command: []string{"/kaniko/executor"},
			Args: []string{"--context=dir://$(workspaces.input.path)/src",
				"--destination=gcr.io/wlynch-test/kaniko-test",
				"--verbosity=debug"},
		}}}}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("GitHubSpec mismatch (-want +got):\n %s", diff)
	}

	file.Close()
}
