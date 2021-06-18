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
	"testing"

	cdeevents "github.com/cdfoundation/sig-events/cde/sdk/go/pkg/cdf/events"
	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/test/diff"
	"github.com/tektoncd/pipeline/test/names"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"
)

const (
	taskRunName     = "faketaskrunname"
	pipelineRunName = "fakepipelinerunname"
)

func getTaskRunByCondition(status corev1.ConditionStatus, reason string) *v1beta1.TaskRun {
	return &v1beta1.TaskRun{
		TypeMeta: metav1.TypeMeta{
			Kind:       "TaskRun",
			APIVersion: "v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      taskRunName,
			Namespace: "marshmallow",
		},
		Spec: v1beta1.TaskRunSpec{},
		Status: v1beta1.TaskRunStatus{
			Status: duckv1beta1.Status{
				Conditions: []apis.Condition{{
					Type:   apis.ConditionSucceeded,
					Status: status,
					Reason: reason,
				}},
			},
		},
	}
}

func getPipelineRunByCondition(status corev1.ConditionStatus, reason string) *v1beta1.PipelineRun {
	return &v1beta1.PipelineRun{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PipelineRun",
			APIVersion: "v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      pipelineRunName,
			Namespace: "marshmallow",
		},
		Spec: v1beta1.PipelineRunSpec{},
		Status: v1beta1.PipelineRunStatus{
			Status: duckv1beta1.Status{
				Conditions: []apis.Condition{{
					Type:   apis.ConditionSucceeded,
					Status: status,
					Reason: reason,
				}},
			},
		},
	}
}

func TestEventForTaskRun(t *testing.T) {
	taskRunTests := []struct {
		desc          string
		taskRun       *v1beta1.TaskRun
		wantEventType cdeevents.CDEventType
	}{{
		desc:          "send a cloud event when a taskrun starts",
		taskRun:       getTaskRunByCondition(corev1.ConditionUnknown, v1beta1.TaskRunReasonStarted.String()),
		wantEventType: cdeevents.TaskRunStartedEventV1,
	}, {
		desc:          "send a cloud event when a taskrun starts running",
		taskRun:       getTaskRunByCondition(corev1.ConditionUnknown, v1beta1.TaskRunReasonRunning.String()),
		wantEventType: cdeevents.TaskRunStartedEventV1,
	}, {
		desc:    "send a cloud event with unknown status taskrun",
		taskRun: getTaskRunByCondition(corev1.ConditionUnknown, "doesn't matter"),
	}, {
		desc:          "send a cloud event with failed status taskrun",
		taskRun:       getTaskRunByCondition(corev1.ConditionFalse, "meh"),
		wantEventType: cdeevents.TaskRunFinishedEventV1,
	}, {
		desc:          "send a cloud event with successful status taskrun",
		taskRun:       getTaskRunByCondition(corev1.ConditionTrue, "yay"),
		wantEventType: cdeevents.TaskRunFinishedEventV1,
	}, {
		desc: "send a cloud event with successful status taskrun, empty selflink",
		taskRun: func() *v1beta1.TaskRun {
			tr := getTaskRunByCondition(corev1.ConditionTrue, "yay")
			// v1.20 does not set selfLink in controller
			tr.ObjectMeta.SelfLink = ""
			return tr
		}(),
		wantEventType: cdeevents.TaskRunFinishedEventV1,
	}}

	for _, c := range taskRunTests {
		t.Run(c.desc, func(t *testing.T) {
			names.TestingSeed()

			got, err := eventForObjectWithCondition(c.taskRun)
			if err != nil {
				// If not event type was set, don't expect an event
				if c.wantEventType != "" {
					t.Fatalf("I did not expect an error but I got %s", err)
				}
			} else {
				wantSubject := taskRunName
				if d := cmp.Diff(wantSubject, got.Subject()); d != "" {
					t.Errorf("Wrong Event ID %s", diff.PrintWantGot(d))
				}
				if d := cmp.Diff(string(c.wantEventType), got.Type()); d != "" {
					t.Errorf("Wrong Event Type %s", diff.PrintWantGot(d))
				}
				wantData, _ := getEventData(c.taskRun)
				gotData := CDECloudEventData{}
				if err := got.DataAs(&gotData); err != nil {
					t.Errorf("Unexpected error from DataAsl; %s", err)
				}
				if d := cmp.Diff(wantData, gotData); d != "" {
					t.Errorf("Wrong Event data %s", diff.PrintWantGot(d))
				}

				if err := got.Validate(); err != nil {
					t.Errorf("Expected event to be valid; %s", err)
				}
			}
		})
	}
}

