package pipelineinpod

import (
	"context"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	pipelineclient "github.com/tektoncd/pipeline/pkg/client/injection/client"
	run "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1alpha1/run"
	v1alpha1run "github.com/tektoncd/pipeline/pkg/client/injection/reconciler/pipeline/v1alpha1/run"
	tkncontroller "github.com/tektoncd/pipeline/pkg/controller"
	"k8s.io/client-go/tools/cache"
	kubeclient "knative.dev/pkg/client/injection/kube/client"
	filteredpodinformer "knative.dev/pkg/client/injection/kube/informers/core/v1/pod/filtered"
	configmap "knative.dev/pkg/configmap"
	controller "knative.dev/pkg/controller"
	logging "knative.dev/pkg/logging"
)

const (
	ControllerName = "pipelineinpod-controller"
	kind           = "Run"
)

func NewController(opts *pipeline.Options) func(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
	return func(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
		logger := logging.FromContext(ctx)

		logger.Info("Setting up event handlers")

		pipelineClientSet := pipelineclient.Get(ctx)
		runInformer := run.Get(ctx)
		kubeClientSet := kubeclient.Get(ctx)
		podInformer := filteredpodinformer.Get(ctx, v1beta1.ManagedByLabelKey)
		entrypointCache, err := NewEntrypointCache(kubeClientSet)
		if err != nil {
			logger.Fatalf("Error creating entrypoint cache: %v", err)
		}

		r := &Reconciler{
			pipelineClientSet: pipelineClientSet,
			kubeClientSet:     kubeClientSet,
			Images:            opts.Images,
			entrypointCache:   entrypointCache,
		}
		impl := v1alpha1run.NewImpl(ctx, r, func(impl *controller.Impl) controller.Options {
			return controller.Options{
				AgentName: ControllerName,
			}
		})

		logger.Info("Setting up event handlers")

		// Add event handler for Runs
		runInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
			FilterFunc: tkncontroller.FilterRunRef(v1alpha1.SchemeGroupVersion.String(), "ColocatedPipelineRun"),
			Handler:    controller.HandleAll(impl.Enqueue),
		})

		podInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
			FilterFunc: controller.FilterController(&v1alpha1.Run{}),
			Handler:    controller.HandleAll(impl.EnqueueControllerOf),
		})

		return impl
	}
}
