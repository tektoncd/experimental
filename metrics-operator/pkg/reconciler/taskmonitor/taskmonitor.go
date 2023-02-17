package taskmonitor

import (
	"context"

	monitoringv1alpha1 "github.com/tektoncd/experimental/metrics-operator/pkg/apis/monitoring/v1alpha1"
	taskmonitorreconciler "github.com/tektoncd/experimental/metrics-operator/pkg/client/injection/reconciler/monitoring/v1alpha1/taskmonitor"
	"knative.dev/pkg/reconciler"
)

type Reconciler struct {
	// TaskMonitorInformer taskmonitorinformer.Informer
}

var (
	_ taskmonitorreconciler.Interface = (*Reconciler)(nil)
)

func (r *Reconciler) ReconcileKind(ctx context.Context, taskMonitor *monitoringv1alpha1.TaskMonitor) reconciler.Event {
	return nil
}
