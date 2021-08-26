//go:build e2e
// +build e2e

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

package test

import (
	"context"
	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/experimental/celeval/pkg/apis/celeval"
	celevalv1alpha1 "github.com/tektoncd/experimental/celeval/pkg/apis/celeval/v1alpha1"
	"github.com/tektoncd/experimental/celeval/pkg/client/clientset/versioned"
	resourceversioned "github.com/tektoncd/experimental/celeval/pkg/client/clientset/versioned/typed/celeval/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	tektontest "github.com/tektoncd/pipeline/test"
	"github.com/tektoncd/pipeline/test/diff"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
	knativetest "knative.dev/pkg/test"
	"regexp"
	"testing"
	"time"
)

var (
	runTimeout = 10 * time.Minute
)

// Expected events can be required or optional
type ev struct {
	message  string
	required bool
}

func TestCELEvalRun(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name            string
		celEval         *celevalv1alpha1.CELEval
		run             *v1alpha1.Run
		expectedStatus  corev1.ConditionStatus
		expectedReason  string
		expectedResults []v1alpha1.RunResult
		expectedMessage string
		expectedEvents  []ev
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
		expectedEvents: []ev{
			{"", true},
			{"CEL expressions were evaluated successfully", true},
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
		expectedEvents: []ev{
			{"", true},
			{"CEL expressions were evaluated successfully", true},
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
		expectedEvents: []ev{
			{"", true},
			{"CEL expressions were evaluated successfully", true},
		},
	}, {
		name: "multiple expressions and multiple variable successful",
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
		expectedEvents: []ev{
			{"", true},
			{"CEL expressions were evaluated successfully", true},
		},
	}}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			tc := tc // Copy current tc to local variable due to test parallelization
			t.Parallel()
			ctx := context.Background()
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()
			c, namespace := setup(ctx, t)
			CELEvalClient := getCELEvalClient(t, namespace)

			knativetest.CleanupOnInterrupt(func() { tearDown(ctx, t, c, namespace) }, t.Logf)
			defer tearDown(ctx, t, c, namespace)

			if tc.celEval != nil {
				celEval := tc.celEval.DeepCopy()
				celEval.Namespace = namespace
				if _, err := CELEvalClient.Create(ctx, celEval, metav1.CreateOptions{}); err != nil {
					t.Fatalf("Failed to create CELEval `%s`: %s", tc.celEval.Name, err)
				}
			}

			run := tc.run.DeepCopy()
			run.Namespace = namespace
			run, err := c.RunClient.Create(ctx, run, metav1.CreateOptions{})
			if err != nil {
				t.Fatalf("Failed to create Run `%s`: %s", run.Name, err)
			}

			t.Logf("Waiting for Run %s in namespace %s to complete", run.Name, run.Namespace)
			var inState tektontest.ConditionAccessorFn
			var desc string
			if tc.expectedStatus == corev1.ConditionTrue {
				inState = tektontest.Succeed(run.Name)
				desc = "RunSuccess"
			} else {
				inState = tektontest.FailedWithReason(tc.expectedReason, run.Name)
				desc = "RunFailed"
			}
			if err := WaitForRunState(ctx, c, run.Name, runTimeout, inState, desc); err != nil {
				t.Fatalf("Error waiting for Run %s/%s to finish: %s", run.Namespace, run.Name, err)
			}

			t.Logf("Run %s in namespace %s completed - fetching it", run.Name, run.Namespace)
			run, err = c.RunClient.Get(ctx, run.Name, metav1.GetOptions{})
			if err != nil {
				t.Fatalf("Couldn't get expected Run %s/%s: %s", run.Namespace, run.Name, err)
			}

			t.Logf("Checking CELEval status in the Run status")
			status := &celevalv1alpha1.CELEvalStatus{}
			if err := run.Status.DecodeExtraFields(status); err != nil {
				t.Errorf("DecodeExtraFields error: %v", err.Error())
			}

			t.Logf("Checking the status, reason and message in the Run's ConditionSucceeded")
			checkRunCondition(t, run, tc.expectedStatus, tc.expectedReason, tc.expectedMessage)

			t.Logf("Verifying the expected results are produced")
			if d := cmp.Diff(tc.expectedResults, run.Status.Results); d != "" {
				t.Errorf("Status Results: %s", diff.PrintWantGot(d))
			}

			t.Logf("Checking events that were created from Run")
			checkEvents(ctx, t, c, run, namespace, tc.expectedEvents)
		})
	}
}

func getCELEvalClient(t *testing.T, namespace string) resourceversioned.CELEvalInterface {
	configPath := knativetest.Flags.Kubeconfig
	clusterName := knativetest.Flags.Cluster
	cfg, err := knativetest.BuildClientConfig(configPath, clusterName)
	if err != nil {
		t.Fatalf("failed to create configuration obj from %s for cluster %s: %s", configPath, clusterName, err)
	}
	cs, err := versioned.NewForConfig(cfg)
	if err != nil {
		t.Fatalf("failed to create celeval clientset from config file at %s: %s", configPath, err)
	}
	return cs.CustomV1alpha1().CELEvals(namespace)
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

func checkEvents(ctx context.Context, t *testing.T, c *clients, run *v1alpha1.Run, namespace string, expectedEvents []ev) {
	matchKinds := map[string][]string{"Run": {run.Name}}
	events, err := collectMatchingEvents(ctx, c.KubeClient, namespace, matchKinds)
	if err != nil {
		t.Fatalf("Failed to collect matching events: %q", err)
	}
	// Log the received events.
	receivedEvents := make([]string, 0, len(events))
	for _, receivedEvent := range events {
		receivedEvents = append(receivedEvents, receivedEvent.Message)
	}
	t.Logf("Received events: %q", receivedEvents)
	// In the concurrency scenarios some events may or may not happen based on timing.
	e := 0
	for _, expectedEvent := range expectedEvents {
		if e >= len(events) {
			if !expectedEvent.required {
				continue
			}
			t.Errorf("Did not get expected event %q", expectedEvent.message)
			continue
		}
		if matched, _ := regexp.MatchString(expectedEvent.message, events[e].Message); !matched {
			if !expectedEvent.required {
				continue
			}
			t.Errorf("Expected event %q but got %q", expectedEvent.message, events[e].Message)
		}
		e++
	}
}

// collectMatchingEvents collects a list of events under 5 seconds that match certain objects by kind and name.
// This is copied from pipelinerun_test and modified to drop the reason parameter.
func collectMatchingEvents(ctx context.Context, kubeClient *knativetest.KubeClient, namespace string, kinds map[string][]string) ([]*corev1.Event, error) {
	var events []*corev1.Event

	watchEvents, err := kubeClient.CoreV1().Events(namespace).Watch(ctx, metav1.ListOptions{})
	// close watchEvents channel
	defer watchEvents.Stop()
	if err != nil {
		return events, err
	}

	// create timer to not wait for events longer than 5 seconds
	timer := time.NewTimer(5 * time.Second)

	for {
		select {
		case wevent := <-watchEvents.ResultChan():
			event := wevent.Object.(*corev1.Event)
			if val, ok := kinds[event.InvolvedObject.Kind]; ok {
				for _, expectedName := range val {
					if event.InvolvedObject.Name == expectedName {
						events = append(events, event)
					}
				}
			}
		case <-timer.C:
			return events, nil
		}
	}
}
