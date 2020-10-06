/*
Copyright 2020 The Tekton Authors

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

package v1alpha1

import (
	v1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TaskLoop iteratively executes a Task over elements in an array.
// +k8s:openapi-gen=true
type TaskLoop struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata"`

	// Spec holds the desired state of the TaskLoop from the client
	// +optional
	Spec TaskLoopSpec `json:"spec"`
}

// TaskLoopSpec defines the desired state of the TaskLoop
type TaskLoopSpec struct {
	// TaskRef is a reference to a task definition.
	// +optional
	TaskRef *v1beta1.TaskRef `json:"taskRef,omitempty"`

	// TaskSpec is a specification of a task
	// +optional
	TaskSpec *v1beta1.TaskSpec `json:"taskSpec,omitempty"`

	// IterateParam is the name of the task parameter that is iterated upon.
	IterateParam string `json:"iterateParam"`

	// Time after which the TaskRun times out.
	// +optional
	Timeout *metav1.Duration `json:"timeout,omitempty"`

	// Retries represents how many times a task should be retried in case of task failure.
	// +optional
	Retries int `json:"retries,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TaskLoopList contains a list of TaskLoops
type TaskLoopList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TaskLoop `json:"items"`
}

// TaskLoopRunReason represents a reason for the Run "Succeeded" condition
type TaskLoopRunReason string

const (
	// TaskLoopRunReasonStarted is the reason set when the Run has just started
	TaskLoopRunReasonStarted TaskLoopRunReason = "Started"

	// TaskLoopRunReasonRunning indicates that the Run is in progress
	TaskLoopRunReasonRunning TaskLoopRunReason = "Running"

	// TaskLoopRunReasonFailed indicates that one of the TaskRuns created from the Run failed
	TaskLoopRunReasonFailed TaskLoopRunReason = "Failed"

	// TaskLoopRunReasonSucceeded indicates that all of the TaskRuns created from the Run completed successfully
	TaskLoopRunReasonSucceeded TaskLoopRunReason = "Succeeded"

	// TaskLoopRunReasonCancelled indicates that a Run was cancelled.
	TaskLoopRunReasonCancelled TaskLoopRunReason = "TaskLoopRunCancelled"

	// TaskLoopRunReasonCouldntCancel indicates that a Run was cancelled but attempting to update
	// the running TaskRun as cancelled failed.
	TaskLoopRunReasonCouldntCancel TaskLoopRunReason = "TaskLoopRunCouldntCancel"

	// TaskLoopRunReasonCouldntGetTaskLoop indicates that the associated TaskLoop couldn't be retrieved
	TaskLoopRunReasonCouldntGetTaskLoop TaskLoopRunReason = "CouldntGetTaskLoop"

	// TaskLoopRunReasonFailedValidation indicates that the TaskLoop failed runtime validation
	TaskLoopRunReasonFailedValidation TaskLoopRunReason = "TaskLoopValidationFailed"

	// TaskLoopRunReasonInternalError indicates that the TaskLoop failed due to an internal error in the reconciler
	TaskLoopRunReasonInternalError TaskLoopRunReason = "TaskLoopInternalError"
)

func (t TaskLoopRunReason) String() string {
	return string(t)
}

// TaskLoopRunStatus contains the status stored in the ExtraFields of a Run that references a TaskLoop.
type TaskLoopRunStatus struct {
	// TaskLoopSpec contains the exact spec used to instantiate the Run
	TaskLoopSpec *TaskLoopSpec `json:"taskLoopSpec,omitempty"`
	// map of TaskLoopTaskRunStatus with the taskRun name as the key
	// +optional
	TaskRuns map[string]*TaskLoopTaskRunStatus `json:"taskRuns,omitempty"`
}

// TaskLoopTaskRunStatus contains the iteration number for a TaskRun and the TaskRun's Status
type TaskLoopTaskRunStatus struct {
	// iteration number
	Iteration int `json:"iteration,omitempty"`
	// Status is the TaskRunStatus for the corresponding TaskRun
	// +optional
	Status *v1beta1.TaskRunStatus `json:"status,omitempty"`
}
