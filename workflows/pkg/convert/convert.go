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
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	triggersv1beta1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
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
			Namespace:    w.Namespace, // TODO: Do Runs generate from a Workflow always run in the same namespace
			// TODO: Propagate labels/annotations from Workflows as well?
		},
		Spec: pipelinev1beta1.PipelineRunSpec{
			Params:             params,
			ServiceAccountName: saName,
			Timeouts:           w.Spec.Timeout,
			Workspaces:         makeWorkspaces(w.Spec.Workspaces), // TODO: Add workspaces
		},
	}

	// Add in pipelineSpec or pipelineRef (via git resolver)
	if w.Spec.Pipeline.Git.URL != "" {
		gitConfig := w.Spec.Pipeline.Git
		// TODO: This assumes each field is specified. Support defaults as well
		pr.Spec.PipelineRef = &pipelinev1beta1.PipelineRef{
			ResolverRef: pipelinev1beta1.ResolverRef{
				Resolver: "git",
				Params: []pipelinev1beta1.Param{{
					Name:  "url",
					Value: *pipelinev1beta1.NewArrayOrString(gitConfig.URL),
				}, {
					Name:  "revision",
					Value: *pipelinev1beta1.NewArrayOrString(gitConfig.Revision),
				}, {
					Name:  "pathInRepo",
					Value: *pipelinev1beta1.NewArrayOrString(gitConfig.PathInRepo),
				}},
			},
		}
	} else {
		pr.Spec.PipelineSpec = &w.Spec.Pipeline.Spec
	}

	return &pr, nil
}

// ToTriggerTemplate converts a Workflow into a TriggerTemplate
func ToTriggerTemplate(w *v1alpha1.Workflow) (*triggersv1beta1.TriggerTemplate, error) {
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

		params = append(params, triggersv1beta1.ParamSpec{
			Name:        p.Name,
			Description: p.Description,
			Default:     ptr.String(p.Default.StringVal),
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
func ToTriggers(w *v1alpha1.Workflow) ([]*triggersv1beta1.Trigger, error) {
	tt, err := ToTriggerTemplate(w)
	if err != nil {
		return nil, err
	}
	triggers := []*triggersv1beta1.Trigger{}
	for i, t := range w.Spec.Triggers {
		name := t.Name
		if name == "" {
			// FIXME: What if user re-orders triggers
			// Name field should always exist -> Add it in defautls
			name = fmt.Sprintf("%d", i)
		}

		secretToJson, err := ToV1JSON(t.Event.Secret)
		if err != nil {
			return nil, err
		}
		eventTypesJson, err := ToV1JSON([]string{t.Event.Type})
		if err != nil {
			return nil, err
		}
		// Add an interceptor to validate the payload from GitHub webhook
		payloadValidation := triggersv1beta1.TriggerInterceptor{
			Name: ptr.String("validate-webhook"),
			Ref: triggersv1beta1.InterceptorRef{
				Name: "github",
				Kind: "ClusterInterceptor",
			},
			Params: []triggersv1beta1.InterceptorParams{{
				Name:  "secretRef",
				Value: secretToJson,
			}, {
				Name:  "eventTypes",
				Value: eventTypesJson,
			}},
		}
		interceptors := []*triggersv1beta1.TriggerInterceptor{&payloadValidation}
		interceptors = append(interceptors, t.Interceptors...)

		triggers = append(triggers, &triggersv1beta1.Trigger{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Trigger",
				APIVersion: triggersv1beta1.SchemeGroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-%s", w.Name, name),
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
				Name:         name,
				Interceptors: interceptors,
			},
		})
	}
	return triggers, nil
}

// ToV1JSON is a wrapper around json.Marshal to easily convert to the Kubernetes apiextensionsv1.JSON type
func ToV1JSON(v interface{}) (v1.JSON, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return v1.JSON{}, err
	}
	return v1.JSON{
		Raw: b,
	}, nil
}
