/*
Copyright 2022 The Tekton Authors
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
		http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package convert

import (
	"encoding/json"
	"fmt"

	"github.com/tektoncd/experimental/workflows/pkg/apis/workflows/v1alpha1"
	"github.com/tektoncd/experimental/workflows/pkg/filters"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/substitution"
	triggersv1beta1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"knative.dev/pkg/kmeta"
	"knative.dev/pkg/ptr"
)

func makeWorkspaces(bindings []v1alpha1.WorkflowWorkspaceBinding) []pipelinev1beta1.WorkspaceBinding {
	if bindings == nil && len(bindings) == 0 {
		return []pipelinev1beta1.WorkspaceBinding{}
	}

	res := []pipelinev1beta1.WorkspaceBinding{}
	for _, b := range bindings {

		// b.Name seems to be populated if we parseYAML
		// but b.WorkspaceBinding.Name is populated if create
		// the struct directly
		if b.WorkspaceBinding.Name == "" {
			b.WorkspaceBinding.Name = b.Name
		}
		res = append(res, b.WorkspaceBinding)
	}
	return res
}

// ToPipelineRun converts a Workflow to a PipelineRun.
func ToPipelineRun(w *v1alpha1.Workflow) (*pipelinev1beta1.PipelineRun, error) {
	saName := "default"
	if w.Spec.ServiceAccountName != nil && *w.Spec.ServiceAccountName != "" {
		saName = *w.Spec.ServiceAccountName
	}

	params := []pipelinev1beta1.Param{}
	for _, ps := range w.Spec.Params {
		params = append(params, pipelinev1beta1.Param{
			Name:  ps.Name,
			Value: *ps.Default,
		})
	}

	pr := pipelinev1beta1.PipelineRun{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PipelineRun",
			APIVersion: pipelinev1beta1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-run-", w.Name),
			Namespace:    w.Namespace, // TODO: Do Runs generated from a Workflow always run in the same namespace
			// TODO: Propagate labels/annotations from Workflows as well?
		},
		Spec: pipelinev1beta1.PipelineRunSpec{
			Params:             params,
			ServiceAccountName: saName,
			Timeouts:           w.Spec.Timeout,
			Workspaces:         makeWorkspaces(w.Spec.Workspaces), // TODO: Add workspaces
		},
	}
	if w.Spec.PipelineRef != nil {
		pr.Spec.PipelineRef = w.Spec.PipelineRef
	} else if w.Spec.PipelineSpec != nil {
		pr.Spec.PipelineSpec = w.Spec.PipelineSpec
	}

	return &pr, nil
}

// ToTriggerTemplate converts a Workflow into a TriggerTemplate
func ToTriggerTemplate(w *v1alpha1.Workflow, replacements map[string]string) (*triggersv1beta1.TriggerTemplate, error) {
	pr, err := ToPipelineRun(w)
	if err != nil {
		return nil, err
	}

	params := []triggersv1beta1.ParamSpec{}
	for _, p := range w.Spec.Params {
		// Triggers does not support array values from bindings
		if p.Type == pipelinev1beta1.ParamTypeArray {
			continue
		}
		defaultVal := substitution.ApplyReplacements(p.Default.StringVal, replacements)

		params = append(params, triggersv1beta1.ParamSpec{
			Name:        p.Name,
			Description: p.Description,
			Default:     ptr.String(defaultVal),
		})
		for i, prp := range pr.Spec.Params {
			if prp.Name == p.Name {
				pr.Spec.Params[i].Value.StringVal = fmt.Sprintf("$(tt.params.%s)", prp.Name)
				pr.Spec.Params[i].Value.Type = pipelinev1beta1.ParamTypeString
			}
		}
	}

	prJson, err := json.Marshal(pr)
	if err != nil {
		return nil, err
	}

	tt := &triggersv1beta1.TriggerTemplate{
		TypeMeta: metav1.TypeMeta{
			Kind:       "TriggerTemplate",
			APIVersion: triggersv1beta1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("tt-%s", w.Name),
			Namespace: w.Namespace,
		},
		Spec: triggersv1beta1.TriggerTemplateSpec{
			Params: params,
			// Look in triggers code base for what this should look like
			ResourceTemplates: []triggersv1beta1.TriggerResourceTemplate{{
				RawExtension: runtime.RawExtension{
					Raw: prJson,
				},
			}},
		},
	}

	return tt, nil
}

// ToTriggers creates a new Trigger with inline bindings and template for each type
// TODO: Reuse same triggertemplate for efficiency?
func ToTriggers(w *v1alpha1.Workflow, grs []*v1alpha1.GitRepository) ([]*triggersv1beta1.Trigger, error) {
	repoMap := getRepoMap(grs)
	replacements := getRepoParamReplacements(repoMap)
	tt, err := ToTriggerTemplate(w, replacements)
	if err != nil {
		return nil, err
	}
	triggers := []*triggersv1beta1.Trigger{}
	for _, t := range w.Spec.Triggers {
		interceptors := []*triggersv1beta1.TriggerInterceptor{}

		// TODO: Ideally we'd like to filter for the correct event types,
		// but the Knative GithubSource doesn't preserve GitHub headers,
		// which is what the Triggers GitHub interceptor uses to determine event types.
		// So for now, all the workflows will receive all the events for a repo.
		if t.Event != nil && t.Event.Source.Repo != "" {
			repoName := t.Event.Source.Repo
			url := repoMap[repoName].Spec.URL
			repoFilter, err := getRepoFilter(url)
			if err != nil {
				return nil, err
			}
			interceptors = append(interceptors, repoFilter)
		}

		filterInterceptors, err := filters.ToInterceptors(t.Filters)
		if err != nil {
			return nil, err
		}
		interceptors = append(interceptors, filterInterceptors...)
		triggers = append(triggers, &triggersv1beta1.Trigger{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Trigger",
				APIVersion: triggersv1beta1.SchemeGroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-%s", w.Name, t.Name),
				Namespace: w.Namespace,
				Labels: map[string]string{
					v1alpha1.WorkflowLabelKey: w.Name,             // Used by the controller to list Triggers belonging to this workflow
					"managed-by":              "tekton-workflows", // Used by the workflows EL to select Triggers
				},
				OwnerReferences: []metav1.OwnerReference{*kmeta.NewControllerRef(w)},
			},
			Spec: triggersv1beta1.TriggerSpec{
				Bindings: t.Bindings,
				Template: triggersv1beta1.TriggerSpecTemplate{
					Spec: &tt.Spec,
				},
				Name:         t.Name,
				Interceptors: interceptors,
			},
		})
	}
	return triggers, nil
}

func getRepoMap(grs []*v1alpha1.GitRepository) map[string]*v1alpha1.GitRepository {
	out := map[string]*v1alpha1.GitRepository{}
	for _, repo := range grs {
		out[repo.Name] = repo
	}
	return out
}

func getRepoFilter(url string) (*triggersv1beta1.TriggerInterceptor, error) {
	// For right now we're assuming that the event body
	// contains a "repository.html_url" field, as in GitHub event payloads
	celFilter := fmt.Sprintf("body.repository.html_url.matches('%s')", url)
	celFilterToJSON, err := filters.ToV1JSON(celFilter)
	if err != nil {
		return nil, err
	}
	repoInterceptor := triggersv1beta1.TriggerInterceptor{
		Name: ptr.String("repo"),
		Ref: triggersv1beta1.InterceptorRef{
			Name: "cel",
			Kind: "ClusterInterceptor",
		},
		Params: []triggersv1beta1.InterceptorParams{{
			Name:  "filter",
			Value: celFilterToJSON,
		}},
	}
	return &repoInterceptor, nil
}

func getRepoParamReplacements(repos map[string]*v1alpha1.GitRepository) map[string]string {
	repoPatterns := []string{
		"repos.%s.url",
		"repos[%s].url",
		"repos['%s'].url",
	}
	replacements := map[string]string{}
	for _, pattern := range repoPatterns {
		for name, repo := range repos {
			replacements[fmt.Sprintf(pattern, name)] = repo.Spec.URL
		}
	}
	return replacements
}
