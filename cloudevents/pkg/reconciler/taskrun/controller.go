package taskrun

import (
	"context"

	"github.com/tektoncd/experimental/cloudevents/pkg/apis/config"
	cloudeventscache "github.com/tektoncd/experimental/cloudevents/pkg/reconciler/events/cache"
	cloudeventclient "github.com/tektoncd/experimental/cloudevents/pkg/reconciler/events/cloudevent"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline"
	taskruninformer "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1beta1/taskrun"
	taskrunreconciler "github.com/tektoncd/pipeline/pkg/client/injection/reconciler/pipeline/v1beta1/taskrun"
	"k8s.io/apimachinery/pkg/util/clock"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
)

// NewController instantiates a new controller.Impl from knative.dev/pkg/controller
func NewController(clock clock.PassiveClock) func(context.Context, configmap.Watcher) *controller.Impl {
	return func(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
		logger := logging.FromContext(ctx)

		taskRunInformer := taskruninformer.Get(ctx)
		c := &Reconciler{
			cloudEventClient: cloudeventclient.Get(ctx),
			cacheClient:      cloudeventscache.Get(ctx),
			Clock:            clock,
		}
		impl := taskrunreconciler.NewImpl(ctx, c, func(impl *controller.Impl) controller.Options {
			configStore := config.NewStore(logger.Named("config-store"))
			configStore.WatchConfigs(cmw)
			return controller.Options{
				AgentName:         pipeline.TaskRunControllerName,
				ConfigStore:       configStore,
				SkipStatusUpdates: true,
			}
		})

		logger.Info("Setting up event handlers")
		taskRunInformer.Informer().AddEventHandler(controller.HandleAll(impl.Enqueue))

		return impl
	}
}
