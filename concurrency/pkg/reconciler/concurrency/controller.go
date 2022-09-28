package concurrency

import (
	"context"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline"
	pipelineruninformer "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1beta1/pipelinerun"
	pipelinerunreconciler "github.com/tektoncd/pipeline/pkg/client/injection/reconciler/pipeline/v1beta1/pipelinerun"
	configmap "knative.dev/pkg/configmap"
	controller "knative.dev/pkg/controller"
	logging "knative.dev/pkg/logging"
)

const (
	ControllerName = "concurrency-controller"
)

func NewController(opts *pipeline.Options) func(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
	return func(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
		logger := logging.FromContext(ctx)
		pipelineRunInformer := pipelineruninformer.Get(ctx)

		r := &Reconciler{}
		impl := pipelinerunreconciler.NewImpl(ctx, r, func(impl *controller.Impl) controller.Options {
			return controller.Options{
				AgentName: ControllerName,
			}
		})

		logger.Info("Setting up event handlers")
		pipelineRunInformer.Informer().AddEventHandler(controller.HandleAll(impl.Enqueue))

		return impl
	}
}
