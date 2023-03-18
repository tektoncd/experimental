package taskmonitor

import (
	"context"

	monitoringv1alpha1 "github.com/tektoncd/experimental/metrics-operator/pkg/apis/monitoring/v1alpha1"
	taskmonitorreconciler "github.com/tektoncd/experimental/metrics-operator/pkg/client/injection/reconciler/monitoring/v1alpha1/taskmonitor"
	"github.com/tektoncd/experimental/metrics-operator/pkg/metrics"
	"github.com/tektoncd/experimental/metrics-operator/pkg/metrics/recorder"
	"k8s.io/apimachinery/pkg/util/sets"
	"knative.dev/pkg/reconciler"
)

type Reconciler struct {
	manager *metrics.MetricManager
}

var (
	_ taskmonitorreconciler.Interface = (*Reconciler)(nil)
)

func (r *Reconciler) ReconcileKind(ctx context.Context, taskMonitor *monitoringv1alpha1.TaskMonitor) reconciler.Event {
	// logger := logging.FromContext(ctx)
	latestMetrics := sets.NewString()
	for i, metric := range taskMonitor.Spec.Metrics {
		var runMetric metrics.RunMetric
		if metric.Type == "counter" {
			runMetric = recorder.NewTaskCounter(&taskMonitor.Spec.Metrics[i], taskMonitor)
		}
		if runMetric != nil {
			latestMetrics = latestMetrics.Insert(runMetric.MetricName())
			err := r.manager.GetIndex().RegisterRunMetric(ctx, runMetric)
			if err != nil {
				return err
			}
		}
	}

	registeredMetrics := sets.NewString(r.manager.Index.GetAllMetricNamesFromMonitor(taskMonitor.Name)...)
	removed := registeredMetrics.Difference(latestMetrics)

	for _, removedMetricName := range removed.List() {
		err := r.manager.GetIndex().UnregisterRunMetricByName(removedMetricName)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *Reconciler) FinalizeKind(ctx context.Context, taskMonitor *monitoringv1alpha1.TaskMonitor) reconciler.Event {
	err := r.manager.GetIndex().UnregisterAllMetricsMonitor(taskMonitor.Name)
	if err != nil {
		return err
	}
	return nil
}
