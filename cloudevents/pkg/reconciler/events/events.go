/*
Copyright 2021 The Tekton Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package events

import (
	"context"

	"github.com/tektoncd/experimental/cloudevents/pkg/apis/config"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/tektoncd/experimental/cloudevents/pkg/reconciler/events/cloudevent"
	"k8s.io/apimachinery/pkg/runtime"
	"knative.dev/pkg/logging"
)

// Emit emits cloud events for object
//
// Cloud events are sent if enabled if a sink is available
func Emit(ctx context.Context, object runtime.Object) {
	logger := logging.FromContext(ctx)
	configs := config.FromContextOrDefaults(ctx)
	sendCloudEvents := (configs.Defaults.DefaultCloudEventsSink != "")
	if !sendCloudEvents {
		logger.Warnf("No DefaultCloudEventsSink set: %s", configs.Defaults.DefaultCloudEventsSink)
		return
	}
	cloudEventsFormat := configs.Defaults.DefaultCloudEventsFormat
	ctx = cloudevents.ContextWithTarget(ctx, configs.Defaults.DefaultCloudEventsSink)

	logger.Debugf("Sending events for %s to %s", object, configs.Defaults.DefaultCloudEventsSink)
	err := cloudevent.SendCloudEventWithRetries(ctx, object, cloudEventsFormat)
	if err != nil {
		logger.Warnf("Failed to emit cloud events %v", err.Error())
	}
}