func TestEventForPipelineRun(t *testing.T) {
	pipelineRunTests := []struct {
		desc          string
		pipelineRun   *v1beta1.PipelineRun
		wantEventType cdeevents.CDEventType
		wantStatus    EventStatus
	}{{
		desc:          "send a cloud event with unknown status pipelinerun, just started",
		pipelineRun:   getPipelineRunByCondition(corev1.ConditionUnknown, v1beta1.PipelineRunReasonStarted.String()),
		wantEventType: cdeevents.PipelineRunQueuedEventV1,
		wantStatus:    StatusRunning,
	}, {
		desc:          "send a cloud event with unknown status pipelinerun, just started running",
		pipelineRun:   getPipelineRunByCondition(corev1.ConditionUnknown, v1beta1.PipelineRunReasonRunning.String()),
		wantEventType: cdeevents.PipelineRunStartedEventV1,
		wantStatus:    StatusRunning,
	}, {
		desc:        "send a cloud event with unknown status pipelinerun",
		pipelineRun: getPipelineRunByCondition(corev1.ConditionUnknown, "doesn't matter"),
	}, {
		desc:          "send a cloud event with successful status pipelinerun",
		pipelineRun:   getPipelineRunByCondition(corev1.ConditionTrue, "yay"),
		wantEventType: cdeevents.PipelineRunFinishedEventV1,
		wantStatus:    StatusFinished,
	}, {
		desc:          "send a cloud event with unknown status pipelinerun",
		pipelineRun:   getPipelineRunByCondition(corev1.ConditionFalse, "meh"),
		wantEventType: cdeevents.PipelineRunFinishedEventV1,
		wantStatus:    StatusError,
	}}

	for _, c := range pipelineRunTests {
		t.Run(c.desc, func(t *testing.T) {
			names.TestingSeed()

			got, err := eventForObjectWithCondition(c.pipelineRun)
			if err != nil {
				// If not event type was set, don't expect an event
				if c.wantEventType != "" {
					t.Fatalf("I did not expect an error but I got %s", err)
				}
			} else {
				wantSubject := pipelineRunName
				if d := cmp.Diff(wantSubject, got.Subject()); d != "" {
					t.Errorf("Wrong Event ID %s", diff.PrintWantGot(d))
				}
				if d := cmp.Diff(string(c.wantEventType), got.Type()); d != "" {
					t.Errorf("Wrong Event Type %s", diff.PrintWantGot(d))
				}
				if d := cmp.Diff(string(c.wantStatus), got.Extensions()["pipelinerunstatus"]); d != "" {
					t.Errorf("Wrong Event Type %s", diff.PrintWantGot(d))
				}
				wantData, _ := getEventData(c.pipelineRun)
				gotData := CDECloudEventData{}
				if err := got.DataAs(&gotData); err != nil {
					t.Errorf("Unexpected error from DataAsl; %s", err)
				}
				if d := cmp.Diff(wantData, gotData); d != "" {
					t.Errorf("Wrong Event data %s", diff.PrintWantGot(d))
				}

				if err := got.Validate(); err != nil {
					t.Errorf("Expected event to be valid; %s", err)
				}
			}
		})
	}
}

func TestEventTypeInvalidType(t *testing.T) {
	eventType, err := eventForObjectWithCondition(myObjectWithCondition{})
	if err == nil {
		t.Fatalf("expected an error, got nil and eventType %s", eventType)
	}
}
