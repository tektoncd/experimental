/*
Copyright 2020 The Tekton Authors

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

package sample

import (
	"context"

	runinformer "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1alpha1/run"
	runreconciler "github.com/tektoncd/pipeline/pkg/client/injection/reconciler/pipeline/v1alpha1/run"
	pipelinecontroller "github.com/tektoncd/pipeline/pkg/controller"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
)

const (
	// CHANGE THESE CONSTANTS TO APPROPRIATE VALUES TO DEFINE YOUR CUSTOM TASK.
	// API group for your custom task
	groupName = "sample.tekton.dev"
	// version for your custom task
	version = "v1alpha1"
	// kind for your custom task
	kind = "Example"
)

// SchemeGroupVersion is group version used to register these objects
var schemeGroupVersion = schema.GroupVersion{Group: groupName, Version: version}

// NewController initializes the controller and registers event handlers to enqueue events.
func NewController(
	ctx context.Context,
	cmw configmap.Watcher,
) *controller.Impl {

	runInformer := runinformer.Get(ctx)

	r := &Reconciler{}

	impl := runreconciler.NewImpl(ctx, r, func(impl *controller.Impl) controller.Options {
		return controller.Options{
			AgentName: "sample-custom-task",
		}
	})

	logging.FromContext(ctx).Info("Setting up event handlers")

	// Add event handler for Runs
	runInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: pipelinecontroller.FilterRunRef(schemeGroupVersion.String(), kind),
		Handler:    controller.HandleAll(impl.Enqueue),
	})

	return impl
}
