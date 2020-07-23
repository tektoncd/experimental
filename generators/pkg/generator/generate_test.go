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

// testdata
var github = &GitHub{
	ObjectMeta: metav1.ObjectMeta{
		Name: "github-build",
	},
	Spec: GitHubSpec{
		URL:                "https://github.com/wlynch/test",
		Revision:           "df5b1b84c23c6c4f41a4e51ba02da0095acf59e7",
		Branch:             "master",
		ServiceAccountName: "tekton-generators-demo",
		Storage:            "1Gi",
		SecretName:         "github-secret",
		SecretKey:          "secretToken",
		Steps: []v1beta1.Step{
			{
				Container: corev1.Container{
					Name:    "build",
					Image:   "gcr.io/kaniko-project/executor:latest",
					Command: []string{"/kaniko/executor"},
					Args: []string{"--context=dir://$(workspaces.input.path)/src",
						"--destination=gcr.io/tekton-yolandadu/kaniko-test",
						"--verbosity=debug"},
				},
			},
		},
	},
}

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
							"--destination=gcr.io/tekton-yolandadu/kaniko-test",
							"--verbosity=debug"},
					},
				},
			},
		},
	}
	got, err := GenerateTask(github)
	if err != nil {
		t.Fatalf("error from 'GenerateTask': %v", err)
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Tasks mismatch (-want +got):\n %s", diff)
	}
}

func TestGeneratePipeline(t *testing.T) {
	want := &v1beta1.Pipeline{}
	unmarshal(t, "./testdata/pipeline.yaml", want)

	got, err := GeneratePipeline(github)
	if err != nil {
		t.Fatalf("error from 'GeneratePipeline': %v", err)
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Pipeline mismatch (-want +got):\n %s", diff)
	}
}

func TestGeneratePipelineRun(t *testing.T) {
	tables := []struct {
		name     string
		pipeline string
		config   *GitHub
		path     string
	}{
		{
			name:     "TestWithAllDefinedFields",
			pipeline: "./testdata/pipeline.yaml",
			config:   github,
			path:     "./testdata/run.yaml",
		},
		{
			name:     "TestWithNonDefaultNamespace",
			pipeline: "./testdata/pipeline1.yaml",
			config:   github,
			path:     "./testdata/run1.yaml",
		},
		{
			name:     "TestWithoutOptionalFields",
			pipeline: "./testdata/pipeline.yaml",
			config: &GitHub{
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
							},
						},
					},
				},
			},
			path: "./testdata/run2.yaml",
		},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			p := &v1beta1.Pipeline{}
			unmarshal(t, table.pipeline, p)
			got, err := GeneratePipelineRun(p, table.config)
			if err != nil {
				t.Fatalf("error from 'GeneratePipelineRun': %v", err)
			}
			want := &v1beta1.PipelineRun{}
			unmarshal(t, table.path, want)
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("PipelineRun mismatch (-want +got):\n %s", diff)

			}
		})
	}
}

func TestGenerateTrigger(t *testing.T) {
	// read the want TriggerBinding for push events
	tbPush := &v1alpha1.TriggerBinding{}
	unmarshal(t, "./testdata/triggerbinding.yaml", tbPush)

	// read the want TriggerBinding for pull request events
	tbPr := &v1alpha1.TriggerBinding{}
	unmarshal(t, "./testdata/triggerbinding-pr.yaml", tbPr)

	// read the Trigger's resourcetemplate
	pr := &v1beta1.PipelineRun{}
	unmarshal(t, "./testdata/pipelinerun.yaml", pr)

	// read the want TriggerTemplate
	tt := &v1alpha1.TriggerTemplate{}
	unmarshal(t, "./testdata/triggertemplate.yaml", tt)

	tt.Spec.ResourceTemplates = []v1alpha1.TriggerResourceTemplate{
		{runtime.RawExtension{Object: pr}},
	}

	// read the want EventListener
	el := &v1alpha1.EventListener{}
	unmarshal(t, "./testdata/eventlistener.yaml", el)

	want := &trigger{
		TriggerBinding:  []*v1alpha1.TriggerBinding{tbPush, tbPr},
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
	got, err := GenerateTrigger(pipeline, github)
	if err != nil {
		t.Fatalf("error from 'GenerateTrigger': %v", err)
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Trigger mismatch (-want +got):\n %s", diff)
	}
}

func unmarshal(t *testing.T, path string, i interface{}) {
	t.Helper()

	file, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatalf("fail to read file %s: %v", path, err)
	}

	if err := yaml.Unmarshal(file, i); err != nil {
		t.Fatalf("fail to unmarshal from the input: %v", err)
	}
}
