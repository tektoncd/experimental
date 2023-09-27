package metrics

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/tektoncd/experimental/metrics-operator/pkg/apis/monitoring/v1alpha1"
	"github.com/tektoncd/experimental/metrics-operator/pkg/metrics/recorder"
	"github.com/tektoncd/experimental/metrics-operator/pkg/server"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"go.opencensus.io/stats/view"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/ptr"
)

func TestMetricIndex(t *testing.T) {

	// Start prometheus server
	exporter, err := server.NewPrometheusExporter(&server.MetricConfig{
		PrometheusHost: "0.0.0.0",
		PrometheusPort: 2112,
		Namespace:      "tekton_metrics",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer exporter.Stop()
	go exporter.Start()

	external := view.NewMeter()
	external.Start()

	// Setup data
	index := MetricIndex{
		external: external,
		store:    map[string]RunMetric{},
	}

	taskMonitor := &v1alpha1.TaskMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name: "hello",
		},
		Spec: v1alpha1.TaskMonitorSpec{
			TaskName: "hello-world",
			Metrics: []v1alpha1.Metric{
				{
					Name: "status",
					Type: "counter",
					By: []v1alpha1.ByStatement{
						{MetricDimensionRef: v1alpha1.MetricDimensionRef{Condition: ptr.String("Succeeded")}},
					},
				},
			},
		},
	}
	ctx := context.Background()
	taskMetric := &taskMonitor.Spec.Metrics[0]
	counter := recorder.NewTaskCounter(taskMetric, taskMonitor)

	t.Run("able to register metric", func(t *testing.T) {
		err = index.RegisterRunMetric(ctx, counter)
		if err != nil {
			t.Fatal(err)
		}
		index.Record(ctx, &v1beta1.TaskRun{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "hello-world-xpto0",
				Namespace: "dev",
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Type:   apis.ConditionSucceeded,
							Status: v1.ConditionTrue,
						},
					},
				},
			},
		}, "counter")

		resp, err := http.Get("http://0.0.0.0:2112/metrics")
		if err != nil {
			t.Fatal(err)
		}
		var parser expfmt.TextParser
		mf, err := parser.TextToMetricFamilies(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		expected := map[string]*dto.MetricFamily{
			"tekton_metrics_task_hello_status_total": {
				Name: ptr.String("tekton_metrics_task_hello_status_total"),
				Help: ptr.String("count samples for TaskMonitor hello/status"), // TODO: fixme
				Type: dto.MetricType_COUNTER.Enum(),
				Metric: []*dto.Metric{
					{
						Counter: &dto.Counter{Value: ptr.Float64(1)},
						Label: []*dto.LabelPair{
							{
								Name:  ptr.String("status"),
								Value: ptr.String("success"),
							},
						},
					},
				},
			},
		}
		if diff := cmp.Diff(expected, mf); diff != "" {
			t.Errorf("metrics (-want, +got):\n%s\n", diff)
		}
	})

	t.Run("able to unregister metric", func(t *testing.T) {

		err = index.UnregisterRunMetric(counter)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := http.Get("http://0.0.0.0:2112/metrics")
		if err != nil {
			t.Fatal(err)
		}
		var parser expfmt.TextParser
		mf, err := parser.TextToMetricFamilies(resp.Body)
		if err != nil {
			t.Fatal(err)
		}

		expected := map[string]*dto.MetricFamily{}
		if diff := cmp.Diff(expected, mf); diff != "" {
			t.Errorf("metrics (-want, +got):\n%s\n", diff)
		}
	})

}
