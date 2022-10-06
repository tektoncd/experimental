package workflows

import (
	"context"

	"github.com/tektoncd/experimental/workflows/pkg/apis/workflows/v1alpha1"
	workflowsinformer "github.com/tektoncd/experimental/workflows/pkg/client/injection/informers/workflows/v1alpha1/workflow"
	workflowsreconciler "github.com/tektoncd/experimental/workflows/pkg/client/injection/reconciler/workflows/v1alpha1/workflow"
	triggersclient "github.com/tektoncd/triggers/pkg/client/injection/client"
	triggersinformer "github.com/tektoncd/triggers/pkg/client/injection/informers/triggers/v1beta1/trigger"
	"k8s.io/client-go/tools/cache"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
)

// NewController creates a Reconciler and returns the result of NewImpl.
func NewController(
	ctx context.Context,
	cmw configmap.Watcher,
) *controller.Impl {
	workflowsInformer := workflowsinformer.Get(ctx)
	triggersclientset := triggersclient.Get(ctx)
	triggersInformer := triggersinformer.Get(ctx)
	r := &Reconciler{
		TriggerClientSet: triggersclientset,
		TriggerLister:    triggersinformer.Get(ctx).Lister(),
	}
	impl := workflowsreconciler.NewImpl(ctx, r)
	workflowsInformer.Informer().AddEventHandler(controller.HandleAll(impl.Enqueue))
	triggersInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: controller.FilterController(&v1alpha1.Workflow{}),
		Handler:    controller.HandleAll(impl.EnqueueControllerOf),
	})
	return impl
}
