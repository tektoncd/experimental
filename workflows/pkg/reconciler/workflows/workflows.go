package workflows

import (
	"context"

	"github.com/tektoncd/experimental/workflows/pkg/apis/workflows/v1alpha1"
	workflowsreconciler "github.com/tektoncd/experimental/workflows/pkg/client/injection/reconciler/workflows/v1alpha1/workflow"
	"knative.dev/pkg/reconciler"
)

type Reconciler struct {
}

var _ workflowsreconciler.Interface = (*Reconciler)(nil)

func (r *Reconciler) ReconcileKind(ctx context.Context, w *v1alpha1.Workflow) reconciler.Event {
	return nil
}
