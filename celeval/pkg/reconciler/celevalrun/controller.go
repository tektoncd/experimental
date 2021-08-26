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

package celevalrun

import (
	"context"
	"github.com/tektoncd/experimental/celeval/pkg/apis/celeval"
	celevalv1alpha1 "github.com/tektoncd/experimental/celeval/pkg/apis/celeval/v1alpha1"
	celevalclient "github.com/tektoncd/experimental/celeval/pkg/client/injection/client"
	celevalinformer "github.com/tektoncd/experimental/celeval/pkg/client/injection/informers/celeval/v1alpha1/celeval"
	pipelineclient "github.com/tektoncd/pipeline/pkg/client/injection/client"
	runinformer "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1alpha1/run"
	runreconciler "github.com/tektoncd/pipeline/pkg/client/injection/reconciler/pipeline/v1alpha1/run"
	pipelinecontroller "github.com/tektoncd/pipeline/pkg/controller"
	"k8s.io/client-go/tools/cache"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
)

// NewController instantiates a new controller.Impl from knative.dev/pkg/controller
func NewController(namespace string) func(context.Context, configmap.Watcher) *controller.Impl {
	return func(ctx context.Context, cmw configmap.Watcher) *controller.Impl {

		pipelineclientset := pipelineclient.Get(ctx)
		celevalclientset := celevalclient.Get(ctx)
		runInformer := runinformer.Get(ctx)
		celevalinformer := celevalinformer.Get(ctx)

		r := &Reconciler{
			pipelineClientSet: pipelineclientset,
			celEvalClientSet:  celevalclientset,
			runLister:         runInformer.Lister(),
			celEvalLister:     celevalinformer.Lister(),
		}

		impl := runreconciler.NewImpl(ctx, r, func(impl *controller.Impl) controller.Options {
			return controller.Options{
				AgentName: celeval.ControllerName,
			}
		})

		// Add event handler for Runs
		runInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
			FilterFunc: pipelinecontroller.FilterRunRef(celevalv1alpha1.SchemeGroupVersion.String(), celeval.ControllerName),
			Handler:    controller.HandleAll(impl.Enqueue),
		})

		return impl
	}
}
