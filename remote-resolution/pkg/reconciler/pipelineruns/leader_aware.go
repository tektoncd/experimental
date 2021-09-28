/*
Copyright 2021 The Tekton Authors

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

package pipelineruns

import (
	"github.com/tektoncd/pipeline/pkg/client/listers/pipeline/v1beta1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"knative.dev/pkg/reconciler"
)

// buildTaskRunLeaderAwareFuncs constructs a LeaderAwareFuncs object to embed in a
// TaskRun-aware controller so that it can meet knative's controller.LeaderAware interface.
func buildTaskRunLeaderAwareFuncs(lister v1beta1.TaskRunLister) reconciler.LeaderAwareFuncs {
	return reconciler.LeaderAwareFuncs{
		PromoteFunc: func(bkt reconciler.Bucket, enq func(reconciler.Bucket, types.NamespacedName)) error {
			all, err := lister.List(labels.Everything())
			if err != nil {
				return err
			}
			for _, elt := range all {
				enq(bkt, types.NamespacedName{
					Namespace: elt.GetNamespace(),
					Name:      elt.GetName(),
				})
			}
			return nil
		},
	}
}

// buildPipelineRunLeaderAwareFuncs constructs a LeaderAwareFuncs object to embed in a
// PipelineRun-aware controller so that it can meet knative's controller.LeaderAware interface.
func buildPipelineRunLeaderAwareFuncs(lister v1beta1.PipelineRunLister) reconciler.LeaderAwareFuncs {
	return reconciler.LeaderAwareFuncs{
		PromoteFunc: func(bkt reconciler.Bucket, enq func(reconciler.Bucket, types.NamespacedName)) error {
			all, err := lister.List(labels.Everything())
			if err != nil {
				return err
			}
			for _, elt := range all {
				enq(bkt, types.NamespacedName{
					Namespace: elt.GetNamespace(),
					Name:      elt.GetName(),
				})
			}
			return nil
		},
	}
}
