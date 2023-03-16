package taskmonitor

import (
	"context"
	"fmt"
	"strings"

	monitoringv1alpha1 "github.com/tektoncd/experimental/metrics-operator/pkg/apis/monitoring/v1alpha1"
	taskmonitorreconciler "github.com/tektoncd/experimental/metrics-operator/pkg/client/injection/reconciler/monitoring/v1alpha1/taskmonitor"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/reconciler"
)

type Reconciler struct {
	// TaskMonitorInformer taskmonitorinformer.Informer
	external view.Meter
}

var (
	_ taskmonitorreconciler.Interface = (*Reconciler)(nil)
)

func (r *Reconciler) ReconcileKind(ctx context.Context, taskMonitor *monitoringv1alpha1.TaskMonitor) reconciler.Event {
	logger := logging.FromContext(ctx)
	metricName := countMetricName("taskrun", taskMonitor.GetName())
	existingView := r.external.Find(metricName)
	if existingView != nil {
		logger.Info("view already registered")
		return nil
	}

	countMeasure := stats.Float64(metricName, fmt.Sprintf("count samples for TaskMonitor %s", taskMonitor.GetName()), stats.UnitDimensionless)
	countView := &view.View{
		Description: countMeasure.Description(),
		Measure:     countMeasure,
		Aggregation: view.Count(),
	}


	logger.Info("registering view")
	r.external.Register(countView)
	logger.Info("registered view")

	fmt.Printf("Recording...\n")
	r.external.Record(&tag.Map{}, []stats.Measurement{countMeasure.M(1)}, map[string]any{})
	fmt.Printf("Recorded...\n")
	return nil
}

func countMetricName(resource string, name string) string {
	return fmt.Sprintf("tekton_metrics_%s_%s_count", resource, strings.ReplaceAll(name, "-", "_"))
}
