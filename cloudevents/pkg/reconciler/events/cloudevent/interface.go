package cloudevent

import (
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
