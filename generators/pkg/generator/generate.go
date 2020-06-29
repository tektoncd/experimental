// Package generator provides a method to generating Tekton spec
// from simplified configs.
package generator

import (
	"fmt"
	"net/url"

	v1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GitHub defines Github fields
type GitHub struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              GitHubSpec `json:"spec"`
}

// GithubSpec defines Github spec
type GitHubSpec struct {
	URL   string         `json:"url,omitempty"`
	Steps []v1beta1.Step `json:"steps,omitempty"`
}

// GenerateTask generates Tekton Task
// from simplified Github configs.
func GenerateTask(github *GitHub) *v1beta1.Task {
	labels := github.Labels
	if labels == nil {
		labels = make(map[string]string)
	}
	labels["generator.tekton.dev"] = github.Name
	return &v1beta1.Task{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1beta1.SchemeGroupVersion.String(),
			Kind:       "Task",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   github.Name,
			Labels: labels,
		},
		Spec: v1beta1.TaskSpec{
			Workspaces: []v1beta1.WorkspaceDeclaration{
				{
					Name:      "input",
					MountPath: "/input",
				},
			},
			Steps: github.Spec.Steps,
		},
	}

}

// GeneratePipeline generates Tekton Pipeline
// from simplified Github configs.
func GeneratePipeline(github *GitHub) (*v1beta1.Pipeline, error) {
	ws := "source"
	name := "build-from-git-repo"
	tasksName := []string{"fetch-git-repo", "build-from-repo", "final-set-status"}

	u, err := url.Parse(github.Spec.URL)
	if err != nil {
		return nil, fmt.Errorf("fail to parse the url %s: %w", github.Spec.URL, err)
	}

	pipeline := &v1beta1.Pipeline{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1beta1.SchemeGroupVersion.String(),
			Kind:       "Pipeline",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{"generator.tekton.dev": name},
		},
		Spec: v1beta1.PipelineSpec{
			Tasks: []v1beta1.PipelineTask{
				{
					Name: tasksName[0],
					TaskRef: &v1beta1.TaskRef{
						Name: "git-clone",
					},
					Params: []v1beta1.Param{
						{
							Name: "url",
							Value: v1beta1.ArrayOrString{
								Type:      v1beta1.ParamTypeString,
								StringVal: github.Spec.URL,
							},
						},
					},
					Workspaces: []v1beta1.WorkspacePipelineTaskBinding{
						{
							Name:      "output",
							Workspace: ws,
						},
					},
				},

				{
					Name: tasksName[1],
					TaskRef: &v1beta1.TaskRef{
						Name: github.Name,
					},

					Workspaces: []v1beta1.WorkspacePipelineTaskBinding{
						{
							Name:      "input",
							Workspace: ws,
						},
					},
					RunAfter: []string{
						tasksName[0],
					},
				},
			},

			Finally: []v1beta1.PipelineTask{
				{
					Name: tasksName[2],
					TaskRef: &v1beta1.TaskRef{
						Name: "set-status",
					},
					Params: []v1beta1.Param{
						{
							Name: "REPO_FULL_NAME",
							Value: v1beta1.ArrayOrString{
								Type:      v1beta1.ParamTypeString,
								StringVal: u.Path,
							},
						},
					},
				},
			},

			Workspaces: []v1beta1.PipelineWorkspaceDeclaration{
				{
					Name: ws,
				},
			},
		},
	}

	return pipeline, nil
}
