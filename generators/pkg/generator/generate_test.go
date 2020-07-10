package generator

import (
	"io/ioutil"
	"testing"

	"github.com/google/go-cmp/cmp"
	v1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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

func TestGenerateTrigger(t *testing.T) {
	// read the want TriggerBinding
	path := "./testdata/triggerbinding.yaml"
	file, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatalf("fail to read file %s: %v", path, err)
	}

	tb := &v1alpha1.TriggerBinding{}
	if err := yaml.Unmarshal(file, tb); err != nil {
		t.Fatalf("fail to unmarshal from the input: %v", err)
	}

	// read the Trigger's resourcetemplate
	path = "./testdata/pipelinerun.yaml"
	file, err = ioutil.ReadFile(path)
	if err != nil {
		t.Fatalf("fail to read file %s: %v", path, err)
	}

	pr := &v1beta1.PipelineRun{}
	if err := yaml.Unmarshal(file, pr); err != nil {
		t.Fatalf("fail to unmarshal from the input: %v", err)
	}

	// read the want TriggerTemplate
	path = "./testdata/triggertemplate.yaml"
	file, err = ioutil.ReadFile(path)
	if err != nil {
		t.Fatalf("fail to read file %s: %v", path, err)
	}

	tt := &v1alpha1.TriggerTemplate{}
	if err := yaml.Unmarshal(file, tt); err != nil {
		t.Fatalf("fail to unmarshal from the input: %v", err)
	}

	tt.Spec.ResourceTemplates = []v1alpha1.TriggerResourceTemplate{
		{runtime.RawExtension{Object: pr}},
	}

	// read the want EventListener
	path = "./testdata/eventlistener.yaml"
	file, err = ioutil.ReadFile(path)
	if err != nil {
		t.Fatalf("fail to read file %s: %v", path, err)
	}

	el := &v1alpha1.EventListener{}
	if err := yaml.Unmarshal(file, el); err != nil {
		t.Fatalf("fail to unmarshal from the input: %v", err)
	}

	want := &Trigger{
		TriggerBinding:  tb,
		TriggerTemplate: tt,
		EventListener:   el,
	}

	pipeline := &v1beta1.Pipeline{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "github-pipeline",
			Labels: map[string]string{"generator.tekton.dev": "github-pipeline"},
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1beta1.SchemeGroupVersion.String(),
			Kind:       "Pipeline",
		},
		Spec: v1beta1.PipelineSpec{
			Workspaces: []v1beta1.PipelineWorkspaceDeclaration{
				{
					Name: "source",
				},
			},
		},
	}
	got := GenerateTrigger(pipeline)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Trigger mismatch (-want +got):\n %s", diff)
	}
}
