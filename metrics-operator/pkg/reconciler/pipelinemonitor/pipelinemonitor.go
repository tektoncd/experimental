package pipelinemonitor

import (
	"context"
	"fmt"

	monitoringv1alpha1 "github.com/tektoncd/experimental/metrics-operator/pkg/apis/monitoring/v1alpha1"
	pipelinemonitorreconciler "github.com/tektoncd/experimental/metrics-operator/pkg/client/injection/reconciler/monitoring/v1alpha1/pipelinemonitor"
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
	_ pipelinemonitorreconciler.Interface = (*Reconciler)(nil)
)

func (r *Reconciler) ReconcileKind(ctx context.Context, pipelineMonitor *monitoringv1alpha1.PipelineMonitor) reconciler.Event {
	logger := logging.FromContext(ctx).With("monitor", pipelineMonitor.Name)
	latestMetrics := sets.NewString()
	for _, metric := range pipelineMonitor.Spec.Metrics {
		var runMetric metrics.RunMetric
		// TODO: fail if type is invalid
		switch metric.Type {
		case "counter":
			runMetric = recorder.NewPipelineCounter(metric.DeepCopy(), pipelineMonitor)
		case "histogram":
			runMetric = recorder.NewPipelineHistogram(metric.DeepCopy(), pipelineMonitor)
		case "gauge":
			runMetric = recorder.NewPipelineGauge(metric.DeepCopy(), pipelineMonitor)
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

	registeredMetrics := sets.NewString(r.manager.Index.GetAllMetricNamesFromMonitor("task", pipelineMonitor.Name)...)
	removed := registeredMetrics.Difference(latestMetrics)

	for _, removedMetricName := range removed.List() {
		err := r.manager.GetIndex().UnregisterRunMetricByName(removedMetricName)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *Reconciler) FinalizeKind(ctx context.Context, pipelineMonitor *monitoringv1alpha1.PipelineMonitor) reconciler.Event {
	err := r.manager.GetIndex().UnregisterAllMetricsMonitor("task", pipelineMonitor.Name)
	if err != nil {
		return err
	}
	return nil
}
