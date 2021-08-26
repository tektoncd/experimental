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

package celevalrun

import (
	"context"
	"fmt"
	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/experimental/celeval/pkg/apis/celeval"
	celevalv1alpha1 "github.com/tektoncd/experimental/celeval/pkg/apis/celeval/v1alpha1"
	fakeclient "github.com/tektoncd/experimental/celeval/pkg/client/injection/client/fake"
	fakecelinformer "github.com/tektoncd/experimental/celeval/pkg/client/injection/informers/celeval/v1alpha1/celeval/fake"
	"github.com/tektoncd/experimental/celeval/test"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	ttesting "github.com/tektoncd/pipeline/pkg/reconciler/testing"
	"github.com/tektoncd/pipeline/test/diff"
	cminformer "knative.dev/pkg/configmap/informer"

	"github.com/tektoncd/pipeline/test/names"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/reconciler"
	"knative.dev/pkg/system"
	_ "knative.dev/pkg/system/testing"
	"strings"
	"testing"
	"time"
)

var (
	namespace = ""
)

func TestReconcileCELEvalRun(t *testing.T) {
	testcases := []struct {
		name            string
		celEval         *celevalv1alpha1.CELEval
		run             *v1alpha1.Run
		expectedStatus  corev1.ConditionStatus
		expectedReason  string
		expectedResults []v1alpha1.RunResult
		expectedMessage string
		expectedEvents  []string
	}{{
		name: "one expression successful",
		celEval: &celevalv1alpha1.CELEval{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "a-celeval",
				Namespace:   "foo",
				Labels:      map[string]string{"myCELEvalLabel": "myCELEvalLabelValue"},
				Annotations: map[string]string{"myCELEvalAnnotation": "myCELEvalAnnotationValue"},
			},
			Spec: celevalv1alpha1.CELEvalSpec{
				Expressions: []*v1beta1.Param{{
					Name:  "expr1",
					Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "type(100)"},
				}},
			},
		},
		run: &v1alpha1.Run{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "a-celeval-run",
				Namespace:   "foo",
				Labels:      map[string]string{"myTestLabel": "myTestLabelValue"},
				Annotations: map[string]string{"myTestAnnotation": "myTestAnnotationValue"},
			},
			Spec: v1alpha1.RunSpec{
				Ref: &v1alpha1.TaskRef{
					APIVersion: celevalv1alpha1.SchemeGroupVersion.String(),
					Kind:       celeval.ControllerName,
					Name:       "a-celeval",
				},
			},
		},
		expectedStatus:  corev1.ConditionTrue,
		expectedReason:  celevalv1alpha1.CELEvalRunReasonEvaluationSuccess.String(),
		expectedMessage: "CEL expressions were evaluated successfully",
		expectedResults: []v1alpha1.RunResult{{
			Name:  "expr1",
			Value: "int",
		}},
		expectedEvents: []string{
			"Normal Started",
			"Normal Succeeded CEL expressions were evaluated successfully",
		},
	}, {
		name: "multiple expressions successful",
		celEval: &celevalv1alpha1.CELEval{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "a-celeval",
				Namespace:   "foo",
				Labels:      map[string]string{"myCELEvalLabel": "myCELEvalLabelValue"},
				Annotations: map[string]string{"myCELEvalAnnotation": "myCELEvalAnnotationValue"},
			},
			Spec: celevalv1alpha1.CELEvalSpec{
				Expressions: []*v1beta1.Param{{
					Name:  "expr1",
					Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "type(100)"},
				}, {
					Name:  "expr2",
					Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "3 == 3"},
				}},
			},
		},
		run: &v1alpha1.Run{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "a-celeval-run",
				Namespace:   "foo",
				Labels:      map[string]string{"myTestLabel": "myTestLabelValue"},
				Annotations: map[string]string{"myTestAnnotation": "myTestAnnotationValue"},
			},
			Spec: v1alpha1.RunSpec{
				Ref: &v1alpha1.TaskRef{
					APIVersion: celevalv1alpha1.SchemeGroupVersion.String(),
					Kind:       celeval.ControllerName,
					Name:       "a-celeval",
				},
			},
		},
		expectedStatus:  corev1.ConditionTrue,
		expectedReason:  celevalv1alpha1.CELEvalRunReasonEvaluationSuccess.String(),
		expectedMessage: "CEL expressions were evaluated successfully",
		expectedResults: []v1alpha1.RunResult{{
			Name:  "expr1",
			Value: "int",
		}, {
			Name:  "expr2",
			Value: "true",
		}},
		expectedEvents: []string{
			"Normal Started",
			"Normal Succeeded CEL expressions were evaluated successfully",
		},
	}, {
		name: "one expression and one variable successful",
		celEval: &celevalv1alpha1.CELEval{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "a-celeval",
				Namespace:   "foo",
				Labels:      map[string]string{"myCELEvalLabel": "myCELEvalLabelValue"},
				Annotations: map[string]string{"myCELEvalAnnotation": "myCELEvalAnnotationValue"},
			},
			Spec: celevalv1alpha1.CELEvalSpec{
				Variables: []*v1beta1.Param{{
					Name:  "var1",
					Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "foo"},
				}},
				Expressions: []*v1beta1.Param{{
					Name:  "expr1",
					Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "var1 in ['foo', 'bar']"},
				}},
			},
		},
		run: &v1alpha1.Run{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "a-celeval-run",
				Namespace:   "foo",
				Labels:      map[string]string{"myTestLabel": "myTestLabelValue"},
				Annotations: map[string]string{"myTestAnnotation": "myTestAnnotationValue"},
			},
			Spec: v1alpha1.RunSpec{
				Ref: &v1alpha1.TaskRef{
					APIVersion: celevalv1alpha1.SchemeGroupVersion.String(),
					Kind:       celeval.ControllerName,
					Name:       "a-celeval",
				},
			},
		},
		expectedStatus:  corev1.ConditionTrue,
		expectedReason:  celevalv1alpha1.CELEvalRunReasonEvaluationSuccess.String(),
		expectedMessage: "CEL expressions were evaluated successfully",
		expectedResults: []v1alpha1.RunResult{{
			Name:  "expr1",
			Value: "true",
		}},
		expectedEvents: []string{
			"Normal Started",
			"Normal Succeeded CEL expressions were evaluated successfully",
		},
	}, {
		name: "multiple expressions and multiple variable successful",
		celEval: &celevalv1alpha1.CELEval{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "a-celeval",
				Namespace:   "foo",
				Labels:      map[string]string{"myCELEvalLabel": "myCELEvalLabelValue"},
				Annotations: map[string]string{"myCELAnnotation": "myCELEvalAnnotationValue"},
			},
			Spec: celevalv1alpha1.CELEvalSpec{
				Variables: []*v1beta1.Param{{
					Name:  "var1",
					Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "foo"},
				}, {
					Name:  "var2",
					Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "bar"},
				}},
				Expressions: []*v1beta1.Param{{
					Name:  "expr1",
					Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "'foo' in [var1, var2]"},
				}},
			},
		},
		run: &v1alpha1.Run{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "a-celeval-run",
				Namespace:   "foo",
				Labels:      map[string]string{"myTestLabel": "myTestLabelValue"},
				Annotations: map[string]string{"myTestAnnotation": "myTestAnnotationValue"},
			},
			Spec: v1alpha1.RunSpec{
				Ref: &v1alpha1.TaskRef{
					APIVersion: celevalv1alpha1.SchemeGroupVersion.String(),
					Kind:       celeval.ControllerName,
					Name:       "a-celeval",
				},
			},
		},
		expectedStatus:  corev1.ConditionTrue,
		expectedReason:  celevalv1alpha1.CELEvalRunReasonEvaluationSuccess.String(),
		expectedMessage: "CEL expressions were evaluated successfully",
		expectedResults: []v1alpha1.RunResult{{
			Name:  "expr1",
			Value: "true",
		}},
		expectedEvents: []string{
			"Normal Started",
			"Normal Succeeded CEL expressions were evaluated successfully",
		},
	}, {
		name: "CEL expressions with invalid type",
		celEval: &celevalv1alpha1.CELEval{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "a-celeval",
				Namespace:   "foo",
				Labels:      map[string]string{"myCELEvalLabel": "myCELEvalLabelValue"},
				Annotations: map[string]string{"myCELEvalAnnotation": "myCELEvalAnnotationValue"},
			},
			Spec: celevalv1alpha1.CELEvalSpec{
				Expressions: []*v1beta1.Param{{
					Name:  "expr1",
					Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeArray, ArrayVal: []string{"type(100)", "3 == 3"}},
				}},
			},
		},
		run: &v1alpha1.Run{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "a-celeval-run",
				Namespace:   "foo",
				Labels:      map[string]string{"myTestLabel": "myTestLabelValue"},
				Annotations: map[string]string{"myTestAnnotation": "myTestAnnotationValue"},
			},
			Spec: v1alpha1.RunSpec{
				Ref: &v1alpha1.TaskRef{
					APIVersion: celevalv1alpha1.SchemeGroupVersion.String(),
					Kind:       celeval.ControllerName,
					Name:       "a-celeval",
				},
			},
		},
		expectedStatus:  corev1.ConditionFalse,
		expectedReason:  celevalv1alpha1.CELEvalRunReasonFailedValidation.String(),
		expectedMessage: "Run can't be run because it has an invalid spec - invalid value: CEL expression expr1 must be a string: expressions[expr1].value",
		expectedEvents: []string{
			"Normal Started",
			"Warning Failed Run can't be run because it has an invalid spec - invalid value: CEL expression expr1 must be a string: expressions[expr1].value",
		},
	}, {
		name: "missing CEL expressions",
		celEval: &celevalv1alpha1.CELEval{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "a-celeval",
				Namespace:   "foo",
				Labels:      map[string]string{"myCELEvalLabel": "myCELEvalLabelValue"},
				Annotations: map[string]string{"myCELEvalAnnotation": "myCELEvalAnnotationValue"},
			},
		},
		run: &v1alpha1.Run{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "a-celeval-run",
				Namespace:   "foo",
				Labels:      map[string]string{"myTestLabel": "myTestLabelValue"},
				Annotations: map[string]string{"myTestAnnotation": "myTestAnnotationValue"},
			},
			Spec: v1alpha1.RunSpec{
				Ref: &v1alpha1.TaskRef{
					APIVersion: celevalv1alpha1.SchemeGroupVersion.String(),
					Kind:       celeval.ControllerName,
					Name:       "a-celeval",
				},
			},
		},
		expectedStatus:  corev1.ConditionFalse,
		expectedReason:  celevalv1alpha1.CELEvalRunReasonFailedValidation.String(),
		expectedMessage: "Run can't be run because it has an invalid spec - missing field(s): expressions",
		expectedEvents: []string{
			"Normal Started",
			"Warning Failed Run can't be run because it has an invalid spec - missing field(s): expressions",
		},
	}}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			names.TestingSeed()

			d := test.Data{
				Runs: []*v1alpha1.Run{tc.run},
			}

			testAssets, _ := getCELEvalController(t, d, []*celevalv1alpha1.CELEval{tc.celEval})
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

func getCELEvalController(t *testing.T, d test.Data, celevals []*celevalv1alpha1.CELEval) (test.Assets, func()) {
	ctx, _ := ttesting.SetupFakeContext(t)
	ctx, cancel := context.WithCancel(ctx)
	c, informers := test.SeedTestData(t, ctx, d)

	client := fakeclient.Get(ctx)
	client.PrependReactor("*", "celevals", test.AddToInformer(t, fakecelinformer.Get(ctx).Informer().GetIndexer()))
	for _, celeval := range celevals {
		celeval := celeval.DeepCopy() // Avoid assumptions that the informer's copy is modified.
		if _, err := client.CustomV1alpha1().CELEvals(celeval.Namespace).Create(ctx, celeval, metav1.CreateOptions{}); err != nil {
			t.Fatal(err)
		}
	}

	configMapWatcher := cminformer.NewInformedWatcher(c.Kube, system.Namespace())
	ctl := NewController(namespace)(ctx, configMapWatcher)

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
