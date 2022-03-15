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

	cetest "github.com/tektoncd/experimental/cloudevents/test"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"
	"knative.dev/pkg/controller"
)

const (
	taskRunName     = "faketaskrunname"
	pipelineRunName = "fakepipelinerunname"
	runName         = "fakerunname"
)

func createTaskRunWithCondition(status corev1.ConditionStatus, reason string) *v1beta1.TaskRun {
	mytaskrun := &v1beta1.TaskRun{
		TypeMeta: metav1.TypeMeta{
			Kind:       "TaskRun",
			APIVersion: "v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      taskRunName,
			Namespace: "marshmallow",
		},
		Spec: v1beta1.TaskRunSpec{},
	}
	switch status {
	case corev1.ConditionFalse, corev1.ConditionUnknown, corev1.ConditionTrue:
		mytaskrun.Status = v1beta1.TaskRunStatus{
			Status: duckv1beta1.Status{
				Conditions: []apis.Condition{{
					Type:   apis.ConditionSucceeded,
					Status: status,
					Reason: reason,
				}},
			},
		}
	}
	return mytaskrun
}

func createPipelineRunWithCondition(status corev1.ConditionStatus, reason string) *v1beta1.PipelineRun {
	mypipelinerun := &v1beta1.PipelineRun{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PipelineRun",
			APIVersion: "v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      pipelineRunName,
			Namespace: "marshmallow",
		},
		Spec: v1beta1.PipelineRunSpec{},
	}
	switch status {
	case corev1.ConditionFalse, corev1.ConditionUnknown, corev1.ConditionTrue:
		mypipelinerun.Status = v1beta1.PipelineRunStatus{
			Status: duckv1beta1.Status{
				Conditions: []apis.Condition{{
					Type:   apis.ConditionSucceeded,
					Status: status,
					Reason: reason,
				}},
			},
		}
	}
	return mypipelinerun
}

func createRunWithCondition(status corev1.ConditionStatus, reason string) *v1alpha1.Run {
	myrun := &v1alpha1.Run{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Run",
			APIVersion: "v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      runName,
			Namespace: "marshmallow",
		},
		Spec: v1alpha1.RunSpec{},
	}
	switch status {
	case corev1.ConditionFalse, corev1.ConditionUnknown, corev1.ConditionTrue:
		myrun.Status = v1alpha1.RunStatus{
			Status: duckv1.Status{
				Conditions: []apis.Condition{{
					Type:   apis.ConditionSucceeded,
					Status: status,
					Reason: reason,
				}},
			},
		}
	}
	return myrun
}

func TestSendCloudEventWithRetries(t *testing.T) {

	objectStatus := duckv1beta1.Status{
		Conditions: []apis.Condition{{
			Type:   apis.ConditionSucceeded,
			Status: corev1.ConditionTrue,
		}},
	}

	tests := []struct {
		name            string
		clientBehaviour FakeClientBehaviour
		object          objectWithCondition
		wantCEvents     []string
		wantEvents      []string
		eventsFormat    string
	}{{
		name: "test-send-cloud-event-taskrun",
		clientBehaviour: FakeClientBehaviour{
			SendSuccessfully: true,
		},
		object: &v1beta1.TaskRun{
			ObjectMeta: metav1.ObjectMeta{
				SelfLink: "/taskruns/test1",
			},
			Status: v1beta1.TaskRunStatus{Status: objectStatus},
		},
		wantCEvents:  []string{"cd.taskrun.finished"},
		wantEvents:   []string{},
		eventsFormat: "cdevents",
	}, {
		name: "test-send-cloud-event-taskrun-legacy",
		clientBehaviour: FakeClientBehaviour{
			SendSuccessfully: true,
		},
		object: &v1beta1.TaskRun{
			ObjectMeta: metav1.ObjectMeta{
				SelfLink: "/taskruns/test1",
			},
			Status: v1beta1.TaskRunStatus{Status: objectStatus},
		},
		wantCEvents:  []string{"dev.tekton.event.taskrun.successful.v1"},
		wantEvents:   []string{},
		eventsFormat: "legacy",
	}, {
		name: "test-send-cloud-event-pipelinerun",
		clientBehaviour: FakeClientBehaviour{
			SendSuccessfully: true,
		},
		object: &v1beta1.PipelineRun{
			ObjectMeta: metav1.ObjectMeta{
				SelfLink: "/pipelineruns/test1",
			},
			Status: v1beta1.PipelineRunStatus{Status: objectStatus},
		},
		wantCEvents:  []string{"Context Attributes,"},
		wantEvents:   []string{},
		eventsFormat: "legacy",
	}, {
		name: "test-send-cloud-event-failed",
		clientBehaviour: FakeClientBehaviour{
			SendSuccessfully: false,
		},
		object: &v1beta1.PipelineRun{
			Status: v1beta1.PipelineRunStatus{Status: objectStatus},
		},
		wantCEvents:  []string{},
		wantEvents:   []string{"Warning Cloud Event Failure"},
		eventsFormat: "legacy",
	}, {
		name: "test-send-cloud-event-nothing-sent",
		clientBehaviour: FakeClientBehaviour{
			SendSuccessfully: true,
		},
		object: &v1beta1.PipelineRun{
			Status: v1beta1.PipelineRunStatus{},
		},
		wantCEvents:  []string{},
		wantEvents:   []string{},
		eventsFormat: "legacy",
	}, {
		name: "test-send-cloud-event-queued",
		clientBehaviour: FakeClientBehaviour{
			SendSuccessfully: true,
		},
		object: &v1beta1.PipelineRun{
			Status: v1beta1.PipelineRunStatus{},
		},
		wantCEvents:  []string{"cd.pipelinerun.queued"},
		wantEvents:   []string{},
		eventsFormat: "cdevents",
	}}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx, _ := cetest.SetupFakeContext(t)
			// Override the client to set the behaviour from the test
			ctx = WithClient(ctx, &tc.clientBehaviour)
			if err := SendCloudEventWithRetries(ctx, tc.object, tc.eventsFormat); err != nil {
				t.Fatalf("Unexpected error sending cloud events: %v", err)
			}
			ceClient := Get(ctx).(FakeClient)
			if err := cetest.CheckEventsUnordered(t, ceClient.Events, tc.name, tc.wantCEvents); err != nil {
				t.Fatalf(err.Error())
			}
			recorder := controller.GetEventRecorder(ctx).(*record.FakeRecorder)
			if err := cetest.CheckEventsOrdered(t, recorder.Events, tc.name, tc.wantEvents); err != nil {
				t.Fatalf(err.Error())
			}
			// Try to second a second time to check that the cache is working
			if err := SendCloudEventWithRetries(ctx, tc.object, tc.eventsFormat); err != nil {
				t.Fatalf("Unexpected error sending cloud events: %v", err)
			}
			if err := cetest.CheckEventsUnordered(t, ceClient.Events, tc.name, []string{}); err != nil {
				t.Fatalf(err.Error())
			}
		})
	}
}
