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
	"fmt"

	"github.com/tektoncd/experimental/workflows/pkg/apis/workflows/v1alpha1"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
