package cloudevent

import (
	"context"
	"errors"
	"fmt"
	cdeevents "github.com/cdfoundation/sig-events/cde/sdk/go/pkg/cdf/events"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/reconciler/events/cloudevent"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
	"time"
)

// CEClient matches the `Client` interface from github.com/cloudevents/sdk-go/v2/cloudevents
type CEClient cloudevents.Client

// TektonCloudEventData type is used to marshal and unmarshal the payload of
// a Tekton cloud event. It can include a TaskRun or a PipelineRun
type TektonCloudEventData struct {
	TaskRun     *v1beta1.TaskRun     `json:"taskRun,omitempty"`
	PipelineRun *v1beta1.PipelineRun `json:"pipelineRun,omitempty"`
}

// newTektonCloudEventData returns a new instance of TektonCloudEventData
func newTektonCloudEventData(runObject objectWithCondition) TektonCloudEventData {
	tektonCloudEventData := TektonCloudEventData{}
	switch v := runObject.(type) {
	case *v1beta1.TaskRun:
		tektonCloudEventData.TaskRun = v
	case *v1beta1.PipelineRun:
		tektonCloudEventData.PipelineRun = v
	}
	return tektonCloudEventData
}

func getEventType(runObject objectWithCondition) (*cdeevents.CDEventType, error) {
	c := runObject.GetStatusCondition().GetCondition(apis.ConditionSucceeded)
	if c == nil {
		return nil, fmt.Errorf("no condition for ConditionSucceeded in %T", runObject)
	}
	var eventType cdeevents.CDEventType
	switch {
	// TODO : Update TaskRun/PipelineRun events from finished to success/fa
	case c.IsUnknown():
		switch runObject.(type) {
		case *v1beta1.TaskRun:
			switch c.Reason {
			case v1beta1.TaskRunReasonStarted.String():
				eventType = cdeevents.TaskRunStartedEventV1
				//case v1beta1.TaskRunReasonRunning.String():
				//	eventType = TaskRunRunningEventV1
				//default:
				//	eventType = TaskRunUnknownEventV1
			}
		case *v1beta1.PipelineRun:
			switch c.Reason {
			case v1beta1.PipelineRunReasonStarted.String():
				eventType = cdeevents.PipelineRunStartedEventV1
				//case v1beta1.PipelineRunReasonRunning.String():
				//	eventType = PipelineRunRunningEventV1
				//default:
				//	eventType = PipelineRunUnknownEventV1
			}
		}
	case c.IsFalse():
		switch runObject.(type) {
		case *v1beta1.TaskRun:
			eventType = cdeevents.TaskRunFinishedEventV1 //TaskRunFailedEventV1
		case *v1beta1.PipelineRun:
			eventType = cdeevents.PipelineRunFinishedEventV1 //PipelineRunFailedEventV1
		}
	case c.IsTrue():
		switch runObject.(type) {
		case *v1beta1.TaskRun:
			eventType = cdeevents.TaskRunFinishedEventV1 //TaskRunSuccessfulEventV1
		case *v1beta1.PipelineRun:
			eventType = cdeevents.PipelineRunFinishedEventV1 //PipelineRunSuccessfulEventV1
		}
	default:
		return nil, fmt.Errorf("unknown condition for in %T.Status %s", runObject, c.Status)
	}
	return &eventType, nil
}

// eventForObjectWithCondition creates a new event based for a objectWithCondition,
// or return an error if not possible.
func eventForObjectWithCondition(runObject objectWithCondition) (*cloudevents.Event, error) {
	event := cloudevents.NewEvent()
	event.SetID(uuid.New().String())
	event.SetSubject(runObject.GetObjectMeta().GetName())
	// TODO: SelfLink is deprecated https://github.com/tektoncd/pipeline/issues/2676
	source := runObject.GetObjectMeta().GetSelfLink()
	if source == "" {
		gvk := runObject.GetObjectKind().GroupVersionKind()
		source = fmt.Sprintf("/apis/%s/%s/namespaces/%s/%s/%s",
			gvk.Group,
			gvk.Version,
			runObject.GetObjectMeta().GetNamespace(),
			gvk.Kind,
			runObject.GetObjectMeta().GetName())
	}
	event.SetSource(source)
	eventType, err := getEventType(runObject)
	if err != nil {
		return nil, err
	}
	if eventType == nil {
		return nil, errors.New("No matching event type found")
	}
	event.SetType(eventType.String())

	if err := event.SetData(cloudevents.ApplicationJSON, newTektonCloudEventData(runObject)); err != nil {
		return nil, err
	}
	return &event, nil
}

// SendCloudEventWithRetries sends a cloud event for the specified resource.
// It does not block and it perform retries with backoff using the cloudevents
// sdk-go capabilities.
// It accepts a runtime.Object to avoid making objectWithCondition public since
// it's only used within the events/cloudevents packages.
func SendCloudEventWithRetries(ctx context.Context, object runtime.Object) error {
	var (
		o  objectWithCondition
		ok bool
	)
	if o, ok = object.(objectWithCondition); !ok {
		return errors.New("Input object does not satisfy objectWithCondition")
	}
	logger := logging.FromContext(ctx)
	ceClient := cloudevent.Get(ctx)
	if ceClient == nil {
		return errors.New("No cloud events client found in the context")
	}
	event, err := eventForObjectWithCondition(o)
	if err != nil {
		return err
	}

	wasIn := make(chan error)
	go func() {
		wasIn <- nil
		logger.Debugf("Sending cloudevent of type %q", event.Type())
		if result := ceClient.Send(cloudevents.ContextWithRetriesExponentialBackoff(ctx, 10*time.Millisecond, 10), *event); !cloudevents.IsACK(result) {
			logger.Warnf("Failed to send cloudevent: %s", result.Error())
			recorder := controller.GetEventRecorder(ctx)
			if recorder == nil {
				logger.Warnf("No recorder in context, cannot emit error event")
			}
			recorder.Event(object, corev1.EventTypeWarning, "Cloud Event Failure", result.Error())
		}
	}()

	return <-wasIn
}
