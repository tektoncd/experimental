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
	"context"
	"testing"

	lru "github.com/hashicorp/golang-lru"

	"github.com/tektoncd/experimental/cloudevents/pkg/reconciler/events/cache"
	eventstest "github.com/tektoncd/experimental/cloudevents/test/events"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"knative.dev/pkg/apis"
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"
	"knative.dev/pkg/controller"
	rtesting "knative.dev/pkg/reconciler/testing"
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

func setupFakeContext(t *testing.T, behaviour FakeClientBehaviour, withCEClient bool, withCacheClient bool) (context.Context, func()) {
	var ctx context.Context
	ctx, _ = rtesting.SetupFakeContext(t)
	if withCEClient {
		ctx = WithClient(ctx, &behaviour)
	}
	if withCacheClient {
		cacheClient, _ := lru.New(128)
		ctx = cache.ToContext(ctx, cacheClient)
	}
	ctx, cancel := context.WithCancel(ctx)
	return ctx, cancel
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
		name: "test-send-cloud-event-nothing-sent",
		clientBehaviour: FakeClientBehaviour{
			SendSuccessfully: true,
		},
		object: &v1beta1.PipelineRun{
			Status: v1beta1.PipelineRunStatus{},
		},
		wantCEvents:  []string{},
		wantEvents:   []string{},
		eventsFormat: "cdevents",
	}}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := setupFakeContext(t, tc.clientBehaviour, true, true)
			defer cancel()
			if err := SendCloudEventWithRetries(ctx, tc.object, tc.eventsFormat); err != nil {
				t.Fatalf("Unexpected error sending cloud events: %v", err)
			}
			// Try to second a second time to check that the cache is working
			if err := SendCloudEventWithRetries(ctx, tc.object, tc.eventsFormat); err != nil {
				t.Fatalf("Unexpected error sending cloud events: %v", err)
			}
			ceClient := Get(ctx).(FakeClient)
			if err := eventstest.CheckEventsUnordered(t, ceClient.Events, tc.name, tc.wantCEvents); err != nil {
				t.Fatalf(err.Error())
			}
			recorder := controller.GetEventRecorder(ctx).(*record.FakeRecorder)
			if err := eventstest.CheckEventsOrdered(t, recorder.Events, tc.name, tc.wantEvents); err != nil {
				t.Fatalf(err.Error())
			}
		})
	}
}
