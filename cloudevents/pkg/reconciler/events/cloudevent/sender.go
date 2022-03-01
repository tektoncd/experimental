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

package cloudevent

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/tektoncd/experimental/cloudevents/pkg/reconciler/events/cache"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
)

var cdeventsEventCreators = []eventCreator{
	coreEventForObjectWithCondition,
	artifactPackagedEventForObjectWithCondition,
	artifactPublishedEventForObjectWithCondition,
	serviceRemovedEventForObjectWithCondition,
	serviceUpgradedEventForObjectWithCondition,
	serviceDeployedEventForObjectWithCondition,
	serviceRolledbackEventForObjectWithCondition,
}

var legacyEventCreators = []eventCreator{
	eventForObjectWithCondition,
}

func getSource(runObject objectWithCondition) string {
	meta := runObject.GetObjectMeta()
	var kindString string
	switch runObject.(type) {
	case *v1beta1.TaskRun:
		kindString = "taskrun"
	case *v1beta1.PipelineRun:
		kindString = "pipelinerun"
	}
	return fmt.Sprintf("/tekton/namespaces/%s/%s",
		meta.GetNamespace(),
		kindString)
}

// SendCloudEventWithRetries sends a cloud event for the specified resource.
// It does not block and it perform retries with backoff using the cloudevents
// sdk-go capabilities.
// It accepts a runtime.Object to avoid making objectWithCondition public since
// it's only used within the events/cloudevents packages.
func SendCloudEventWithRetries(ctx context.Context, object runtime.Object, format string) error {
	logging.FromContext(ctx).Infof("send event for object of kind: %s", object.GetObjectKind())
	var (
		o             objectWithCondition
		ok            bool
		eventCreators []eventCreator
	)
	if o, ok = object.(objectWithCondition); !ok {
		return errors.New("input object does not satisfy objectWithCondition")
	}
	ceClient := Get(ctx)
	if ceClient == nil {
		return errors.New("no cloud events client found in the context")
	}
	if format == "cdevents" {
		eventCreators = cdeventsEventCreators
	} else {
		eventCreators = legacyEventCreators
	}

	for _, eventCreator := range eventCreators {
		event, err := eventCreator(o)
		if err != nil {
			logging.FromContext(ctx).Warnf("no event to send %s", err)
			continue
		}
		err = sendCloudEventWithRetries(ctx, object, event)
		if err != nil {
			logging.FromContext(ctx).Warnf("got error %s while sending event %T", err, event)
		}
	}
	return nil
}

func sendCloudEventWithRetries(ctx context.Context, object runtime.Object, event *cloudevents.Event) error {
	logger := logging.FromContext(ctx)
	ceClient := Get(ctx)
	cacheClient := cache.Get(ctx)
	wasIn := make(chan error)
	go func() {
		wasIn <- nil
		logger.Debugf("Sending cloudevent of type %q", event.Type())
		// check cache if cloudevent is already sent
		cloudEventSent, err := cache.IsCloudEventSent(cacheClient, event)
		if err != nil {
			logger.Errorf("error while checking cache: %s", err)
		}
		if cloudEventSent {
			logger.Infof("cloudevent %s already sent", cache.EventKey(event))
			return
		}
		if result := ceClient.Send(cloudevents.ContextWithRetriesExponentialBackoff(ctx, 10*time.Millisecond, 10), *event); !cloudevents.IsACK(result) {
			logger.Warnf("Failed to send cloudevent: %s", result.Error())
			recorder := controller.GetEventRecorder(ctx)
			if recorder == nil {
				logger.Warnf("No recorder in context, cannot emit error event")
			} else {
				recorder.Event(object, corev1.EventTypeWarning, "Cloud Event Failure", result.Error())
			}
		}
		if err := cache.AddEventSentToCache(cacheClient, event); err != nil {
			logger.Errorf("error while adding sent event to cache: %s", err)
		}
	}()
	return <-wasIn
}
