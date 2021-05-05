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

package pip

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/experimental/pipelines-in-pipelines/test"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	ttesting "github.com/tektoncd/pipeline/pkg/reconciler/testing"
	"github.com/tektoncd/pipeline/test/diff"
	"github.com/tektoncd/pipeline/test/names"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ktesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/record"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/reconciler"
)

func getPipController(t *testing.T, d test.Data) (test.Assets, func()) {
	ctx, _ := ttesting.SetupFakeContext(t)
	ctx, cancel := context.WithCancel(ctx)
	c, informers := test.SeedTestData(t, ctx, d)

	configMapWatcher := configmap.NewStaticWatcher()
	ctl := NewController(ctx, configMapWatcher)

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
	}, cancel
}

func checkRunCondition(t *testing.T, run *v1alpha1.Run, expectedStatus corev1.ConditionStatus, expectedReason string, expectedMessage string) {
	condition := run.Status.GetCondition(apis.ConditionSucceeded)
	if condition == nil {
		t.Error("Condition missing in Run")
	} else {
		if condition.Status != expectedStatus {
			t.Errorf("Expected Run status to be %v but was %v", expectedStatus, condition)
		}
		if condition.Reason != expectedReason {
			t.Errorf("Expected reason to be %q but was %q", expectedReason, condition.Reason)
		}
		if condition.Message != expectedMessage {
			t.Errorf("Expected message to be %q but was %q", expectedMessage, condition.Message)
		}
	}
	if run.Status.StartTime == nil {
		t.Errorf("Expected Run start time to be set but it wasn't")
	}
	if expectedStatus == corev1.ConditionUnknown {
		if run.Status.CompletionTime != nil {
			t.Errorf("Expected Run completion time to not be set but it was")
		}
	} else if run.Status.CompletionTime == nil {
		t.Errorf("Expected Run completion time to be set but it wasn't")
	}
}

func checkEvents(fr *record.FakeRecorder, testName string, wantEvents []string) error {
	// The fake recorder runs in a go routine, so the timeout is here to avoid waiting
	// on the channel forever if fewer than expected events are received.
	// We only hit the timeout in case of failure of the test, so the actual value
	// of the timeout is not so relevant. It's only used when tests are going to fail.
	timer := time.NewTimer(1 * time.Second)
	foundEvents := []string{}
	for ii := 0; ii < len(wantEvents)+1; ii++ {
		// We loop over all the events that we expect. Once they are all received
		// we exit the loop. If we never receive enough events, the timeout takes us
		// out of the loop.
		select {
		case event := <-fr.Events:
			foundEvents = append(foundEvents, event)
			if ii > len(wantEvents)-1 {
				return fmt.Errorf(`received extra event "%s" for test "%s"`, event, testName)
			}
			wantEvent := wantEvents[ii]
			if !(strings.HasPrefix(event, wantEvent)) {
				return fmt.Errorf(`expected event "%s" but got "%s" instead for test "%s"`, wantEvent, event, testName)
			}
		case <-timer.C:
			if len(foundEvents) > len(wantEvents) {
				return fmt.Errorf(`received %d events but %d expected for test "%s". Found events: %#v`, len(foundEvents), len(wantEvents), testName, foundEvents)
			}
		}
	}
	return nil
}

func getRunName(run *v1alpha1.Run) string {
	return strings.Join([]string{run.Namespace, run.Name}, "/")
}

func getCreatedPipelineRun(clients test.Clients) *v1beta1.PipelineRun {
	for _, a := range clients.Pipeline.Actions() {
		if a.GetVerb() == "create" {
			obj := a.(ktesting.CreateAction).GetObject()
			if pr, ok := obj.(*v1beta1.PipelineRun); ok {
				return pr
			}
		}
	}
	return nil
}

func running(pr *v1beta1.PipelineRun) *v1beta1.PipelineRun {
	prWithStatus := pr.DeepCopy()
	prWithStatus.Status.SetCondition(&apis.Condition{
		Type:   apis.ConditionSucceeded,
		Status: corev1.ConditionUnknown,
		Reason: v1beta1.PipelineRunReasonRunning.String(),
	})
	return prWithStatus
}

