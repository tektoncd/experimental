package cloudevent

import (
	cloudevents "github.com/cloudevents/sdk-go/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"knative.dev/pkg/apis"
)

// objectWithCondition is implemented by TaskRun and PipelineRun
type objectWithCondition interface {

	// Object requires GetObjectKind() and DeepCopyObject()
	runtime.Object

	// ObjectMetaAccessor requires a GetObjectMeta that returns the ObjectMeta
	metav1.ObjectMetaAccessor

	// GetStatusCondition returns a ConditionAccessor for the status of the RunsToCompletion
	GetStatusCondition() apis.ConditionAccessor
}

// cdEventAnnotationKey is the name of the annotations used to store the kind of CD Event
const CDEventAnnotationTypeKey string = "cd.events/type"

// cdEventAnnotationType is an ENUM with all possible values for cdEventAnnotationKey
type cdEventAnnotationType string

// cdEventCreate is a function that creates a cd event from an objectWithCondition
type cdEventCreator func(runObject objectWithCondition) (*cloudevents.Event, error)
