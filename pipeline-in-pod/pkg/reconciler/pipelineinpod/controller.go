package pipelineinpod

import (
	context "context"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	pipelineclient "github.com/tektoncd/pipeline/pkg/client/injection/client"
	pipelineruninformer "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1beta1/pipelinerun"
	pipelinerunreconciler "github.com/tektoncd/pipeline/pkg/client/injection/reconciler/pipeline/v1beta1/pipelinerun"
	"github.com/tektoncd/pipeline/pkg/pod"
	"k8s.io/client-go/tools/cache"
	kubeclient "knative.dev/pkg/client/injection/kube/client"
	filteredpodinformer "knative.dev/pkg/client/injection/kube/informers/core/v1/pod/filtered"
	configmap "knative.dev/pkg/configmap"
	controller "knative.dev/pkg/controller"
	logging "knative.dev/pkg/logging"
)

const (
	ControllerName = "pipelineinpod-controller"
	kind           = "PipelineInPod"
)

func NewController(opts *pipeline.Options) func(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
	return func(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
		logger := logging.FromContext(ctx)

		pipelineClientSet := pipelineclient.Get(ctx)
		kubeClientSet := kubeclient.Get(ctx)
		pipelineRunInformer := pipelineruninformer.Get(ctx)
		podInformer := filteredpodinformer.Get(ctx, v1beta1.ManagedByLabelKey)
		entrypointCache, err := pod.NewEntrypointCache(kubeClientSet)
		if err != nil {
			logger.Fatalf("Error creating entrypoint cache: %v", err)
		}

		r := &Reconciler{
			pipelineClientSet: pipelineClientSet,
			kubeClientSet:     kubeClientSet,
			pipelineRunLister: pipelineRunInformer.Lister(),
			Images:            opts.Images,
			entrypointCache:   entrypointCache,
		}

		impl := pipelinerunreconciler.NewImpl(ctx, r, func(impl *controller.Impl) controller.Options {
			return controller.Options{
				AgentName: ControllerName,
			}
		})

		logger.Info("Setting up event handlers")

		pipelineRunInformer.Informer().AddEventHandler(controller.HandleAll(impl.Enqueue))

		podInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
			FilterFunc: controller.FilterController(&v1beta1.PipelineRun{}),
			Handler:    controller.HandleAll(impl.EnqueueControllerOf),
		})

		return impl
	}
}
