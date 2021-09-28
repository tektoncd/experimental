package v1alpha1

import (
	"github.com/tektoncd/experimental/remote-resolution/pkg/resolution"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"knative.dev/pkg/apis"
)

// ResourceRequests only have apis.ConditionSucceeded for now.
var resourceRequestCondSet = apis.NewBatchConditionSet()

// GetGroupVersionKind implements kmeta.OwnerRefable.
func (*ResourceRequest) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("ResourceRequest")
}

// GetConditionSet implements KRShaped.
func (*ResourceRequest) GetConditionSet() apis.ConditionSet {
	return resourceRequestCondSet
}

// InitializeConditions set ths initial values of the conditions.
func (rr *ResourceRequestStatus) InitializeConditions() {
	resourceRequestCondSet.Manage(rr).InitializeConditions()
}

// HasStarted returns whether a ResourceRequests Status is considered to
// be in-progress.
//
// TODO: This might be better served by having a "start time" recorded
// in the status at the point that a resource request's processing has
// begun.
func (rr *ResourceRequest) HasStarted() bool {
	return rr.Status.GetCondition(apis.ConditionSucceeded).IsUnknown()
}

// IsDone returns whether a ResourceRequests Status is considered to be
// in a completed state, independent of success/failure.
func (rr *ResourceRequest) IsDone() bool {
	finalStateIsUnknown := rr.Status.GetCondition(apis.ConditionSucceeded).IsUnknown()
	return !finalStateIsUnknown
}

// MarkFailed sets the Succeeded condition to False with an accompanying
// error message.
func (s *ResourceRequestStatus) MarkFailed(reason, message string) {
	resourceRequestCondSet.Manage(s).MarkFalse(apis.ConditionSucceeded, reason, message)
}

// MarkSucceeded sets the Succeeded condition to True.
func (s *ResourceRequestStatus) MarkSucceeded() {
	resourceRequestCondSet.Manage(s).MarkTrue(apis.ConditionSucceeded)
}

// MarkInProgress updates the Succeeded condition to Unknown with an
// accompanying message.
func (s *ResourceRequestStatus) MarkInProgress(message string) {
	resourceRequestCondSet.Manage(s).MarkUnknown(apis.ConditionSucceeded, resolution.ReasonResolutionInProgress, message)
}
