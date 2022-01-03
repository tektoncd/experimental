// Copyright 2020 The Tekton Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package builder

import (
	v1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	resource "github.com/tektoncd/pipeline/pkg/apis/resource/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
)

// PipelineRunOp modifies a PipelineRun.
type PipelineRunOp func(*v1beta1.PipelineRun)

// PipelineRunSpecOp is an operation which modify a PipelineRunSpec struct.
type PipelineRunSpecOp func(*v1beta1.PipelineRunSpec)

// PipelineRunStatusOp is an operation which modifies a PipelineRunStatus
type PipelineRunStatusOp func(*v1beta1.PipelineRunStatus)

// PipelineResourceBindingOp is an operation which modify a PipelineResourceBinding struct.
type PipelineResourceBindingOp func(*v1beta1.PipelineResourceBinding)

// PipelineRun creates a new v1beta1 PipelineRun.
func PipelineRun(name, namespace string, ops ...PipelineRunOp) *v1beta1.PipelineRun {
	pr := &v1beta1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1beta1.PipelineRunSpec{},
	}

	for _, op := range ops {
		op(pr)
	}

	return pr
}

// PipelineRunAnnotation adds a annotation to the PipelineRun.
func PipelineRunAnnotation(key, value string) PipelineRunOp {
	return func(pr *v1beta1.PipelineRun) {
		if pr.ObjectMeta.Annotations == nil {
			pr.ObjectMeta.Annotations = map[string]string{}
		}
		pr.ObjectMeta.Annotations[key] = value
	}
}

// PipelineRunSpec sets the PipelineRunSpec, references Pipeline with specified name, to the PipelineRun.
// Any number of PipelineRunSpec modifier can be passed to transform it.
func PipelineRunSpec(name string, ops ...PipelineRunSpecOp) PipelineRunOp {
	return func(pr *v1beta1.PipelineRun) {
		prs := &pr.Spec
		prs.PipelineRef = &v1beta1.PipelineRef{
			Name: name,
		}

		for _, op := range ops {
			op(prs)
		}

		pr.Spec = *prs
	}
}

// PipelineRunStatus sets the PipelineRunStatus to the PipelineRun.
// Any number of PipelineRunStatus modifier can be passed to transform it.
func PipelineRunStatus(ops ...PipelineRunStatusOp) PipelineRunOp {
	return func(pr *v1beta1.PipelineRun) {
		s := &v1beta1.PipelineRunStatus{}
		for _, op := range ops {
			op(s)
		}
		pr.Status = *s
	}
}

// PipelineRunStatusCondition adds a StatusCondition to the TaskRunStatus.
func PipelineRunStatusCondition(condition apis.Condition) PipelineRunStatusOp {
	return func(s *v1beta1.PipelineRunStatus) {
		s.Conditions = append(s.Conditions, condition)
	}
}

// PipelineRunTaskRunsStatus sets the status of TaskRun to the PipelineRunStatus.
func PipelineRunTaskRunsStatus(taskRunName string, status *v1beta1.PipelineRunTaskRunStatus) PipelineRunStatusOp {
	return func(s *v1beta1.PipelineRunStatus) {
		if s.TaskRuns == nil {
			s.TaskRuns = make(map[string]*v1beta1.PipelineRunTaskRunStatus)
		}
		s.TaskRuns[taskRunName] = status
	}
}

// PipelineRunLabel adds a label to the PipelineRun.
func PipelineRunLabel(key, value string) PipelineRunOp {
	return func(pr *v1beta1.PipelineRun) {
		if pr.ObjectMeta.Labels == nil {
			pr.ObjectMeta.Labels = map[string]string{}
		}
		pr.ObjectMeta.Labels[key] = value
	}
}

// PipelineRunResourceBinding adds bindings from actual instances to a Pipeline's declared resources.
func PipelineRunResourceBinding(name string, ops ...PipelineResourceBindingOp) PipelineRunSpecOp {
	return func(prs *v1beta1.PipelineRunSpec) {
		r := &v1beta1.PipelineResourceBinding{
			Name: name,
		}
		for _, op := range ops {
			op(r)
		}
		prs.Resources = append(prs.Resources, *r)
	}
}

// PipelineResourceBindingResourceSpec set the PipelineResourceResourceSpec to the PipelineResourceBinding.
func PipelineResourceBindingResourceSpec(spec *resource.PipelineResourceSpec) PipelineResourceBindingOp {
	return func(b *v1beta1.PipelineResourceBinding) {
		b.ResourceSpec = spec
	}
}
