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

package pip

import (
	context "context"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	pipelineclient "github.com/tektoncd/pipeline/pkg/client/injection/client"
	run "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1alpha1/run"
	pipelineruninformer "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1beta1/pipelinerun"
	v1alpha1run "github.com/tektoncd/pipeline/pkg/client/injection/reconciler/pipeline/v1alpha1/run"
	pipelinecontroller "github.com/tektoncd/pipeline/pkg/controller"
	tkncontroller "github.com/tektoncd/pipeline/pkg/controller"
	"k8s.io/client-go/tools/cache"
	configmap "knative.dev/pkg/configmap"
	controller "knative.dev/pkg/controller"
	logging "knative.dev/pkg/logging"
)

const (
	ControllerName = "pip-controller"
	kind           = "Pipeline"
)

// NewController creates a Reconciler for Run and returns the result of NewImpl.
func NewController(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
	logger := logging.FromContext(ctx)

	pipelineClientSet := pipelineclient.Get(ctx)
	runInformer := run.Get(ctx)
	pipelineRunInformer := pipelineruninformer.Get(ctx)

	r := &Reconciler{
		pipelineClientSet: pipelineClientSet,
		runLister:         runInformer.Lister(),
		pipelineRunLister: pipelineRunInformer.Lister(),
	}

	impl := v1alpha1run.NewImpl(ctx, r, func(impl *controller.Impl) controller.Options {
		return controller.Options{
			AgentName: ControllerName,
		}
	})

	logger.Info("Setting up event handlers")

	// Add event handler for Runs
	runInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: tkncontroller.FilterRunRef(v1beta1.SchemeGroupVersion.String(), kind),
		Handler:    controller.HandleAll(impl.Enqueue),
	})

	// Add event handler for PipelineRuns controlled by Run
	pipelineRunInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: pipelinecontroller.FilterOwnerRunRef(runInformer.Lister(), v1beta1.SchemeGroupVersion.String(), kind),
		Handler:    controller.HandleAll(impl.EnqueueControllerOf),
	})

	return impl
}
