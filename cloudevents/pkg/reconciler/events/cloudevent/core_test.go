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
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/test/diff"
	corev1 "k8s.io/api/core/v1"
)

const (
	coreRunName         = "fakerunname"
	coreTaskRunName     = "faketaskrunname"
	corePipelineRunName = "fakepipelinerunname"
)

func TestCoreEventForTaskRun(t *testing.T) {
	taskRunTests := []struct {
		desc          string
		taskRun       *v1beta1.TaskRun
		wantEventType cdeevents.CDEventType
	}{{
		desc:          "send a cloud event for a taskrun with no condition",
		taskRun:       createTaskRunWithCondition("", "doesn't matter"),
		wantEventType: cdeevents.TaskRunStartedEventV1,
	}, {
		desc:          "send a cloud event with unknown status taskrun",
		taskRun:       createTaskRunWithCondition(corev1.ConditionUnknown, "doesn't matter"),
		wantEventType: cdeevents.TaskRunStartedEventV1,
	}, {
		desc:          "send a cloud event with failed status taskrun",
		taskRun:       createTaskRunWithCondition(corev1.ConditionFalse, "meh"),
		wantEventType: cdeevents.TaskRunFinishedEventV1,
	}, {
		desc:          "send a cloud event with successful status taskrun",
		taskRun:       createTaskRunWithCondition(corev1.ConditionTrue, "yay"),
		wantEventType: cdeevents.TaskRunFinishedEventV1,
	}, {
		desc: "send a cloud event with successful status taskrun, empty selflink",
		taskRun: func() *v1beta1.TaskRun {
			tr := createTaskRunWithCondition(corev1.ConditionTrue, "yay")
			// v1.20 does not set selfLink in controller
			tr.ObjectMeta.SelfLink = ""
			return tr
		}(),
		wantEventType: cdeevents.TaskRunFinishedEventV1,
	}}

	for _, c := range taskRunTests {
		t.Run(c.desc, func(t *testing.T) {

			got, err := coreEventForObjectWithCondition(c.taskRun)
			if err != nil {
				// If not event type was set, don't expect an event
				if c.wantEventType != "" {
					t.Fatalf("I did not expect an error but I got %s", err)
				}
			} else {
				wantSubject := coreTaskRunName
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

func TestCoreEventForRun(t *testing.T) {
	runTests := []struct {
		desc          string
		run           *v1alpha1.Run
		wantEventType cdeevents.CDEventType
	}{{
		desc:          "send a cloud event with no condition",
		run:           createRunWithCondition("", "doesn't matter"),
		wantEventType: cdeevents.TaskRunStartedEventV1,
	}, {
		desc:          "send a cloud event with unknown status run",
		run:           createRunWithCondition(corev1.ConditionUnknown, "some condition"),
		wantEventType: cdeevents.TaskRunStartedEventV1,
	}, {
		desc:          "send a cloud event with failed status run",
		run:           createRunWithCondition(corev1.ConditionFalse, "meh"),
		wantEventType: cdeevents.TaskRunFinishedEventV1,
	}, {
		desc:          "send a cloud event with successful status taskrun",
		run:           createRunWithCondition(corev1.ConditionTrue, "yay"),
		wantEventType: cdeevents.TaskRunFinishedEventV1,
	}}

	for _, c := range runTests {
		t.Run(c.desc, func(t *testing.T) {

			got, err := coreEventForObjectWithCondition(c.run)
			if err != nil {
				// If not event type was set, don't expect an event
				if c.wantEventType != "" {
					t.Fatalf("I did not expect an error but I got %s", err)
				}
			} else {
				wantSubject := coreRunName
				if d := cmp.Diff(wantSubject, got.Subject()); d != "" {
					t.Errorf("Wrong Event ID %s", diff.PrintWantGot(d))
				}
				if d := cmp.Diff(string(c.wantEventType), got.Type()); d != "" {
					t.Errorf("Wrong Event Type %s", diff.PrintWantGot(d))
				}
				wantData, _ := getEventData(c.run)
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

func TestCoreEventForPipelineRun(t *testing.T) {
	pipelineRunTests := []struct {
		desc          string
		pipelineRun   *v1beta1.PipelineRun
		wantEventType cdeevents.CDEventType
		wantStatus    EventStatus
	}{{
		desc:          "send a cloud event with no condition, queued",
		pipelineRun:   createPipelineRunWithCondition("", "doesn't matter"),
		wantEventType: cdeevents.PipelineRunQueuedEventV1,
		wantStatus:    "",
	}, {
		desc:          "send a cloud event with unknown status pipelinerun, just started",
		pipelineRun:   createPipelineRunWithCondition(corev1.ConditionUnknown, v1beta1.PipelineRunReasonStarted.String()),
		wantEventType: cdeevents.PipelineRunQueuedEventV1,
		wantStatus:    StatusRunning,
	}, {
		desc:          "send a cloud event with unknown status pipelinerun, just started running",
		pipelineRun:   createPipelineRunWithCondition(corev1.ConditionUnknown, v1beta1.PipelineRunReasonRunning.String()),
		wantEventType: cdeevents.PipelineRunStartedEventV1,
		wantStatus:    StatusRunning,
	}, {
		desc:          "send a cloud event with unknown status pipelinerun",
		pipelineRun:   createPipelineRunWithCondition(corev1.ConditionUnknown, "doesn't matter"),
		wantEventType: cdeevents.PipelineRunStartedEventV1,
		wantStatus:    StatusRunning,
	}, {
		desc:          "send a cloud event with successful status pipelinerun",
		pipelineRun:   createPipelineRunWithCondition(corev1.ConditionTrue, "yay"),
		wantEventType: cdeevents.PipelineRunFinishedEventV1,
		wantStatus:    StatusFinished,
	}, {
		desc:          "send a cloud event with unknown status pipelinerun",
		pipelineRun:   createPipelineRunWithCondition(corev1.ConditionFalse, "meh"),
		wantEventType: cdeevents.PipelineRunFinishedEventV1,
		wantStatus:    StatusError,
	}}

	for _, c := range pipelineRunTests {
		t.Run(c.desc, func(t *testing.T) {

			got, err := coreEventForObjectWithCondition(c.pipelineRun)
			if err != nil {
				// If not event type was set, don't expect an event
				if c.wantEventType != "" {
					t.Fatalf("I did not expect an error but I got %s", err)
				}
			} else {
				wantSubject := corePipelineRunName
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
	eventType, err := coreEventForObjectWithCondition(myObjectWithCondition{})
	if err == nil {
		t.Fatalf("expected an error, got nil and eventType %s", eventType)
	}
}
