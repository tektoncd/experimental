package events

import (
	"context"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/tektoncd/experimental/cloudevents/pkg/apis/config"
	"github.com/tektoncd/pipeline/pkg/reconciler/events/cloudevent"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/runtime"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/logging"
)

// Emit emits events for object
// Two types of events are supported, k8s and cloud events.
//
// k8s events are always sent if afterCondition is different from beforeCondition
// Cloud events are always sent if enabled, i.e. if a sink is available
func Emit(ctx context.Context, beforeCondition *apis.Condition, afterCondition *apis.Condition, object runtime.Object) {
	logger := logging.FromContext(ctx)
	configs := config.FromContextOrDefaults(ctx)
	sendCloudEvents := configs.Defaults.DefaultCloudEventsSink != ""
	if sendCloudEvents {
		ctx = cloudevents.ContextWithTarget(ctx, configs.Defaults.DefaultCloudEventsSink)
	}

	if sendCloudEvents {
		// Only send events if the new condition represents a change
		if !equality.Semantic.DeepEqual(beforeCondition, afterCondition) {
			err := cloudevent.SendCloudEventWithRetries(ctx, object)
			if err != nil {
				logger.Warnf("Failed to emit cloud events %v", err.Error())
			}
		}
	}
}
