/*
Copyright 2019 The Tekton Authors

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

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/test/diff"
	corev1 "k8s.io/api/core/v1"
)

func TestEventForTaskRun(t *testing.T) {
	taskRunTests := []struct {
		desc          string
		taskRun       *v1beta1.TaskRun
		wantEventType TektonEventType
	}{{
		desc:          "send a cloud event when a taskrun starts",
		taskRun:       createTaskRunWithCondition(corev1.ConditionUnknown, v1beta1.TaskRunReasonStarted.String()),
		wantEventType: TaskRunStartedEventV1,
	}, {
		desc:          "send a cloud event when a taskrun starts running",
		taskRun:       createTaskRunWithCondition(corev1.ConditionUnknown, v1beta1.TaskRunReasonRunning.String()),
		wantEventType: TaskRunRunningEventV1,
	}, {
		desc:          "send a cloud event with unknown status taskrun",
		taskRun:       createTaskRunWithCondition(corev1.ConditionUnknown, "doesn't matter"),
		wantEventType: TaskRunUnknownEventV1,
	}, {
		desc:          "send a cloud event with failed status taskrun",
		taskRun:       createTaskRunWithCondition(corev1.ConditionFalse, "meh"),
		wantEventType: TaskRunFailedEventV1,
	}, {
		desc:          "send a cloud event with successful status taskrun",
		taskRun:       createTaskRunWithCondition(corev1.ConditionTrue, "yay"),
		wantEventType: TaskRunSuccessfulEventV1,
	}, {
		desc: "send a cloud event with successful status taskrun, empty selflink",
		taskRun: func() *v1beta1.TaskRun {
			tr := createTaskRunWithCondition(corev1.ConditionTrue, "yay")
			// v1.20 does not set selfLink in controller
			tr.ObjectMeta.SelfLink = ""
			return tr
		}(),
		wantEventType: TaskRunSuccessfulEventV1,
	}}

	for _, c := range taskRunTests {
		t.Run(c.desc, func(t *testing.T) {

			got, err := eventForTaskRun(c.taskRun)
			if err != nil {
				t.Fatalf("I did not expect an error but I got %s", err)
			} else {
				wantSubject := taskRunName
				if d := cmp.Diff(wantSubject, got.Subject()); d != "" {
					t.Errorf("Wrong Event ID %s", diff.PrintWantGot(d))
				}
				if d := cmp.Diff(string(c.wantEventType), got.Type()); d != "" {
					t.Errorf("Wrong Event Type %s", diff.PrintWantGot(d))
				}
				wantData := newTektonCloudEventData(c.taskRun)
				gotData := TektonCloudEventData{}
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
		wantEventType TektonEventType
	}{{
		desc:          "send a cloud event with unknown status pipelinerun, just started",
		pipelineRun:   createPipelineRunWithCondition(corev1.ConditionUnknown, v1beta1.PipelineRunReasonStarted.String()),
		wantEventType: PipelineRunStartedEventV1,
	}, {
		desc:          "send a cloud event with unknown status pipelinerun, just started running",
		pipelineRun:   createPipelineRunWithCondition(corev1.ConditionUnknown, v1beta1.PipelineRunReasonRunning.String()),
		wantEventType: PipelineRunRunningEventV1,
	}, {
		desc:          "send a cloud event with unknown status pipelinerun",
		pipelineRun:   createPipelineRunWithCondition(corev1.ConditionUnknown, "doesn't matter"),
		wantEventType: PipelineRunUnknownEventV1,
	}, {
		desc:          "send a cloud event with successful status pipelinerun",
		pipelineRun:   createPipelineRunWithCondition(corev1.ConditionTrue, "yay"),
		wantEventType: PipelineRunSuccessfulEventV1,
	}, {
		desc:          "send a cloud event with unknown status pipelinerun",
		pipelineRun:   createPipelineRunWithCondition(corev1.ConditionFalse, "meh"),
		wantEventType: PipelineRunFailedEventV1,
	}}

	for _, c := range pipelineRunTests {
		t.Run(c.desc, func(t *testing.T) {

			got, err := eventForPipelineRun(c.pipelineRun)
			if err != nil {
				t.Fatalf("I did not expect an error but I got %s", err)
			} else {
				wantSubject := pipelineRunName
				if d := cmp.Diff(wantSubject, got.Subject()); d != "" {
					t.Errorf("Wrong Event ID %s", diff.PrintWantGot(d))
				}
				if d := cmp.Diff(string(c.wantEventType), got.Type()); d != "" {
					t.Errorf("Wrong Event Type %s", diff.PrintWantGot(d))
				}
				wantData := newTektonCloudEventData(c.pipelineRun)
				gotData := TektonCloudEventData{}
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

func TestEventForRun(t *testing.T) {
	runTests := []struct {
		desc          string
		run           *v1alpha1.Run
		wantEventType TektonEventType
	}{{
		desc:          "send a cloud event with unset condition, just started",
		run:           createRunWithCondition("", ""),
		wantEventType: RunStartedEventV1,
	}, {
		desc:          "send a cloud event with unknown status run, empty reason",
		run:           createRunWithCondition(corev1.ConditionUnknown, ""),
		wantEventType: RunRunningEventV1,
	}, {
		desc:          "send a cloud event with unknown status run, some reason set",
		run:           createRunWithCondition(corev1.ConditionUnknown, "custom controller reason"),
		wantEventType: RunRunningEventV1,
	}, {
		desc:          "send a cloud event with successful status run",
		run:           createRunWithCondition(corev1.ConditionTrue, "yay"),
		wantEventType: RunSuccessfulEventV1,
	}, {
		desc:          "send a cloud event with unknown status run",
		run:           createRunWithCondition(corev1.ConditionFalse, "meh"),
		wantEventType: RunFailedEventV1,
	}}

	for _, c := range runTests {
		t.Run(c.desc, func(t *testing.T) {

			got, err := eventForRun(c.run)
			if err != nil {
				t.Fatalf("I did not expect an error but I got %s", err)
			} else {
				wantSubject := runName
				if d := cmp.Diff(wantSubject, got.Subject()); d != "" {
					t.Errorf("Wrong Event ID %s", diff.PrintWantGot(d))
				}
				if d := cmp.Diff(string(c.wantEventType), got.Type()); d != "" {
					t.Errorf("Wrong Event Type %s", diff.PrintWantGot(d))
				}
				wantData := newTektonCloudEventData(c.run)
				gotData := TektonCloudEventData{}
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
