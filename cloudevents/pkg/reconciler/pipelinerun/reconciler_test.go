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

package pipelinerun

import (
	"context"
	"fmt"
	"regexp"
	"testing"
	"time"

	lru "github.com/hashicorp/golang-lru"
	"knative.dev/pkg/apis"

	"github.com/tektoncd/experimental/cloudevents/pkg/apis/config"
	"github.com/tektoncd/experimental/cloudevents/pkg/reconciler/events/cache"
	"github.com/tektoncd/experimental/cloudevents/pkg/reconciler/events/cloudevent"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	pipelinecloudevent "github.com/tektoncd/pipeline/pkg/reconciler/events/cloudevent"
	ttesting "github.com/tektoncd/pipeline/pkg/reconciler/testing"
	"github.com/tektoncd/pipeline/test"
	"github.com/tektoncd/pipeline/test/names"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/clock"
	"k8s.io/client-go/tools/record"
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"
	cminformer "knative.dev/pkg/configmap/informer"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/reconciler"
	"knative.dev/pkg/system"
	_ "knative.dev/pkg/system/testing"
)

var (
	now       = time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC)
	testClock = clock.NewFakePassiveClock(now)
)

type PipelineRunTest struct {
	test.Data  `json:"inline"`
	Test       *testing.T
	TestAssets test.Assets
	Cancel     func()
}

func (prt PipelineRunTest) reconcileRun(namespace, pipelineRunName string, permanentError bool) (*v1beta1.PipelineRun, test.Clients) {
	prt.Test.Helper()
	c := prt.TestAssets.Controller
	clients := prt.TestAssets.Clients

	reconcileError := c.Reconciler.Reconcile(prt.TestAssets.Ctx, namespace+"/"+pipelineRunName)
	if permanentError {
		// When a PipelineRun is invalid and can't run, we expect a permanent error that will
		// tell the Reconciler to not keep trying to reconcile.
		if reconcileError == nil {
			prt.Test.Fatalf("Expected an error to be returned by Reconcile, got nil instead")
		}
		if controller.IsPermanentError(reconcileError) != permanentError {
			prt.Test.Fatalf("Expected the error to be permanent: %v but got: %v", permanentError, controller.IsPermanentError(reconcileError))
		}
	} else if reconcileError != nil {
		prt.Test.Fatalf("Error reconciling: %s", reconcileError)
	}
	// Check that the PipelineRun was reconciled correctly
	reconciledRun, err := clients.Pipeline.TektonV1beta1().PipelineRuns(namespace).Get(prt.TestAssets.Ctx, pipelineRunName, metav1.GetOptions{})
	if err != nil {
		prt.Test.Fatalf("Somehow had error getting reconciled run out of fake client: %s", err)
	}

	return reconciledRun, clients
}

func checkCloudEvents(t *testing.T, fce *pipelinecloudevent.FakeClient, testName string, wantEvents []string) error {
	t.Helper()
	return eventFromChannel(fce.Events, testName, wantEvents)
}

func eventFromChannel(c chan string, testName string, wantEvents []string) error {
	// We get events from a channel, so the timeout is here to avoid waiting
	// on the channel forever if fewer than expected events are received.
	// We only hit the timeout in case of failure of the test, so the actual value
	// of the timeout is not so relevant, it's only used when tests are going to fail.
	// on the channel forever if fewer than expected events are received
	timer := time.NewTimer(10 * time.Millisecond)
	foundEvents := []string{}
	for ii := 0; ii < len(wantEvents)+1; ii++ {
		// We loop over all the events that we expect. Once they are all received
		// we exit the loop. If we never receive enough events, the timeout takes us
		// out of the loop.
		select {
		case event := <-c:
			foundEvents = append(foundEvents, event)
			if ii > len(wantEvents)-1 {
				return fmt.Errorf("received event \"%s\" for %s but not more expected", event, testName)
			}
			wantEvent := wantEvents[ii]
			matching, err := regexp.MatchString(wantEvent, event)
			if err == nil {
				if !matching {
					return fmt.Errorf("expected event \"%s\" but got \"%s\" instead for %s", wantEvent, event, testName)
				}
			} else {
				return fmt.Errorf("something went wrong matching the event: %s", err)
			}
		case <-timer.C:
			if len(foundEvents) > len(wantEvents) {
				return fmt.Errorf("received %d events for %s but %d expected. Found events: %#v", len(foundEvents), testName, len(wantEvents), foundEvents)
			}
		}
	}
	return nil
}

func ensureConfigurationConfigMapsExist(d *test.Data) {
	var defaultsExists bool
	for _, cm := range d.ConfigMaps {
		if cm.Name == config.GetDefaultsConfigName() {
			defaultsExists = true
		}
	}
	if !defaultsExists {
		d.ConfigMaps = append(d.ConfigMaps, &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: config.GetDefaultsConfigName(), Namespace: system.Namespace()},
			Data:       map[string]string{},
		})
	}
}