func successful(pr *v1beta1.PipelineRun) *v1beta1.PipelineRun {
	prWithStatus := pr.DeepCopy()
	prWithStatus.Status.SetCondition(&apis.Condition{
		Type:   apis.ConditionSucceeded,
		Status: corev1.ConditionTrue,
		Reason: v1beta1.PipelineRunReasonSuccessful.String(),
	})
	return prWithStatus
}

func failed(pr *v1beta1.PipelineRun) *v1beta1.PipelineRun {
	prWithStatus := pr.DeepCopy()
	prWithStatus.Status.SetCondition(&apis.Condition{
		Type:   apis.ConditionSucceeded,
		Status: corev1.ConditionFalse,
		Reason: v1beta1.PipelineRunReasonFailed.String(),
	})
	return prWithStatus
}

func withResults(pr *v1beta1.PipelineRun, name string, value string) *v1beta1.PipelineRun {
	prWithStatus := pr.DeepCopy()
	prWithStatus.Status.PipelineResults = append(prWithStatus.Status.PipelineResults, v1beta1.PipelineRunResult{
		Name:  name,
		Value: value,
	})
	return prWithStatus
}

var p = &v1beta1.Pipeline{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "pipeline",
		Namespace: "foo",
	},
	Spec: v1beta1.PipelineSpec{
		Tasks: []v1beta1.PipelineTask{{
			Name: "task",
			TaskSpec: &v1beta1.EmbeddedTask{TaskSpec: v1beta1.TaskSpec{
				Steps: []v1beta1.Step{{Container: corev1.Container{
					Image:   "ubuntu",
					Command: []string{"/bin/bash"},
					Args:    []string{"-c", "echo hello world"},
				}}},
			}},
		}},
	},
}

var blockOwnerDeletion = true
var isController = true
var pr = &v1beta1.PipelineRun{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "run-with-pipeline",
		Namespace: "foo",
		Labels: map[string]string{
			"tekton.dev/pipeline": "pipeline",
			"tekton.dev/run":      "run-with-pipeline",
		},
		OwnerReferences: []v1.OwnerReference{
			{
				APIVersion:         "tekton.dev/v1alpha1",
				Kind:               "Run",
				Name:               "run-with-pipeline",
				Controller:         &isController,
				BlockOwnerDeletion: &blockOwnerDeletion,
			},
		},
		Annotations: map[string]string{},
	},
	Spec: v1beta1.PipelineRunSpec{
		PipelineRef: &v1beta1.PipelineRef{
			Name:       "pipeline",
			APIVersion: "tekton.dev",
		},
		ServiceAccountName: "default",
	},
}

var runWithPipeline = &v1alpha1.Run{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "run-with-pipeline",
		Namespace: "foo",
	},
	Spec: v1alpha1.RunSpec{
		Ref: &v1alpha1.TaskRef{
			APIVersion: "tekton.dev/v1beta1",
			Kind:       "Pipeline",
			Name:       "pipeline",
		},
	},
}

var runWithoutPipelineName = &v1alpha1.Run{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "run-with-missing-pipeline",
		Namespace: "foo",
	},
	Spec: v1alpha1.RunSpec{
		Ref: &v1alpha1.TaskRef{
			APIVersion: "tekton.dev/v1beta1",
			Kind:       "Pipeline",
		},
	},
}

var runWithoutKind = &v1alpha1.Run{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "run-with-missing-pipeline",
		Namespace: "foo",
	},
	Spec: v1alpha1.RunSpec{
		Ref: &v1alpha1.TaskRef{
			APIVersion: "tekton.dev/v1beta1",
			Name:       "pipeline",
		},
	},
}

var runWithoutAPIVersion = &v1alpha1.Run{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "run-with-missing-pipeline",
		Namespace: "foo",
	},
	Spec: v1alpha1.RunSpec{
		Ref: &v1alpha1.TaskRef{
			Kind: "Pipeline",
			Name: "pipeline",
		},
	},
}

