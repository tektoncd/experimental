package workflows

import (
	"context"

	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"

	workflowsinformer "github.com/tektoncd/experimental/workflows/pkg/client/injection/informers/workflows/v1alpha1/workflow"
	workflowsreconciler "github.com/tektoncd/experimental/workflows/pkg/client/injection/reconciler/workflows/v1alpha1/workflow"
)

// NewController creates a Reconciler and returns the result of NewImpl.
func NewController(
	ctx context.Context,
	cmw configmap.Watcher,
) *controller.Impl {
	workflowsInformer := workflowsinformer.Get(ctx)
	r := &Reconciler{}
	impl := workflowsreconciler.NewImpl(ctx, r)
	workflowsInformer.Informer().AddEventHandler(controller.HandleAll(impl.Enqueue))
	return impl
}
