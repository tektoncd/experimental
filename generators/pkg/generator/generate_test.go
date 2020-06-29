package generator

import (
	"io/ioutil"
	"testing"

	"github.com/google/go-cmp/cmp"
	v1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

func TestGenerateTask(t *testing.T) {
	want := &v1beta1.Task{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1beta1.SchemeGroupVersion.String(),
			Kind:       "Task",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   "github-build",
			Labels: map[string]string{"generator.tekton.dev": "github-build"},
		},
		Spec: v1beta1.TaskSpec{
			Workspaces: []v1beta1.WorkspaceDeclaration{
				{
					Name:      "input",
					MountPath: "/input",
				},
			},
			Steps: []v1beta1.Step{
				{
					Container: corev1.Container{
						Name:    "build",
						Image:   "gcr.io/kaniko-project/executor:latest",
						Command: []string{"/kaniko/executor"},
						Args: []string{"--context=dir://$(workspaces.input.path)/src",
							"--destination=gcr.io/wlynch-test/kaniko-test",
							"--verbosity=debug"},
					},
				},
			},
		},
	}
	github := &GitHub{
		ObjectMeta: metav1.ObjectMeta{
			Name: "github-build",
		},
		Spec: GitHubSpec{
			URL: "https://github.com/wlynch/test",
			Steps: []v1beta1.Step{{Container: corev1.Container{
				Name:    "build",
				Image:   "gcr.io/kaniko-project/executor:latest",
				Command: []string{"/kaniko/executor"},
				Args: []string{"--context=dir://$(workspaces.input.path)/src",
					"--destination=gcr.io/wlynch-test/kaniko-test",
					"--verbosity=debug"},
			},
			}}}}
	got := GenerateTask(github)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Tasks mismatch (-want +got):\n %s", diff)
	}
}

func TestGeneratePipeline(t *testing.T) {
	path := "./testdata/pipeline.yaml"
	pipeline, err := ioutil.ReadFile(path)

	if err != nil {
		t.Fatalf("fail to read file %s: %v", path, err)
	}

	want := &v1beta1.Pipeline{}
	if err := yaml.Unmarshal(pipeline, want); err != nil {
		t.Fatalf("fail to unmarshal from the input: %v", err)
	}
	github := &GitHub{
		ObjectMeta: metav1.ObjectMeta{
			Name: "github-build",
		},
		Spec: GitHubSpec{
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
	got, err := GeneratePipeline(github)
	if err != nil {
		t.Fatalf("error from 'GeneratePipeline': %v", err)
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Pipeline mismatch (-want +got):\n %s", diff)
	}
}
