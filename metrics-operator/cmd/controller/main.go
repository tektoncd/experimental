package main

import (
	"fmt"

	"github.com/tektoncd/experimental/metrics-operator/pkg/metrics"
	"github.com/tektoncd/experimental/metrics-operator/pkg/reconciler/taskmonitor"
	"go.opencensus.io/stats/view"
	"knative.dev/pkg/injection/sharedmain"
	"knative.dev/pkg/signals"
)

func main() {

	fmt.Printf("Starting metric-operator...\n")
	exporter, err := metrics.NewPrometheusExporter(&metrics.MetricConfig{
		PrometheusHost: "0.0.0.0",
		PrometheusPort: 2112,
	})
	if err != nil {
		panic("failed to start external prometheus exporter")
	}
	fmt.Printf("Starting external prometheus exporter...\n")
	go func() {
		exporter.Start()
	}()

	fmt.Printf("Starting meter...\n")
	external := view.NewMeter()
	external.Start()
	fmt.Printf("Starting registering external exporter...\n")
	external.RegisterExporter(exporter.GetExporter())

	// ctx := context.Background()

	// fmt.Printf("Recording...\n")
	// external.Record(&tag.Map{}, []stats.Measurement{countMeasure.M(1)}, map[string]any{})
	// fmt.Printf("Recorded...\n")

	ctx := signals.NewContext()
	sharedmain.MainWithContext(ctx, "metrics-operator-controller", taskmonitor.NewController(external))
}
