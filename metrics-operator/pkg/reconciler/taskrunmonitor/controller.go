package taskrunmonitor

import (
	"context"

	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/injection"

	taskrunmonitorinformer "github.com/tektoncd/experimental/metrics-operator/pkg/client/injection/informers/monitoring/v1alpha1/taskrunmonitor"
	taskrunmonitorreconciler "github.com/tektoncd/experimental/metrics-operator/pkg/client/injection/reconciler/monitoring/v1alpha1/taskrunmonitor"
	"github.com/tektoncd/experimental/metrics-operator/pkg/metrics"
	taskruninformer "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1beta1/taskrun"
)

func NewController(manager *metrics.MetricManager) injection.ControllerConstructor {
	return func(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
		taskRunMonitorInformer := taskrunmonitorinformer.Get(ctx)
		taskRunInformer := taskruninformer.Get(ctx)

		c := &Reconciler{
			manager:       manager,
			taskRunLister: taskRunInformer.Lister(),
		}

		impl := taskrunmonitorreconciler.NewImpl(ctx, c, func(impl *controller.Impl) controller.Options {
			return controller.Options{
				AgentName: "TaskRunMonitor",
			}
		})
		taskRunMonitorInformer.Informer().AddEventHandler(controller.HandleAll(impl.Enqueue))
		return impl
	}
}
