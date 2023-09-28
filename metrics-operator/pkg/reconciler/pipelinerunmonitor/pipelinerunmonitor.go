package pipelinerunmonitor

import (
	"context"
	"fmt"

	monitoringv1alpha1 "github.com/tektoncd/experimental/metrics-operator/pkg/apis/monitoring/v1alpha1"
	pipelinerunmonitorreconciler "github.com/tektoncd/experimental/metrics-operator/pkg/client/injection/reconciler/monitoring/v1alpha1/pipelinerunmonitor"
	"github.com/tektoncd/experimental/metrics-operator/pkg/metrics"
	"github.com/tektoncd/experimental/metrics-operator/pkg/metrics/recorder"
	pipelinev1beta1listers "github.com/tektoncd/pipeline/pkg/client/listers/pipeline/v1beta1"
	"k8s.io/apimachinery/pkg/util/sets"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/reconciler"
)

type Reconciler struct {
	manager       *metrics.MetricManager
	pipelineRunLister pipelinev1beta1listers.PipelineRunLister
}

var (
	resource = "pipelinerun"
	_ pipelinerunmonitorreconciler.Interface = (*Reconciler)(nil)
)

func (r *Reconciler) ReconcileKind(ctx context.Context, pipelineRunMonitor *monitoringv1alpha1.PipelineRunMonitor) reconciler.Event {
	logger := logging.FromContext(ctx).With("monitor", pipelineRunMonitor.Name)
	latestMetrics := sets.NewString()
	for _, metric := range pipelineRunMonitor.Spec.Metrics {
		var runMetric metrics.RunMetric
		// TODO: fail if type is invalid
		switch metric.Type {
		case "counter":
			runMetric = recorder.NewPipelineRunCounter(metric.DeepCopy(), pipelineRunMonitor)
		case "histogram":
			runMetric = recorder.NewPipelineRunHistogram(metric.DeepCopy(), pipelineRunMonitor)
		case "gauge":
			runMetric = recorder.NewPipelineRunGauge(metric.DeepCopy(), pipelineRunMonitor)
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

	registeredMetrics := sets.NewString(r.manager.Index.GetAllMetricNamesFromMonitor(resource, pipelineRunMonitor.Name)...)
	removed := registeredMetrics.Difference(latestMetrics)

	for _, removedMetricName := range removed.List() {
		err := r.manager.GetIndex().UnregisterRunMetricByName(removedMetricName)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *Reconciler) FinalizeKind(ctx context.Context, pipelineRunMonitor *monitoringv1alpha1.PipelineRunMonitor) reconciler.Event {
	err := r.manager.GetIndex().UnregisterAllMetricsMonitor(resource, pipelineRunMonitor.Name)
	if err != nil {
		return err
	}
	return nil
}
