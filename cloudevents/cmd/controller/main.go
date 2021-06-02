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

package main

import (
	"context"
	"github.com/tektoncd/pipeline/pkg/apis/config"
	"k8s.io/client-go/tools/cache"
	"knative.dev/pkg/logging"

	"github.com/tektoncd/experimental/cloudevents/pkg/reconciler"
	pipelineruninformer "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1beta1/pipelinerun"
 	pipelinerunreconciler "github.com/tektoncd/pipeline/pkg/client/injection/reconciler/pipeline/v1beta1/pipelinerun"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/injection/sharedmain"
)

const pipelineRunControllerName = "cloudevents-pipelinerun-controller"

func main() {
	sharedmain.Main(pipelineRunControllerName, newPipelineRunController)
}

func newPipelineRunController(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
	logger := logging.FromContext(ctx)
	pipelineRunInformer := pipelineruninformer.Get(ctx)

	c := reconciler.NewController(ctx)
	impl := pipelinerunreconciler.NewImpl(ctx, c, func(impl *controller.Impl) controller.Options {
		configStore := config.NewStore(logger.Named("config-store"))
		return controller.Options{
			AgentName: "cloudevents-pipelinerun-controller",
			ConfigStore: configStore,
		}
	})

	pipelineRunInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    impl.Enqueue,
		UpdateFunc: controller.PassNew(impl.Enqueue),
		DeleteFunc: impl.Enqueue,
	})

	return impl
}
