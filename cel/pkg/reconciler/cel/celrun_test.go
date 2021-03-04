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

package cel

import (
	"context"
	"fmt"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/experimental/cel/test"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	ttesting "github.com/tektoncd/pipeline/pkg/reconciler/testing"
	"github.com/tektoncd/pipeline/test/diff"

	"strings"
	"testing"
	"time"

	"github.com/tektoncd/pipeline/test/names"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/reconciler"
)

func TestReconcileCelRun(t *testing.T) {
	testcases := []struct {
		name            string
		run             *v1alpha1.Run
		expectedStatus  corev1.ConditionStatus
		expectedReason  string
		expectedResults []v1alpha1.RunResult
		expectedMessage string
		expectedEvents  []string
	}{{
		name: "one expression successful",
		run: &v1alpha1.Run{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cel-run",
				Namespace: "foo",
				Labels: map[string]string{
					"myTestLabel": "myTestLabelValue",
				},
				Annotations: map[string]string{
					"myTestAnnotation": "myTestAnnotationValue",
				},
			},
			Spec: v1alpha1.RunSpec{
				Params: []v1beta1.Param{{
					Name:  "expr1",
					Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "type(100)"},
				}},
				Ref: &v1alpha1.TaskRef{
					APIVersion: apiVersion,
					Kind:       kind,
					Name:       "a-celrun",
				},
			},
		},
		expectedStatus:  corev1.ConditionTrue,
		expectedReason:  ReasonEvaluationSuccess,
		expectedMessage: "CEL expressions were evaluated successfully",
		expectedResults: []v1alpha1.RunResult{{
			Name:  "expr1",
			Value: "int",
		}},
		expectedEvents: []string{"Normal RunReconciled Run reconciled: \"foo/cel-run\""},
	}, {
		name: "multiple expressions successful",
		run: &v1alpha1.Run{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cel-run",
				Namespace: "foo",
				Labels: map[string]string{
					"myTestLabel": "myTestLabelValue",
				},
				Annotations: map[string]string{
					"myTestAnnotation": "myTestAnnotationValue",
				},
			},
			Spec: v1alpha1.RunSpec{
				Params: []v1beta1.Param{{
					Name:  "expr1",
					Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "type(100)"},
				}, {
					Name:  "expr2",
					Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "3 == 3"},
				}},
				Ref: &v1alpha1.TaskRef{
					APIVersion: apiVersion,
					Kind:       kind,
					Name:       "a-celrun",
				},
			},
		},
		expectedStatus:  corev1.ConditionTrue,
		expectedReason:  ReasonEvaluationSuccess,
		expectedMessage: "CEL expressions were evaluated successfully",
		expectedResults: []v1alpha1.RunResult{{
			Name:  "expr1",
			Value: "int",
		}, {
			Name:  "expr2",
			Value: "true",
		}},
		expectedEvents: []string{"Normal RunReconciled Run reconciled: \"foo/cel-run\""},
	}, {
		name: "expressions with context variables successful",
		run: &v1alpha1.Run{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cel-run",
				Namespace: "foo",
				Labels: map[string]string{
					"myTestLabel": "myTestLabelValue",
				},
				Annotations: map[string]string{
					"myTestAnnotation": "myTestAnnotationValue",
				},
			},
			Spec: v1alpha1.RunSpec{
				Params: []v1beta1.Param{{
					Name:  "expr1",
					Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "type(100)"},
				}, {
					Name:  "expr2",
					Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "expr1 == 'int'"},
				}},
				Ref: &v1alpha1.TaskRef{
					APIVersion: apiVersion,
					Kind:       kind,
					Name:       "a-celrun",
				},
			},
		},
		expectedStatus:  corev1.ConditionTrue,
		expectedReason:  ReasonEvaluationSuccess,
		expectedMessage: "CEL expressions were evaluated successfully",
		expectedResults: []v1alpha1.RunResult{{
			Name:  "expr1",
			Value: "int",
		}, {
			Name:  "expr2",
			Value: "true",
		}},
		expectedEvents: []string{"Normal RunReconciled Run reconciled: \"foo/cel-run\""},
	}, {
		name: "CEL expressions with invalid type",
		run: &v1alpha1.Run{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cel-run",
				Namespace: "foo",
				Labels: map[string]string{
					"myTestLabel": "myTestLabelValue",
				},
				Annotations: map[string]string{
					"myTestAnnotation": "myTestAnnotationValue",
				},
			},
			Spec: v1alpha1.RunSpec{
				Params: []v1beta1.Param{{
					Name:  "expr1",
					Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeArray, ArrayVal: []string{"type(100)", "3 == 3"}},
				}},
				Ref: &v1alpha1.TaskRef{
					APIVersion: apiVersion,
					Kind:       kind,
					Name:       "a-celrun",
				},
			},
		},
		expectedStatus:  corev1.ConditionFalse,
		expectedReason:  ReasonFailedValidation,
		expectedMessage: "Run can't be run because it has an invalid spec - invalid value: CEL expression parameter expr1 must be a string: params[expr1].value",
	}, {
		name: "missing CEL expressions",
		run: &v1alpha1.Run{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cel-run",
				Namespace: "foo",
				Labels: map[string]string{
					"myTestLabel": "myTestLabelValue",
				},
				Annotations: map[string]string{
					"myTestAnnotation": "myTestAnnotationValue",
				},
			},
			Spec: v1alpha1.RunSpec{
				Ref: &v1alpha1.TaskRef{
					APIVersion: apiVersion,
					Kind:       kind,
					Name:       "a-celrun",
				},
			},
		},
		expectedStatus:  corev1.ConditionFalse,
		expectedReason:  ReasonFailedValidation,
		expectedMessage: "Run can't be run because it has an invalid spec - missing field(s): params",
	}}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			names.TestingSeed()

			d := test.Data{
				Runs: []*v1alpha1.Run{tc.run},
			}

			testAssets, _ := getCelController(t, d)
			c := testAssets.Controller
			clients := testAssets.Clients

			if err := c.Reconciler.Reconcile(ctx, getRunName(tc.run)); err != nil {
				t.Fatalf("Error reconciling: %s", err)
			}

			// Fetch the updated Run
			reconciledRun, err := clients.Pipeline.TektonV1alpha1().Runs(tc.run.Namespace).Get(ctx, tc.run.Name, metav1.GetOptions{})
			if err != nil {
				t.Fatalf("Error getting reconciled run from fake client: %s", err)
			}

			// Verify that the Run has the expected status and reason
			checkRunCondition(t, reconciledRun, tc.expectedStatus, tc.expectedReason, tc.expectedMessage)

			// Verify expected events were created
			if err := checkEvents(testAssets.Recorder, tc.name, tc.expectedEvents); err != nil {
				t.Errorf(err.Error())
			}

			// Verify the expected Results were produced
			if d := cmp.Diff(tc.expectedResults, reconciledRun.Status.Results); d != "" {
				t.Errorf("Status Results: %s", diff.PrintWantGot(d))
			}

		})
	}
}

func getCelController(t *testing.T, d test.Data) (test.Assets, func()) {
	ctx, _ := ttesting.SetupFakeContext(t)
	ctx, cancel := context.WithCancel(ctx)
	c, informers := test.SeedTestData(t, ctx, d)

	configMapWatcher := configmap.NewStaticWatcher()
	//configMapWatcher := configmap.NewInformedWatcher(c.Kube, system.GetNamespace())
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

func getRunName(run *v1alpha1.Run) string {
	return strings.Join([]string{run.Namespace, run.Name}, "/")
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
				return fmt.Errorf(`Received extra event "%s" for test "%s"`, event, testName)
			}
			wantEvent := wantEvents[ii]
			if !(strings.HasPrefix(event, wantEvent)) {
				return fmt.Errorf(`Expected event "%s" but got "%s" instead for test "%s"`, wantEvent, event, testName)
			}
		case <-timer.C:
			if len(foundEvents) > len(wantEvents) {
				return fmt.Errorf(`Received %d events but %d expected for test "%s". Found events: %#v`, len(foundEvents), len(wantEvents), testName, foundEvents)
			}
		}
	}
	return nil
}
