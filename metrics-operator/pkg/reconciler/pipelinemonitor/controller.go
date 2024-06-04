package pipelinemonitor

import (
	"context"

	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/injection"

	pipelinemonitorinformer "github.com/tektoncd/experimental/metrics-operator/pkg/client/injection/informers/monitoring/v1alpha1/pipelinemonitor"
	pipelinemonitorreconciler "github.com/tektoncd/experimental/metrics-operator/pkg/client/injection/reconciler/monitoring/v1alpha1/pipelinemonitor"
	"github.com/tektoncd/experimental/metrics-operator/pkg/metrics"
	pipelineruninformer "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1beta1/pipelinerun"
)

func NewController(manager *metrics.MetricManager) injection.ControllerConstructor {
	return func(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
		pipelineMonitorInformer := pipelinemonitorinformer.Get(ctx)
		pipelineRunInformer := pipelineruninformer.Get(ctx)

		c := &Reconciler{
			manager:           manager,
			pipelineRunLister: pipelineRunInformer.Lister(),
		}

		impl := pipelinemonitorreconciler.NewImpl(ctx, c, func(impl *controller.Impl) controller.Options {
			return controller.Options{}
		})
		pipelineMonitorInformer.Informer().AddEventHandler(controller.HandleAll(impl.Enqueue))
		return impl
	}
}
