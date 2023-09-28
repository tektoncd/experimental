package pipelinerun

import (
	"context"

	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/injection"

	"github.com/tektoncd/experimental/metrics-operator/pkg/metrics"
	pipelineruninformer "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1beta1/pipelinerun"
	pipelinerunreconciler "github.com/tektoncd/pipeline/pkg/client/injection/reconciler/pipeline/v1beta1/pipelinerun"
)

func NewController(manager *metrics.MetricManager) injection.ControllerConstructor {
	return func(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
		pipelineRunInformer := pipelineruninformer.Get(ctx)

		c := &Reconciler{
			manager: manager,
		}

		impl := pipelinerunreconciler.NewImpl(ctx, c, func(impl *controller.Impl) controller.Options {
			return controller.Options{
				FinalizerName:     "pipelinerun.metrics.tekton.dev",
				SkipStatusUpdates: true,
			}
		})
		pipelineRunInformer.Informer().AddEventHandler(controller.HandleAll(impl.Enqueue))
		return impl
	}
}
