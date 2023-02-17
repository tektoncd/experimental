package taskmonitor

import (
	"context"

	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"

	taskmonitorinformer "github.com/tektoncd/experimental/metrics-operator/pkg/client/injection/informers/monitoring/v1alpha1/taskmonitor"
	taskmonitorreconciler "github.com/tektoncd/experimental/metrics-operator/pkg/client/injection/reconciler/monitoring/v1alpha1/taskmonitor"
)

func NewController() func(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
	return func(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
		taskMonitorInformer := taskmonitorinformer.Get(ctx)

		c := &Reconciler{
			// TaskMonitorInformer: taskMonitorInformer,
		}

		impl := taskmonitorreconciler.NewImpl(ctx, c, func(impl *controller.Impl) controller.Options {
			return controller.Options{
				AgentName: "TaskMonitor",
			}
		})
		taskMonitorInformer.Informer().AddEventHandler(controller.HandleAll(impl.Enqueue))
		return impl
	}
}
