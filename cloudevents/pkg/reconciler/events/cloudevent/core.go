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
	"fmt"

	"k8s.io/apimachinery/pkg/util/json"

	cdeevents "github.com/cdfoundation/sig-events/cde/sdk/go/pkg/cdf/events"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"knative.dev/pkg/apis"
)

// TODO(afrittoli) The valid statuses should be encoded in the SDK
// EvenStatus encodes valid statuses defined in https://github.com/cdfoundation/sig-events/blob/main/vocabulary-draft/core.md#continuous-delivery-core-events
type EventStatus string

const (
	StatusRunning  EventStatus = "Running"
	StatusFinished EventStatus = "Finished"
	StatusError    EventStatus = "Error"
)

type CDECloudEventData map[string]string

// getEventData returns a new instance of CDECloudEventData
func getEventData(runObject objectWithCondition) (CDECloudEventData, error) {
	cdeCloudEventData := map[string]string{}
	switch v := runObject.(type) {
	case *v1beta1.TaskRun:
		data, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		cdeCloudEventData["taskrun"] = string(data)
	case *v1beta1.PipelineRun:
		data, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		cdeCloudEventData["pipelinerun"] = string(data)
	}
	return cdeCloudEventData, nil
}

type EventType struct {
	Type   cdeevents.CDEventType
	Status EventStatus
}

// getEventType returns the event type and status
func getEventType(runObject objectWithCondition) (*EventType, error) {
	statusCondition := runObject.GetStatusCondition()
	if statusCondition == nil {
		return nil, fmt.Errorf("no condition for ConditionSucceeded in %T", runObject)
	}
	c := statusCondition.GetCondition(apis.ConditionSucceeded)
	if c == nil {
		return nil, fmt.Errorf("no condition for ConditionSucceeded in %T", runObject)
	}
	eventType := EventType{}
	switch {
	case c.IsUnknown():
		eventType.Status = StatusRunning
		switch runObject.(type) {
		case *v1beta1.TaskRun:
			switch c.Reason {
			case v1beta1.TaskRunReasonStarted.String():
				eventType.Type = cdeevents.TaskRunStartedEventV1
			case v1beta1.TaskRunReasonRunning.String():
				eventType.Type = cdeevents.TaskRunStartedEventV1
			// Unknown status, unknown reason -> no event type
			default:
				return nil, fmt.Errorf("unknown status with unknown reason %s", c.Reason)
			}
		case *v1beta1.PipelineRun:
			switch c.Reason {
			case v1beta1.PipelineRunReasonStarted.String():
				eventType.Type = cdeevents.PipelineRunQueuedEventV1
			case v1beta1.PipelineRunReasonRunning.String():
				eventType.Type = cdeevents.PipelineRunStartedEventV1
			// Unknown status, unknown reason -> no event type
			default:
				return nil, fmt.Errorf("unknown status with unknown reason %s", c.Reason)
			}
		}
	case c.IsTrue():
		eventType.Status = StatusFinished
		switch runObject.(type) {
		case *v1beta1.TaskRun:
			eventType.Type = cdeevents.TaskRunFinishedEventV1 //TaskRunFailedEventV1
		case *v1beta1.PipelineRun:
			eventType.Type = cdeevents.PipelineRunFinishedEventV1 //PipelineRunFailedEventV1
		}
	case c.IsFalse():
		eventType.Status = StatusError
		switch runObject.(type) {
		case *v1beta1.TaskRun:
			eventType.Type = cdeevents.TaskRunFinishedEventV1 //TaskRunFailedEventV1
		case *v1beta1.PipelineRun:
			eventType.Type = cdeevents.PipelineRunFinishedEventV1 //PipelineRunFailedEventV1
		}
	default:
		return nil, fmt.Errorf("unknown condition for in %T.Status %s", runObject, c.Status)
	}
	return &eventType, nil
}

// coreEventForObjectWithCondition creates a new event based for a objectWithCondition,
// or return an error if not possible.
func coreEventForObjectWithCondition(runObject objectWithCondition) (*cloudevents.Event, error) {
	var (
		event cloudevents.Event
		err   error
	)
	etype, err := getEventType(runObject)
	if err != nil {
		return nil, err
	}
	data, err := getEventData(runObject)
	if err != nil {
		return nil, err
	}
	meta := runObject.GetObjectMeta()
	switch runObject.(type) {
	case *v1beta1.TaskRun:
		event, err = cdeevents.CreateTaskRunEvent(etype.Type, string(meta.GetUID()), meta.GetName(), "", data)
		if err != nil {
			return nil, err
		}
	case *v1beta1.PipelineRun:
		event, err = cdeevents.CreatePipelineRunEvent(etype.Type, string(meta.GetUID()), meta.GetName(), string(etype.Status), "", "", data)
		if err != nil {
			return nil, err
		}
	}
	event.SetSubject(meta.GetName())
	event.SetSource(getSource(runObject))

	return &event, nil
}
