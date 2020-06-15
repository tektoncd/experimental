// Package generator provides a method to generating Tekton spec
// from simplified configs.
package generator

import (
	v1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

// GithubSpec defines Github spec
type GitHubSpec struct {
	URL   string         `json:"url,omitempty"`
	Steps []v1beta1.Step `json:"steps,omitempty"`
}

// GenerateTask generates Tekton Task spec
// from simplified GithubSpec configs.
func GenerateTask(spec GitHubSpec) (v1beta1.Task, error) {
	var task v1beta1.Task
	task.Spec.Steps = spec.Steps
	task.Spec.Workspaces = append(task.Spec.Workspaces,
		v1beta1.WorkspaceDeclaration{
			Name:      "input",
			MountPath: "/input",
		},
	)
	return task, nil
}
