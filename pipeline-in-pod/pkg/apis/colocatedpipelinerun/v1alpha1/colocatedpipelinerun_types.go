package v1alpha1

import (
	"context"
	"time"

	"github.com/tektoncd/pipeline/pkg/apis/config"
	v1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

const ControllerName = "ColocatedPipelineRun"

var cprCondSet = apis.NewBatchConditionSet()

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ColocatedPipelineRun TODO
// +k8s:openapi-gen=true
type ColocatedPipelineRun struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata"`

	// Spec holds the desired state of the ColocatedPipelineRun from the client
	// +optional
	Spec ColocatedPipelineRunSpec `json:"spec"`

	Status ColocatedPipelineRunStatus `json:"status,omitempty"`
}

// ColocatedPipelineRunSpec defines the desired state of the ColocatedPipelineRun
type ColocatedPipelineRunSpec struct {
	// +optional
	PipelineRef *v1beta1.PipelineRef `json:"pipelineRef,omitempty"`

	// +optional
	PipelineSpec *v1beta1.PipelineSpec `json:"pipelineSpec,omitempty"`

	// +optional
	ServiceAccountName string `json:"serviceAccountName,omitempty"`

	// Time after which the ColocatedPipelineRun times out.
	// +optional
	Timeouts *v1beta1.TimeoutFields `json:"timeouts,omitempty"`

	// +optional
	Params []v1beta1.Param `json:"params,omitempty"`

	// +optional
	// +listType=atomic
	Workspaces []v1beta1.WorkspaceBinding `json:"workspaces,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ColocatedPipelineRunList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ColocatedPipelineRun `json:"items"`
}

// ColocatedPipelineRunReason represents a reason for the Run "Succeeded" condition
type ColocatedPipelineRunReason string

func (t ColocatedPipelineRunReason) String() string {
	return string(t)
}

// ColocatedPipelineRunStatus defines the observed state of PipelineRun
type ColocatedPipelineRunStatus struct {
	duckv1.Status `json:",inline"`

	// ColocatedPipelineRunStatusFields inlines the status fields.
	ColocatedPipelineRunStatusFields `json:",inline"`
}

// ColocatedPipelineRunStatusFields contains the status stored in the ExtraFields of a Run that references a ColocatedPipelineRun.
type ColocatedPipelineRunStatusFields struct {
	// StartTime is the time the ColocatedPipelineRun is actually started.
	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"`
	// CompletionTime is the time the ColocatedPipelineRun completed.
	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`
	// PipelineSpec contains the exact spec used to instantiate the ColocatedPipelineRun
	PipelineSpec *v1beta1.PipelineSpec `json:"pipelineSpec,omitempty"`
	// ChildStatuses
	ChildStatuses []ChildStatus `json:"childstatuses,omitempty"`
}

type ChildStatus struct {
	PipelineTaskName string            `json:"pipelineTaskName,omitempty"`
	Spec             *v1beta1.TaskSpec // TODO: custom tasks?
	StepStatuses     []v1beta1.StepState
	// TODO: task status in addition to the step statuses
	// TaskRunResults are the list of results written out by the task's containers
	// +optional
	TaskRunResults []v1beta1.TaskRunResult `json:"taskResults,omitempty"` // TODO: custom tasks?
}

// GetGroupVersionKind implements kmeta.OwnerRefable.
func (*ColocatedPipelineRun) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind(ControllerName)
}

// SetCondition sets the condition, unsetting previous conditions with the same
// type as necessary.
func (cpr *ColocatedPipelineRunStatus) SetCondition(newCond *apis.Condition) {
	if newCond != nil {
		cprCondSet.Manage(cpr).SetCondition(*newCond)
	}
}

// MarkFailed changes the Succeeded condition to False with the provided reason and message.
func (cpr *ColocatedPipelineRunStatus) MarkFailed(reason, messageFormat string, messageA ...interface{}) {
	cprCondSet.Manage(cpr).MarkFalse(apis.ConditionSucceeded, reason, messageFormat, messageA...)
	succeeded := cpr.GetCondition(apis.ConditionSucceeded)
	cpr.CompletionTime = &succeeded.LastTransitionTime.Inner
}

func (cpr *ColocatedPipelineRun) IsDone() bool {
	return !cpr.Status.GetCondition(apis.ConditionSucceeded).IsUnknown()
}

func (cpr *ColocatedPipelineRun) PipelineTimeout(ctx context.Context) time.Duration {
	if cpr.Spec.Timeouts != nil && cpr.Spec.Timeouts.Pipeline != nil {
		return cpr.Spec.Timeouts.Pipeline.Duration
	}
	return time.Duration(config.FromContextOrDefaults(ctx).Defaults.DefaultTimeoutMinutes) * time.Minute
}
