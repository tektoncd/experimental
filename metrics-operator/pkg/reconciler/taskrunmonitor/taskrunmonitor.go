package taskrunmonitor

import (
	"context"
	"fmt"

	monitoringv1alpha1 "github.com/tektoncd/experimental/metrics-operator/pkg/apis/monitoring/v1alpha1"
	taskrunmonitorreconciler "github.com/tektoncd/experimental/metrics-operator/pkg/client/injection/reconciler/monitoring/v1alpha1/taskrunmonitor"
	"github.com/tektoncd/experimental/metrics-operator/pkg/metrics"
	"github.com/tektoncd/experimental/metrics-operator/pkg/metrics/recorder"
	pipelinev1beta1listers "github.com/tektoncd/pipeline/pkg/client/listers/pipeline/v1beta1"
	"k8s.io/apimachinery/pkg/util/sets"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/reconciler"
)

type Reconciler struct {
	manager       *metrics.MetricManager
	taskRunLister pipelinev1beta1listers.TaskRunLister
}

var (
	_ taskrunmonitorreconciler.Interface = (*Reconciler)(nil)
)

func (r *Reconciler) ReconcileKind(ctx context.Context, taskRunMonitor *monitoringv1alpha1.TaskRunMonitor) reconciler.Event {
	logger := logging.FromContext(ctx).With("monitor", taskRunMonitor.Name)
	latestMetrics := sets.NewString()
	for _, metric := range taskRunMonitor.Spec.Metrics {
		var runMetric metrics.RunMetric
		// TODO: fail if type is invalid
		switch metric.Type {
		case "counter":
			runMetric = recorder.NewTaskRunCounter(metric.DeepCopy(), taskRunMonitor)
		case "histogram":
			runMetric = recorder.NewTaskRunHistogram(metric.DeepCopy(), taskRunMonitor)
		case "gauge":
			runMetric = recorder.NewTaskRunGauge(metric.DeepCopy(), taskRunMonitor, r.taskRunLister)
		default:
			logger.Errorw("invalid metric type", "metric", metric.Name, "type", metric.Type)
			return fmt.Errorf("invalid metric type: %q", metric.Type)
		}
		if runMetric != nil {
			latestMetrics = latestMetrics.Insert(runMetric.MetricName())
			err := r.manager.GetIndex().RegisterRunMetric(ctx, runMetric)
			if err != nil {
				return err
			}
		}
	}

	registeredMetrics := sets.NewString(r.manager.Index.GetAllMetricNamesFromMonitor(taskRunMonitor.Name)...)
	removed := registeredMetrics.Difference(latestMetrics)

	for _, removedMetricName := range removed.List() {
		err := r.manager.GetIndex().UnregisterRunMetricByName(removedMetricName)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *Reconciler) FinalizeKind(ctx context.Context, taskRunMonitor *monitoringv1alpha1.TaskMonitor) reconciler.Event {
	err := r.manager.GetIndex().UnregisterAllMetricsMonitor(taskRunMonitor.Name)
	if err != nil {
		return err
	}
	return nil
}
