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

package cache

import (
	"encoding/json"
	lru "github.com/hashicorp/golang-lru"
	"net/url"
	"testing"
	"time"

	"github.com/cloudevents/sdk-go/v2/event"
	cetypes "github.com/cloudevents/sdk-go/v2/types"
	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/test/diff"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func strptr(s string) *string { return &s }

func getEventData(run interface{}) map[string]string {
	cdeCloudEventData := map[string]string{}
	switch v := run.(type) {
	case *v1beta1.TaskRun:
		data, err := json.Marshal(v)
		if err != nil {
			panic(err)
		}
		cdeCloudEventData["taskrun"] = string(data)
	case *v1beta1.PipelineRun:
		data, err := json.Marshal(v)
		if err != nil {
			panic(err)
		}
		cdeCloudEventData["pipelinerun"] = string(data)
	}
	return cdeCloudEventData
}

func getEventToTest(eventtype string, run interface{}) *event.Event {
	e := event.Event{
		Context: event.EventContextV1{
			Type:    eventtype,
			Source:  cetypes.URIRef{URL: url.URL{Path: "/foo/bar/source"}},
			ID:      "test-event",
			Time:    &cetypes.Timestamp{Time: time.Now()},
			Subject: strptr("topic"),
		}.AsV1(),
	}
	if err := e.SetData("text/json", getEventData(run)); err != nil {
		panic(err)
	}
	return &e
}

func getTaskRunByMeta(name string, namespace string) *v1beta1.TaskRun {
	return &v1beta1.TaskRun{
		TypeMeta: metav1.TypeMeta{
			Kind:       "TaskRun",
			APIVersion: "v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec:   v1beta1.TaskRunSpec{},
		Status: v1beta1.TaskRunStatus{},
	}
}

func getPipelineRunByMeta(name string, namespace string) *v1beta1.PipelineRun {
	return &v1beta1.PipelineRun{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PipelineRun",
			APIVersion: "v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec:   v1beta1.PipelineRunSpec{},
		Status: v1beta1.PipelineRunStatus{},
	}
}

// TestEventsKey verifies that keys are extracted correctly from events
func TestEventsKey(t *testing.T) {
	testcases := []struct {
		name      string
		eventtype string
		run       interface{}
		wantKey   string
	}{{
		name:      "taskrun event",
		eventtype: "my.test.taskrun.event",
		run:       getTaskRunByMeta("mytaskrun", "mynamespace"),
		wantKey:   "my.test.taskrun.event/taskrun/mynamespace/mytaskrun",
	}, {
		name:      "pipelinerun event",
		eventtype: "my.test.pipelinerun.event",
		run:       getPipelineRunByMeta("mypipelinerun", "mynamespace"),
		wantKey:   "my.test.pipelinerun.event/pipelinerun/mynamespace/mypipelinerun",
	}}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			gotEvent := getEventToTest(tc.eventtype, tc.run)
			gotKey := EventKey(gotEvent)
			if d := cmp.Diff(tc.wantKey, gotKey); d != "" {
				t.Errorf("Wrong Event key %s", diff.PrintWantGot(d))
			}
		})
	}
}

func TestAddCheckEvent(t *testing.T) {
	run := getTaskRunByMeta("arun", "anamespace")
	runb := getTaskRunByMeta("arun", "bnamespace")
	pipelinerun := getPipelineRunByMeta("arun", "anamespace")
	baseEvent := getEventToTest("some.event.type", run)

	testcases := []struct {
		name        string
		firstEvent  *event.Event
		secondEvent *event.Event
		wantFound   bool
	}{{
		name:        "identical events",
		firstEvent:  baseEvent,
		secondEvent: baseEvent,
		wantFound:   true,
	}, {
		name:        "new timestamp event",
		firstEvent:  baseEvent,
		secondEvent: getEventToTest("some.event.type", run),
		wantFound:   true,
	}, {
		name:        "different namespace",
		firstEvent:  baseEvent,
		secondEvent: getEventToTest("some.event.type", runb),
		wantFound:   false,
	}, {
		name:        "different resource type",
		firstEvent:  baseEvent,
		secondEvent: getEventToTest("some.event.type", pipelinerun),
		wantFound:   false,
	}, {
		name:        "different event type",
		firstEvent:  baseEvent,
		secondEvent: getEventToTest("some.other.event.type", run),
		wantFound:   false,
	}}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			testCache, _ := lru.New(10)
			AddEventSentToCache(testCache, tc.firstEvent)
			found, _ := IsCloudEventSent(testCache, tc.secondEvent)
			if d := cmp.Diff(tc.wantFound, found); d != "" {
				t.Errorf("Cache check failure %s", diff.PrintWantGot(d))
			}
		})
	}
}