func TestReconcilePipRun(t *testing.T) {
	testcases := []struct {
		name                string
		pipeline            *v1beta1.Pipeline
		run                 *v1alpha1.Run
		pipelineRun         *v1beta1.PipelineRun
		expectedStatus      corev1.ConditionStatus
		expectedReason      v1beta1.PipelineRunReason
		expectedResults     []v1alpha1.RunResult
		expectedMessage     string
		expectedEvents      []string
		expectedPipelineRun *v1beta1.PipelineRun
	}{{
		name:                "Reconcile a new run that references a pipeline",
		pipeline:            p,
		run:                 runWithPipeline,
		expectedPipelineRun: pr,
		expectedStatus:      corev1.ConditionUnknown,
		expectedReason:      v1beta1.PipelineRunReasonStarted,
		expectedEvents: []string{
			"Normal Started ",
		},
	}, {
		name:           "Reconcile a run with a running PipelineRun",
		pipeline:       p,
		run:            runWithPipeline,
		pipelineRun:    running(pr),
		expectedStatus: corev1.ConditionUnknown,
		expectedReason: v1beta1.PipelineRunReasonRunning,
		expectedEvents: []string{
			"Normal Started ",
			"Normal Running ",
		},
	}, {
		name:           "Reconcile a run with a failed PipelineRun",
		pipeline:       p,
		run:            runWithPipeline,
		pipelineRun:    failed(pr),
		expectedStatus: corev1.ConditionFalse,
		expectedReason: v1beta1.PipelineRunReasonFailed,
		expectedEvents: []string{
			"Normal Started ",
			"Warning Failed ",
		},
	}, {
		name:           "Reconcile a run with a successful PipelineRun",
		pipeline:       p,
		run:            runWithPipeline,
		pipelineRun:    successful(pr),
		expectedStatus: corev1.ConditionTrue,
		expectedReason: v1beta1.PipelineRunReasonSuccessful,
		expectedEvents: []string{
			"Normal Started ",
			"Normal Succeeded ",
		},
	}, {
		name:            "Reconcile a new run that does not have a pipeline name",
		pipeline:        p,
		run:             runWithoutPipelineName,
		expectedStatus:  corev1.ConditionFalse,
		expectedReason:  ReasonRunFailedValidation,
		expectedMessage: "Run can't be run because it has an invalid spec - missing field(s): name",
		expectedEvents: []string{
			"Normal Started ",
			"Warning Failed Run can't be run because it has an invalid spec - missing field(s): name",
			"Warning InternalError 1 error occurred",
		},
	}, {
		name:           "Reconcile a run with a successful PipelineRun containing PipelineRunResults",
		pipeline:       p,
		run:            runWithPipeline,
		pipelineRun:    successful(withResults(pr, "foo", "bar")),
		expectedStatus: corev1.ConditionTrue,
		expectedReason: v1beta1.PipelineRunReasonSuccessful,
		expectedResults: []v1alpha1.RunResult{{
			Name:  "foo",
			Value: "bar",
		}},
		expectedEvents: []string{
			"Normal Started ",
			"Normal Succeeded ",
		},
	}}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			names.TestingSeed()

			optionalPipelineRuns := []*v1beta1.PipelineRun{tc.pipelineRun}
			if tc.pipelineRun == nil {
				optionalPipelineRuns = nil
			}

			d := test.Data{
				Runs:         []*v1alpha1.Run{tc.run},
				Pipelines:    []*v1beta1.Pipeline{tc.pipeline},
				PipelineRuns: optionalPipelineRuns,
			}

			testAssets, _ := getPipController(t, d)
			c := testAssets.Controller
			clients := testAssets.Clients

			c.Reconciler.Reconcile(ctx, getRunName(tc.run))

			run, err := clients.Pipeline.TektonV1alpha1().Runs(tc.run.Namespace).Get(ctx, tc.run.Name, metav1.GetOptions{})
			if err != nil {
				t.Fatalf("Error getting reconciled run from fake client: %s", err)
			}

			createdPipelineRun := getCreatedPipelineRun(clients)
			if tc.expectedPipelineRun != nil {
				if createdPipelineRun == nil {
					t.Errorf("A PipelineRun should have been created but was not")
				} else {
					if d := cmp.Diff(tc.expectedPipelineRun, createdPipelineRun); d != "" {
						t.Errorf("Expected PipelineRun was not created. Diff %s", diff.PrintWantGot(d))
					}
				}
			}

			checkRunCondition(t, run, tc.expectedStatus, tc.expectedReason.String(), tc.expectedMessage)

			if err := checkEvents(testAssets.Recorder, tc.name, tc.expectedEvents); err != nil {
				t.Errorf(err.Error())
			}

			if d := cmp.Diff(tc.expectedResults, run.Status.Results); d != "" {
				t.Errorf("Status Results: %s", diff.PrintWantGot(d))
			}

		})
	}
}
