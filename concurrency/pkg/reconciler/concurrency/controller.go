package concurrency

import (
	"context"

	concurrencycontrolinformer "github.com/tektoncd/experimental/concurrency/pkg/client/injection/informers/concurrency/v1alpha1/concurrencycontrol"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline"
	pipelineclient "github.com/tektoncd/pipeline/pkg/client/injection/client"
	pipelineruninformer "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1beta1/pipelinerun"
	pipelinerunreconciler "github.com/tektoncd/pipeline/pkg/client/injection/reconciler/pipeline/v1beta1/pipelinerun"
	kubeclient "knative.dev/pkg/client/injection/kube/client"
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

		logger.Info("Setting up event handlers")

		pipelineClientSet := pipelineclient.Get(ctx)
		kubeClientSet := kubeclient.Get(ctx)
		pipelineRunInformer := pipelineruninformer.Get(ctx)
		concurrencyControlInformer := concurrencycontrolinformer.Get(ctx)

		r := &Reconciler{
			PipelineClientSet:        pipelineClientSet,
			KubeClientSet:            kubeClientSet,
			ConcurrencyControlLister: concurrencyControlInformer.Lister(),
			PipelineRunLister:        pipelineRunInformer.Lister(),
		}
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
