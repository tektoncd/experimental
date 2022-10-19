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
	"time"

	fluxnotifications "github.com/fluxcd/notification-controller/api/v1beta1"
	fluxmeta "github.com/fluxcd/pkg/apis/meta"
	fluxsource "github.com/fluxcd/source-controller/api/v1beta2"
	"github.com/tektoncd/experimental/workflows/pkg/apis/workflows/v1alpha1"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	triggersv1beta1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"knative.dev/pkg/kmeta"
	"knative.dev/pkg/ptr"
)

var fluxCELFilter []byte

const FluxNamespace = "flux-system"

func init() {
	var err error
	fluxCELFilter, err = json.Marshal("header.canonical('Gotk-Component') == 'source-controller' && body.involvedObject.kind == 'GitRepository'")
	if err != nil {
		panic(err)
	}
}

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
	for _, t := range w.Spec.Triggers {
		payloadValidation := triggersv1beta1.TriggerInterceptor{
			Name: ptr.String("filter-flux-events"),
			Ref: triggersv1beta1.InterceptorRef{
				Name: "cel",
				Kind: "ClusterInterceptor",
			},
			Params: []triggersv1beta1.InterceptorParams{{
				Name:  "filter",
				Value: v1.JSON{Raw: fluxCELFilter},
			}},
		}
		interceptors := []*triggersv1beta1.TriggerInterceptor{&payloadValidation}
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
				Bindings: t.Bindings, // Problem: Event body from flux != event body from github -> User specified bindings will not work
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

type FluxResources struct {
	Repos     []fluxsource.GitRepository
	Receivers []fluxnotifications.Receiver
	Alert     *fluxnotifications.Alert
	Provider  *fluxnotifications.Provider
}

func GetFluxResources(w *v1alpha1.Workflow) (FluxResources, error) {
	out := FluxResources{}
	repos := map[string]v1alpha1.Repo{}
	for _, r := range w.Spec.Repos {
		repos[r.Name] = r
	}
	var fluxRepos []fluxsource.GitRepository
	var fluxReceivers []fluxnotifications.Receiver
	// TODO: single flux provider for workflows; one flux alert per workflow in tekton-workflows namespace?
	// TODO: It's unclear which objects need to go in the flux-system namespace-- for now we'll put everything there and forgo owner references
	// TODO: since these are all going in the flux-system namespace, their names will need to depend on the workflow namespace
	fluxProvider := fluxnotifications.Provider{ObjectMeta: metav1.ObjectMeta{Name: w.Name, Namespace: FluxNamespace},
		TypeMeta: metav1.TypeMeta{Kind: "Provider", APIVersion: "notification.toolkit.fluxcd.io/v1beta1"},
		Spec:     fluxnotifications.ProviderSpec{Type: "generic", Address: "http://el-workflows-listener.tekton-workflows.svc.cluster.local:8080/"}}
	var fluxAlert = fluxnotifications.Alert{ObjectMeta: metav1.ObjectMeta{Name: w.Name, Namespace: FluxNamespace},
		TypeMeta: metav1.TypeMeta{Kind: "Alert", APIVersion: "notification.toolkit.fluxcd.io/v1beta1"},
		Spec:     fluxnotifications.AlertSpec{ProviderRef: fluxmeta.LocalObjectReference{Name: fluxProvider.Name}, EventSeverity: "info"}}
	// TODO: handle multiple triggers w/ same event source and filter but different event types
	for _, t := range w.Spec.Triggers {
		if t.Event == nil {
			continue
		}
		r, ok := repos[t.Event.Source.Repo]
		if !ok {
			return out, fmt.Errorf("unsupported event source %s", t.Event.Source.Repo) // TODO: handle this in validation code
		}

		repo := fluxsource.GitRepository{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s-%s", w.Name, r.Name), Namespace: FluxNamespace},
			TypeMeta: metav1.TypeMeta{Kind: "GitRepository", APIVersion: "source.toolkit.fluxcd.io/v1beta2"},
			Spec: fluxsource.GitRepositorySpec{
				URL: r.URL, Interval: metav1.Duration{Duration: time.Minute},
				// Flux requires you to specify Branch, Tag, or Commit, so we are assuming the GitRef is a branch
				Reference: &fluxsource.GitRepositoryRef{Branch: t.Filters.GitRef.Regex},
			}}
		if r.SecretRef != "" {
			repo.Spec.SecretRef = &fluxmeta.LocalObjectReference{Name: r.SecretRef}
		}
		fluxRepos = append(fluxRepos, repo)
		receiver := fluxnotifications.Receiver{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s-%s", w.Name, r.Name), Namespace: FluxNamespace},
			TypeMeta: metav1.TypeMeta{Kind: "Receiver", APIVersion: "notification.toolkit.fluxcd.io/v1beta1"},
			Spec: fluxnotifications.ReceiverSpec{Type: r.VCSType, Events: eventTypes(t.Event.Types), SecretRef: fluxmeta.LocalObjectReference{Name: t.Event.Secret.SecretName},
				Resources: []fluxnotifications.CrossNamespaceObjectReference{{
					Kind:      "GitRepository",
					Name:      repo.Name,
					Namespace: repo.Namespace,
				}}}}
		fluxReceivers = append(fluxReceivers, receiver)
		fluxAlert.Spec.EventSources = append(fluxAlert.Spec.EventSources, fluxnotifications.CrossNamespaceObjectReference{
			Kind:      "GitRepository",
			Name:      repo.Name,
			Namespace: repo.Namespace,
		})
	}
	out.Repos = fluxRepos
	out.Receivers = fluxReceivers
	out.Alert = &fluxAlert
	out.Provider = &fluxProvider
	return out, nil
}

func eventTypes(ets []v1alpha1.EventType) []string {
	var out []string
	for _, t := range ets {
		out = append(out, string(t))
	}
	return out
}
