package recorder

import (
	"context"
	"testing"

	"github.com/tektoncd/experimental/metrics-operator/pkg/apis/monitoring/v1alpha1"
	"github.com/tektoncd/experimental/metrics-operator/pkg/metrics"
	"go.opencensus.io/stats/view"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/ptr"
)

func TestX(t *testing.T) {

	// Start prometheus server
	exporter, err := metrics.NewPrometheusExporter(&metrics.MetricConfig{
		PrometheusHost: "0.0.0.0",
		PrometheusPort: 2112,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer exporter.Stop()
	go exporter.Start()

	external := view.NewMeter()
	external.Start()
	index := MetricIndex{
		external: external,
		store:    map[string]TaskRunCounter{},
	}

	taskMonitor := &v1alpha1.TaskMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name: "hello",
		},
		Spec: v1alpha1.TaskMonitorSpec{
			TaskName: "hello-world",
			Metrics: []v1alpha1.TaskMetric{
				{
					Name: "status",
					Type: "counter",
					By: []v1alpha1.TaskByStatement{
						{TaskRunValueRef: v1alpha1.TaskRunValueRef{Condition: ptr.String("Succeeded")}},
					},
				},
			},
		},
	}
	ctx := context.Background()
	err = index.RegisterTaskMetric(ctx, taskMonitor, &taskMonitor.Spec.Metrics[0])
	if err != nil {
		t.Fatal(err)
	}
}
