package generator

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	v1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
)

func TestGenerateTask(t *testing.T) {
	want := v1beta1.Task{
		Spec: v1beta1.TaskSpec{
			Workspaces: []v1beta1.WorkspaceDeclaration{{Name: "input",
				MountPath: "/input"}},
			Steps: []v1beta1.Step{{Container: corev1.Container{
				Name:    "build",
				Image:   "gcr.io/kaniko-project/executor:latest",
				Command: []string{"/kaniko/executor"},
				Args: []string{"--context=dir://$(workspaces.input.path)/src",
					"--destination=gcr.io/wlynch-test/kaniko-test",
					"--verbosity=debug"},
			}}}}}
	spec := GitHubSpec{
		URL: "https://github.com/wlynch/test",
		Steps: []v1beta1.Step{{Container: corev1.Container{
			Name:    "build",
			Image:   "gcr.io/kaniko-project/executor:latest",
			Command: []string{"/kaniko/executor"},
			Args: []string{"--context=dir://$(workspaces.input.path)/src",
				"--destination=gcr.io/wlynch-test/kaniko-test",
				"--verbosity=debug"},
		}}}}
	got, err := GenerateTask(spec)
	if err != nil {
		t.Fatalf("error from 'GenerateTask': %v", err)
	}

	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("Tasks mismatch (-want +got):\n %s", diff)
	}
}
