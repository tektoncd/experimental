package pipelinerun

import (
	"context"
	"github.com/tektoncd/experimental/cloudevents/pkg/apis/config"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline"
	pipelineruninformer "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1beta1/pipelinerun"
	pipelinerunreconciler "github.com/tektoncd/pipeline/pkg/client/injection/reconciler/pipeline/v1beta1/pipelinerun"
	cloudeventclient "github.com/tektoncd/pipeline/pkg/reconciler/events/cloudevent"
	"k8s.io/client-go/tools/cache"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/tracker"
	"time"
)

// NewController instantiates a new controller.Impl from knative.dev/pkg/controller
func NewController() func(context.Context, configmap.Watcher) *controller.Impl {
	return func(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
		logger := logging.FromContext(ctx)

		pipelineRunInformer := pipelineruninformer.Get(ctx)
		c := &Reconciler{
			cloudEventClient: cloudeventclient.Get(ctx),
		}
		impl := pipelinerunreconciler.NewImpl(ctx, c, func(impl *controller.Impl) controller.Options {
			configStore := config.NewStore(logger.Named("config-store"))
			configStore.WatchConfigs(cmw)
			return controller.Options{
				AgentName:   pipeline.PipelineRunControllerName,
				ConfigStore: configStore,
			}
		})

		logger.Info("Setting up event handlers")
		pipelineRunInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc:    impl.Enqueue,
			UpdateFunc: controller.PassNew(impl.Enqueue),
			DeleteFunc: impl.Enqueue,
		})

		c.tracker = tracker.New(impl.EnqueueKey, 30*time.Minute)
		return impl
	}
}
