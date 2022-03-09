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

package taskrun

import (
	"testing"
	"time"

	"knative.dev/pkg/apis"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/tektoncd/experimental/cloudevents/pkg/apis/config"
	"github.com/tektoncd/experimental/cloudevents/pkg/reconciler/events/cloudevent"
	cetest "github.com/tektoncd/experimental/cloudevents/test"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/test"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"
	"knative.dev/pkg/system"
	_ "knative.dev/pkg/system/testing"
)

// TestReconcile_CloudEvents runs reconcile with a cloud event sink configured
// to ensure that events are sent in different cases
func TestReconcile_CloudEvents(t *testing.T) {

	ignoreResourceVersion := cmpopts.IgnoreFields(v1beta1.TaskRun{}, "ObjectMeta.ResourceVersion")

	ts := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-task",
				Namespace: "foo",
			},
			Spec: v1beta1.TaskSpec{
				Steps: []v1beta1.Step{
					{
						Container: corev1.Container{
							Name:    "simple-step",
							Image:   "foo",
							Command: []string{"/mycmd"},
							Env: []corev1.EnvVar{
								{
									Name:  "foo",
									Value: "bar",
								},
							},
						},
					},
				},
			},
		},
	}
	cms := []*corev1.ConfigMap{
		{
			ObjectMeta: metav1.ObjectMeta{Name: config.GetDefaultsConfigName(), Namespace: system.Namespace()},
			Data: map[string]string{
				"default-cloud-events-sink": "http://synk:8080",
			},
		},
	}
	testcases := []struct {
		name            string
		condition       *apis.Condition
		wantCloudEvents []string
		startTime       bool
		annotations     map[string]string
		results         map[string]string
	}{{
		name:            "Task with no condition",
		condition:       nil,
		wantCloudEvents: []string{},
		startTime:       false,
	}, {
		name: "Task with running condition",
		condition: &apis.Condition{
			Type:   apis.ConditionSucceeded,
			Status: corev1.ConditionUnknown,
			Reason: v1beta1.TaskRunReasonRunning.String(),
		},
		startTime:       true,
		wantCloudEvents: []string{`(?s)cd.taskrun.started.v1.*test-taskrun`},
	}, {
		name: "Task with finished true condition",
		condition: &apis.Condition{
			Type:   apis.ConditionSucceeded,
			Status: corev1.ConditionTrue,
			Reason: v1beta1.PipelineRunReasonSuccessful.String(),
		},
		startTime:       true,
		wantCloudEvents: []string{`(?s)cd.taskrun.finished.v1.*test-taskrun`},
	}, {
		name: "Task with finished false condition",
		condition: &apis.Condition{
			Type:   apis.ConditionSucceeded,
			Status: corev1.ConditionFalse,
			Reason: v1beta1.PipelineRunReasonCancelled.String(),
		},
		startTime:       true,
		wantCloudEvents: []string{`(?s)cd.taskrun.finished.v1.*test-taskrun`},
	}, {
		name: "Task with finished successfully, artifact annotations",
		condition: &apis.Condition{
			Type:   apis.ConditionSucceeded,
			Status: corev1.ConditionTrue,
			Reason: v1beta1.PipelineRunReasonSuccessful.String(),
		},
		startTime: true,
		annotations: map[string]string{
			cloudevent.ArtifactPackagedEventAnnotation.String(): "",
		},
		results: map[string]string{
			"cd.artifact.id":      "456",
			"cd.artifact.name":    "image1",
			"cd.artifact.version": "v123",
		},
		wantCloudEvents: []string{
			`(?s)cd.taskrun.finished.v1.*test-taskrun`,
			`(?s)cd.artifact.packaged.v1.*image1.*test-taskrun`,
		},
	}}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {

			objectStatus := duckv1beta1.Status{
				Conditions: []apis.Condition{},
			}
			taskRunStatusFields := v1beta1.TaskRunStatusFields{}
			if tc.condition != nil {
				objectStatus.Conditions = append(objectStatus.Conditions, *tc.condition)
			}
			if tc.startTime {
				taskRunStatusFields.StartTime = &metav1.Time{Time: time.Now()}
			}
			tr := v1beta1.TaskRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-taskrun",
					Namespace: "foo",
				},
				Spec: v1beta1.TaskRunSpec{
					TaskRef: &v1beta1.TaskRef{
						Name: "test-task",
					},
				},
				Status: v1beta1.TaskRunStatus{
					Status:              objectStatus,
					TaskRunStatusFields: taskRunStatusFields,
				},
			}
			// Set annotations, if any
			if tc.annotations != nil {
				if tr.ObjectMeta.Annotations == nil {
					tr.ObjectMeta.Annotations = map[string]string{}
				}
				for k, v := range tc.annotations {
					tr.ObjectMeta.Annotations[k] = v
				}
			}
			// Set results, if any
			if tc.results != nil {
				for k, v := range tc.results {
					trr := v1beta1.TaskRunResult{Name: k, Value: v}
					tr.Status.TaskRunResults = append(tr.Status.TaskRunResults, trr)
				}
			}
			trs := []*v1beta1.TaskRun{&tr}

			d := test.Data{
				TaskRuns:   trs,
				Tasks:      ts,
				ConfigMaps: cms,
			}
			rt := cetest.NewReconcileTest(d, NewController, t)
			defer rt.Cancel()

			// Run the reconciler
			rt.ReconcileRun(t, "foo", "test-taskrun")

			uResource, err := rt.TestAssets.Clients.Pipeline.TektonV1beta1().TaskRuns("foo").Get(rt.TestAssets.Ctx, "test-taskrun", metav1.GetOptions{})
			if err != nil {
				t.Fatalf("getting updated resource: %v", err)
			}

			if d := cmp.Diff(&tr, uResource, ignoreResourceVersion); d != "" {
				t.Fatalf("run should not have changed, go %v instead", d)
			}

			// We get the client from the context, where it's automatically
			// injected. We use TestAssets from pipelinetest, and the logic
			// there relies on the pipeline cloudevents client
			ceClient := cloudevent.Get(rt.TestAssets.Ctx).(cloudevent.FakeClient)
			err = cetest.CheckEventsUnordered(t, ceClient.Events, tc.name, tc.wantCloudEvents)
			if err != nil {
				t.Errorf(err.Error())
			}
		})
	}
}
