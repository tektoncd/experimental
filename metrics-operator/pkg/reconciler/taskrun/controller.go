package taskrun

import (
	"context"

	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/injection"

	"github.com/tektoncd/experimental/metrics-operator/pkg/metrics"
	taskruninformer "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1beta1/taskrun"
	taskrunreconciler "github.com/tektoncd/pipeline/pkg/client/injection/reconciler/pipeline/v1beta1/taskrun"
)

func NewController(manager *metrics.MetricManager) injection.ControllerConstructor {
	return func(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
		taskRunInformer := taskruninformer.Get(ctx)

		c := &Reconciler{
			manager: manager,
		}

		impl := taskrunreconciler.NewImpl(ctx, c, func(impl *controller.Impl) controller.Options {
			return controller.Options{
				FinalizerName:     "taskrun.metrics.tekton.dev",
				SkipStatusUpdates: true,
			}
		})
		taskRunInformer.Informer().AddEventHandler(controller.HandleAll(impl.Enqueue))
		return impl
	}
}