// getPipelineRunController returns an instance of the PipelineRun controller/reconciler that has been seeded with
// d, where d represents the state of the system (existing resources) needed for the test.
func getPipelineRunController(t *testing.T, d test.Data) (test.Assets, func()) {
	// unregisterMetrics()
	ctx, _ := ttesting.SetupFakeContext(t)
	cacheClient, _ := lru.New(128)
	ctx = cache.ToContext(ctx, cacheClient)
	ctx, cancel := context.WithCancel(ctx)
	ensureConfigurationConfigMapsExist(&d)
	c, informers := test.SeedTestData(t, ctx, d)
	configMapWatcher := cminformer.NewInformedWatcher(c.Kube, system.Namespace())

	ctl := NewController(testClock)(ctx, configMapWatcher)

	if la, ok := ctl.Reconciler.(reconciler.LeaderAware); ok {
		la.Promote(reconciler.UniversalBucket(), func(reconciler.Bucket, types.NamespacedName) {})
	}
	if err := configMapWatcher.Start(ctx.Done()); err != nil {
		t.Fatalf("error starting configmap watcher: %v", err)
	}

	return test.Assets{
		Logger:     logging.FromContext(ctx),
		Controller: ctl,
		Clients:    c,
		Informers:  informers,
		Recorder:   controller.GetEventRecorder(ctx).(*record.FakeRecorder),
		Ctx:        ctx,
	}, cancel
}

// newPipelineRunTest returns PipelineRunTest with a new PipelineRun controller created with specified state through data
// This PipelineRunTest can be reused for multiple PipelineRuns by calling reconcileRun for each pipelineRun
func newPipelineRunTest(data test.Data, t *testing.T) *PipelineRunTest {
	t.Helper()
	testAssets, cancel := getPipelineRunController(t, data)
	return &PipelineRunTest{
		Data:       data,
		Test:       t,
		TestAssets: testAssets,
		Cancel:     cancel,
	}
}

// TestReconcile_CloudEvents runs reconcile with a cloud event sink configured
// to ensure that events are sent in different cases
func TestReconcile_CloudEvents(t *testing.T) {
	names.TestingSeed()

	ps := []*v1beta1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pipeline",
				Namespace: "foo",
			},
			Spec: v1beta1.PipelineSpec{
				Tasks: []v1beta1.PipelineTask{
					{
						Name: "test-1",
						TaskRef: &v1beta1.TaskRef{
							Name: "test-task",
						},
					},
				},
			},
		},
	}
	ts := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pipeline",
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
		name:            "Pipeline with no condition",
		condition:       nil,
		wantCloudEvents: []string{`(?s)cd.pipelinerun.queued.v1.*test-pipelinerun`},
		startTime:       false,
	}, {
		name: "Pipeline with running condition",
		condition: &apis.Condition{
			Type:   apis.ConditionSucceeded,
			Status: corev1.ConditionUnknown,
			Reason: v1beta1.PipelineRunReasonRunning.String(),
		},
		startTime:       true,
		wantCloudEvents: []string{`(?s)cd.pipelinerun.started.v1.*test-pipelinerun`},
	}, {
		name: "Pipeline with finished true condition",
		condition: &apis.Condition{
			Type:   apis.ConditionSucceeded,
			Status: corev1.ConditionTrue,
			Reason: v1beta1.PipelineRunReasonSuccessful.String(),
		},
		startTime:       true,
		wantCloudEvents: []string{`(?s)cd.pipelinerun.finished.v1.*test-pipelinerun`},
	}, {
		name: "Pipeline with finished false condition",
		condition: &apis.Condition{
			Type:   apis.ConditionSucceeded,
			Status: corev1.ConditionFalse,
			Reason: v1beta1.PipelineRunReasonCancelled.String(),
		},
		startTime:       true,
		wantCloudEvents: []string{`(?s)cd.pipelinerun.finished.v1.*test-pipelinerun`},
	}, {
		name: "Pipeline with finished successfully, artifact annotations",
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
			`(?s)cd.pipelinerun.finished.v1.*test-pipelinerun`,
			`(?s)cd.artifact.packaged.v1.*image1.*test-pipelinerun`,
		},
	}}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {

			objectStatus := duckv1beta1.Status{
				Conditions: []apis.Condition{},
			}
			pipelineStatusFields := v1beta1.PipelineRunStatusFields{}
			if tc.condition != nil {
				objectStatus.Conditions = append(objectStatus.Conditions, *tc.condition)
			}
			if tc.startTime {
				pipelineStatusFields.StartTime = &metav1.Time{Time: time.Now()}
			}
			pr := v1beta1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pipelinerun",
					Namespace: "foo",
				},
				Spec: v1beta1.PipelineRunSpec{
					PipelineRef: &v1beta1.PipelineRef{
						Name: "test-pipeline",
					},
				},
				Status: v1beta1.PipelineRunStatus{
					Status:                  objectStatus,
					PipelineRunStatusFields: pipelineStatusFields,
				},
			}
			// Set annotations, if any
			if tc.annotations != nil {
				if pr.ObjectMeta.Annotations == nil {
					pr.ObjectMeta.Annotations = map[string]string{}
				}
				for k, v := range tc.annotations {
					pr.ObjectMeta.Annotations[k] = v
				}
			}
			// Set results, if any
			if tc.results != nil {
				for k, v := range tc.results {
					trr := v1beta1.PipelineRunResult{Name: k, Value: v}
					pr.Status.PipelineResults = append(pr.Status.PipelineResults, trr)
				}
			}
			prs := []*v1beta1.PipelineRun{&pr}

			d := test.Data{
				PipelineRuns: prs,
				Pipelines:    ps,
				Tasks:        ts,
				ConfigMaps:   cms,
			}
			prt := newPipelineRunTest(d, t)
			defer prt.Cancel()

			_, clients := prt.reconcileRun("foo", "test-pipelinerun", false)

			ceClient := clients.CloudEvents.(pipelinecloudevent.FakeClient)
			err := checkCloudEvents(t, &ceClient, "reconcile-cloud-events", tc.wantCloudEvents)
			if err != nil {
				t.Errorf(err.Error())
			}
		})
	}
}
