package parser

import (
	"generators/pkg/generator"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	v1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestParse(t *testing.T) {
	file, err := os.Open("testdata/spec-full.yaml")
	if err != nil {
		t.Fatalf("fail to open file 'testdata/spec-full.yaml': %v", err)
	}
	defer file.Close()
	got, err := Parse(file)
	if err != nil {
		t.Fatalf("error from 'Parse': %v", err)
	}

	want := &generator.GitHub{
		TypeMeta: metav1.TypeMeta{
			Kind: "GitHub",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "github-build",
		},
		Spec: generator.GitHubSpec{
			URL: "https://github.com/wlynch/test",
			Steps: []v1beta1.Step{
				{
					Container: corev1.Container{
						Name:    "build",
						Image:   "gcr.io/kaniko-project/executor:latest",
						Command: []string{"/kaniko/executor"},
						Args: []string{"--context=dir://$(workspaces.input.path)/src",
							"--destination=gcr.io/<use your project>/kaniko-test",
							"--verbosity=debug"},
					},
				},
			},
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("GitHubSpec mismatch (-want +got):\n %s", diff)
	}

}
