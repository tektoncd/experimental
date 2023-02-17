package main

import (
	"github.com/tektoncd/experimental/metrics-operator/pkg/reconciler/taskmonitor"
	"knative.dev/pkg/injection/sharedmain"
	"knative.dev/pkg/signals"
)

// "github.com/tektoncd/experimental/concurrency/pkg/apis/concurrency/v1alpha1"
// "github.com/tektoncd/experimental/concurrency/pkg/reconciler/concurrency"

// "go.opencensus.io/stats"
// "go.opencensus.io/tag"
// "knative.dev/pkg/signals"

func main() {
	// taskRunCount := stats.Int64("task_count", "TaskRun count", "By")
	// statusTag, _ := tag.NewKey("status")

	ctx := signals.NewContext()
	sharedmain.MainWithContext(ctx, "metrics-operator-controller", taskmonitor.NewController())
	// ctx := filteredinformerfactory.WithSelectors(signals.NewContext(), v1alpha1.ManagedByLabelKey)
	// sharedmain.MainWithContext(ctx, concurrency.ControllerName, concurrency.NewController())
}
