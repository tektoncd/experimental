// Package generator provides a method to generating Tekton spec
// from simplified configs.
package generator

import (
	"fmt"
	"net/url"

	v1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// GitHub defines Github fields
type GitHub struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              GitHubSpec `json:"spec"`
}

// GithubSpec defines Github spec
type GitHubSpec struct {
	URL                string         `json:"url,omitempty"`
	Revision           string         `json:"revision,omitempty"`
	Branch             string         `json:"branch,omitempty"`
	Storage            string         `json:"storage,omitempty"`
	SecretName         string         `json:"secretName,omitempty"`
	SecretKey          string         `json:"secretKey,omitempty"`
	ServiceAccountName string         `json:"serviceAccountName,omitempty"`
	Steps              []v1beta1.Step `json:"steps,omitempty"`
}

type trigger struct {
	TriggerBinding  []*v1alpha1.TriggerBinding
	TriggerTemplate *v1alpha1.TriggerTemplate
	EventListener   *v1alpha1.EventListener
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
	name := github.Name + "-pipeline"
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
			Params: []v1beta1.ParamSpec{
				{
					Name: "gitrepositoryurl",
					Type: v1beta1.ParamTypeString,
				},

				{
					Name: "gitrevision",
					Type: v1beta1.ParamTypeString,
				},
			},
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
						{
							Name: "revision",
							Value: v1beta1.ArrayOrString{
								Type:      v1beta1.ParamTypeString,
								StringVal: "$(params.gitrevision)",
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
						Name: "github-set-status",
					},
					Params: []v1beta1.Param{
						{
							Name: "REPO_FULL_NAME",
							Value: v1beta1.ArrayOrString{
								Type:      v1beta1.ParamTypeString,
								StringVal: u.Path,
							},
						},
						{
							Name: "SHA",
							Value: v1beta1.ArrayOrString{
								Type:      v1beta1.ParamTypeString,
								StringVal: "$(params.gitrevision)",
							},
						},
						{
							Name: "STATE",
							Value: v1beta1.ArrayOrString{
								Type:      v1beta1.ParamTypeString,
								StringVal: "success",
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

// Generate the trigger with the given generated pipeline
func GenerateTrigger(p *v1beta1.Pipeline, g *GitHub) *trigger {
	if p.Namespace == "" {
		p.Namespace = "default"
	}

	var value int64
	var format resource.Format
	if g.Spec.Storage != "" {
		diskSize := resource.MustParse(g.Spec.Storage)
		value = diskSize.Value()
		format = diskSize.Format
	} else {
		value = 1024 * 1024 * 1024
		format = resource.BinarySI
	}

	if g.Spec.Branch == "" {
		g.Spec.Branch = "master"
	}

	// create pipelinerun in the triggertemplate
	pr := &v1beta1.PipelineRun{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1beta1.SchemeGroupVersion.String(),
			Kind:       "PipelineRun",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    p.Namespace,
			GenerateName: p.Name + "-run-",
			Labels:       p.Labels,
		},
		Spec: v1beta1.PipelineRunSpec{
			PipelineRef: &v1beta1.PipelineRef{
				Name: p.Name,
			},
			Params: []v1beta1.Param{
				{
					Name: "gitrepositoryurl",
					Value: v1beta1.ArrayOrString{
						Type:      v1beta1.ParamTypeString,
						StringVal: "$(tt.params.gitrepositoryurl)",
					},
				},
				{
					Name: "gitrevision",
					Value: v1beta1.ArrayOrString{
						Type:      v1beta1.ParamTypeString,
						StringVal: "$(tt.params.gitrevision)",
					},
				},
			},
			Workspaces: []v1beta1.WorkspaceBinding{
				{
					Name: p.Spec.Workspaces[0].Name,
					VolumeClaimTemplate: &corev1.PersistentVolumeClaim{
						Spec: corev1.PersistentVolumeClaimSpec{
							AccessModes: []corev1.PersistentVolumeAccessMode{
								corev1.ReadWriteOnce,
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									"storage": *resource.NewQuantity(value, format),
								},
							},
						},
					},
				},
			},
		},
	}

	// create the triggertemplate
	tt := &v1alpha1.TriggerTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: p.Namespace,
			Name:      p.Name + "-triggertemplate",
			Labels:    p.Labels,
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
			Kind:       "TriggerTemplate",
		},
		Spec: v1alpha1.TriggerTemplateSpec{
			Params: []v1alpha1.ParamSpec{
				{
					Name:        "gitrevision",
					Description: "The git revision",
				},
				{
					Name:        "gitrepositoryurl",
					Description: "The git repository url",
				},
			},
			ResourceTemplates: []v1alpha1.TriggerResourceTemplate{
				{
					runtime.RawExtension{Object: pr},
				},
			},
		},
	}

	// create the triggerbinding for pull request events
	tbPr := &v1alpha1.TriggerBinding{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: p.Namespace,
			Name:      p.Name + "-pr-triggerbinding",
			Labels:    p.Labels,
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
			Kind:       "TriggerBinding",
		},
		Spec: v1alpha1.TriggerBindingSpec{
			Params: []v1alpha1.Param{
				{
					Name:  "gitrevision",
					Value: "$(body.head.sha)",
				},
				{
					Name:  "gitrepositoryurl",
					Value: "$(body.repository.url)",
				},
			},
		},
	}

	// create the triggerbinding for pushes events
	tbPush := &v1alpha1.TriggerBinding{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: p.Namespace,
			Name:      p.Name + "-push-triggerbinding",
			Labels:    p.Labels,
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
			Kind:       "TriggerBinding",
		},
		Spec: v1alpha1.TriggerBindingSpec{
			Params: []v1alpha1.Param{
				{
					Name:  "gitrevision",
					Value: "$(body.after)",
				},
				{
					Name:  "gitrepositoryurl",
					Value: "$(body.repository.url)",
				},
			},
		},
	}

	// create the eventlistener
	el := &v1alpha1.EventListener{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: p.Namespace,
			Name:      p.Name + "-eventlistener",
			Labels:    p.Labels,
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
			Kind:       "EventListener",
		},
		Spec: v1alpha1.EventListenerSpec{
			ServiceAccountName: g.Spec.ServiceAccountName,
			Triggers: []v1alpha1.EventListenerTrigger{
				{
					Name: "github-push",
					Interceptors: []*v1alpha1.EventInterceptor{
						{
							GitHub: &v1alpha1.GitHubInterceptor{
								EventTypes: []string{"push"},
								SecretRef: &v1alpha1.SecretRef{
									SecretKey:  g.Spec.SecretKey,
									SecretName: g.Spec.SecretName,
								},
							},
						},
						{
							CEL: &v1alpha1.CELInterceptor{
								Filter: "body.ref.split('/')[2] == " + g.Spec.Branch,
							},
						},
					},
					Bindings: []*v1alpha1.EventListenerBinding{
						{
							Ref: tbPush.Name,
						},
					},
					Template: v1alpha1.EventListenerTemplate{
						Name: tt.Name,
					},
				},
				{
					Name: "github-pull-request",
					Interceptors: []*v1alpha1.EventInterceptor{
						{
							GitHub: &v1alpha1.GitHubInterceptor{
								EventTypes: []string{"pull_request"},
								SecretRef: &v1alpha1.SecretRef{
									SecretKey:  g.Spec.SecretKey,
									SecretName: g.Spec.SecretName,
								},
							},
						},
						{
							CEL: &v1alpha1.CELInterceptor{
								Filter: "body.base.ref == " + g.Spec.Branch,
							},
						},
					},
					Bindings: []*v1alpha1.EventListenerBinding{
						{
							Ref: tbPr.Name,
						},
					},
					Template: v1alpha1.EventListenerTemplate{
						Name: tt.Name,
					},
				},
			},
		},
	}

	trigger := &trigger{
		TriggerBinding:  []*v1alpha1.TriggerBinding{tbPush, tbPr},
		TriggerTemplate: tt,
		EventListener:   el,
	}
	return trigger
}
