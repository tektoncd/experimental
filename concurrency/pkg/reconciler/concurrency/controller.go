package concurrency

import (
	"context"

	concurrencycontrolinformer "github.com/tektoncd/experimental/concurrency/pkg/client/injection/informers/concurrency/v1alpha1/concurrencycontrol"
	pipelineclient "github.com/tektoncd/pipeline/pkg/client/injection/client"
	pipelineruninformer "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1beta1/pipelinerun"
	pipelinerunreconciler "github.com/tektoncd/pipeline/pkg/client/injection/reconciler/pipeline/v1beta1/pipelinerun"
	configmap "knative.dev/pkg/configmap"
	controller "knative.dev/pkg/controller"
	logging "knative.dev/pkg/logging"
)

const (
	ControllerName = "concurrency-controller"
)

func NewController() func(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
	return func(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
		logger := logging.FromContext(ctx)
		pipelineClientSet := pipelineclient.Get(ctx)
		pipelineRunInformer := pipelineruninformer.Get(ctx)
		concurrencyControlInformer := concurrencycontrolinformer.Get(ctx)

		r := &Reconciler{
			ConcurrencyControlLister: concurrencyControlInformer.Lister(),
			PipelineRunLister:        pipelineRunInformer.Lister(),
			PipelineClientSet:        pipelineClientSet,
		}
		impl := pipelinerunreconciler.NewImpl(ctx, r, func(impl *controller.Impl) controller.Options {
			return controller.Options{
				AgentName:         ControllerName,
				SkipStatusUpdates: true, // Don't update PipelineRun status. This is the responsibility of Tekton Pipelines
			}
		})

		logger.Info("Setting up event handlers")
		pipelineRunInformer.Informer().AddEventHandler(controller.HandleAll(impl.Enqueue))

		return impl
	}
}
