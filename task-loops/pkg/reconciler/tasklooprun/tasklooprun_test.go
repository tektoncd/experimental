/*
Copyright 2020 The Tekton Authors

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

package tasklooprun

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/tektoncd/experimental/task-loops/pkg/apis/taskloop"
	taskloopv1alpha1 "github.com/tektoncd/experimental/task-loops/pkg/apis/taskloop/v1alpha1"
	fakeclient "github.com/tektoncd/experimental/task-loops/pkg/client/injection/client/fake"
	faketaskloopinformer "github.com/tektoncd/experimental/task-loops/pkg/client/injection/informers/taskloop/v1alpha1/taskloop/fake"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	ttesting "github.com/tektoncd/pipeline/pkg/reconciler/testing"
	"github.com/tektoncd/pipeline/pkg/system"
	"github.com/tektoncd/pipeline/test"
	"github.com/tektoncd/pipeline/test/diff"
	"github.com/tektoncd/pipeline/test/names"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ktesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/record"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/reconciler"
)

var (
	namespace = ""
	trueB     = true
)

func getRunName(run *v1alpha1.Run) string {
	return strings.Join([]string{run.Namespace, run.Name}, "/")
}

func loopRunning(run *v1alpha1.Run) *v1alpha1.Run {
	runWithStatus := run.DeepCopy()
	runWithStatus.Status.InitializeConditions()
	runWithStatus.Status.MarkRunning(taskloopv1alpha1.TaskLoopRunReasonRunning.String(), "")
	return runWithStatus
}

func requestCancel(run *v1alpha1.Run) *v1alpha1.Run {
	runWithCancelStatus := run.DeepCopy()
	runWithCancelStatus.Spec.Status = v1alpha1.RunSpecStatusCancelled
	return runWithCancelStatus
}

func allowRetry(tl *taskloopv1alpha1.TaskLoop) *taskloopv1alpha1.TaskLoop {
	taskLoopWithRetries := tl.DeepCopy()
	taskLoopWithRetries.Spec.Retries = 1
	return taskLoopWithRetries
}

func running(tr *v1beta1.TaskRun) *v1beta1.TaskRun {
	trWithStatus := tr.DeepCopy()
	trWithStatus.Status.SetCondition(&apis.Condition{
		Type:   apis.ConditionSucceeded,
		Status: corev1.ConditionUnknown,
		Reason: v1beta1.TaskRunReasonRunning.String(),
	})
	return trWithStatus
}

func successful(tr *v1beta1.TaskRun) *v1beta1.TaskRun {
	trWithStatus := tr.DeepCopy()
	trWithStatus.Status.SetCondition(&apis.Condition{
		Type:    apis.ConditionSucceeded,
		Status:  corev1.ConditionTrue,
		Reason:  v1beta1.TaskRunReasonSuccessful.String(),
		Message: "All Steps have completed executing",
	})
	return trWithStatus
}

func failed(tr *v1beta1.TaskRun) *v1beta1.TaskRun {
	trWithStatus := tr.DeepCopy()
	trWithStatus.Status.SetCondition(&apis.Condition{
		Type:    apis.ConditionSucceeded,
		Status:  corev1.ConditionFalse,
		Reason:  v1beta1.TaskRunReasonFailed.String(),
		Message: "Something went wrong",
	})
	return trWithStatus
}

func retrying(tr *v1beta1.TaskRun) *v1beta1.TaskRun {
	trWithRetryStatus := tr.DeepCopy()
	trWithRetryStatus.Status.RetriesStatus = nil
	trWithRetryStatus.Status.RetriesStatus = append(tr.Status.RetriesStatus, trWithRetryStatus.Status)
	trWithRetryStatus.Status.SetCondition(&apis.Condition{
		Type:   apis.ConditionSucceeded,
		Status: corev1.ConditionUnknown,
	})
	return trWithRetryStatus
}

// getTaskLoopController returns an instance of the TaskLoop controller/reconciler that has been seeded with
// d, where d represents the state of the system (existing resources) needed for the test.
func getTaskLoopController(t *testing.T, d test.Data, taskloops []*taskloopv1alpha1.TaskLoop) (test.Assets, func()) {
	ctx, _ := ttesting.SetupFakeContext(t)
	ctx, cancel := context.WithCancel(ctx)
	c, informers := test.SeedTestData(t, ctx, d)

	client := fakeclient.Get(ctx)
	client.PrependReactor("*", "taskloops", test.AddToInformer(t, faketaskloopinformer.Get(ctx).Informer().GetIndexer()))
	for _, tl := range taskloops {
		tl := tl.DeepCopy() // Avoid assumptions that the informer's copy is modified.
		if _, err := client.CustomV1alpha1().TaskLoops(tl.Namespace).Create(tl); err != nil {
			t.Fatal(err)
		}
	}

	configMapWatcher := configmap.NewInformedWatcher(c.Kube, system.GetNamespace())
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

func getCreatedTaskrun(t *testing.T, clients test.Clients) *v1beta1.TaskRun {
	t.Log("actions", clients.Pipeline.Actions())
	for _, a := range clients.Pipeline.Actions() {
		if a.GetVerb() == "create" {
			obj := a.(ktesting.CreateAction).GetObject()
			if tr, ok := obj.(*v1beta1.TaskRun); ok {
				return tr
			}
		}
	}
	return nil
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

func checkRunCondition(t *testing.T, run *v1alpha1.Run, expectedStatus corev1.ConditionStatus, expectedReason taskloopv1alpha1.TaskLoopRunReason) {
	condition := run.Status.GetCondition(apis.ConditionSucceeded)
	if condition == nil {
		t.Error("Condition missing in Run")
	} else {
		if condition.Status != expectedStatus {
			t.Errorf("Expected Run status to be %v but was %v", expectedStatus, condition)
		}
		if condition.Reason != expectedReason.String() {
			t.Errorf("Expected reason to be %q but was %q", expectedReason.String(), condition.Reason)
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

func checkRunStatus(t *testing.T, run *v1alpha1.Run, expectedStatus map[string]taskloopv1alpha1.TaskLoopTaskRunStatus) {
	status := &taskloopv1alpha1.TaskLoopRunStatus{}
	if err := run.Status.DecodeExtraFields(status); err != nil {
		t.Errorf("DecodeExtraFields error: %v", err.Error())
	}
	t.Log("taskruns", status.TaskRuns)
	if len(status.TaskRuns) != len(expectedStatus) {
		t.Errorf("Expected Run status to include %d TaskRuns but found %d: %v", len(expectedStatus), len(status.TaskRuns), status.TaskRuns)
		return
	}
	for expectedTaskRunName, expectedTaskRunStatus := range expectedStatus {
		actualTaskRunStatus, exists := status.TaskRuns[expectedTaskRunName]
		if !exists {
			t.Errorf("Expected Run status to include TaskRun status for TaskRun %s", expectedTaskRunName)
			continue
		}
		if actualTaskRunStatus.Iteration != expectedTaskRunStatus.Iteration {
			t.Errorf("Run status for TaskRun %s has iteration number %d instead of %d",
				expectedTaskRunName, actualTaskRunStatus.Iteration, expectedTaskRunStatus.Iteration)
		}
		if d := cmp.Diff(expectedTaskRunStatus.Status, actualTaskRunStatus.Status, cmpopts.IgnoreTypes(apis.Condition{}.LastTransitionTime.Inner.Time)); d != "" {
			t.Errorf("Run status for TaskRun %s is incorrect. Diff %s", expectedTaskRunName, diff.PrintWantGot(d))
		}
	}
}

var aTask = &v1beta1.Task{
	ObjectMeta: metav1.ObjectMeta{Name: "a-task", Namespace: "foo"},
	Spec: v1beta1.TaskSpec{
		Params: []v1beta1.ParamSpec{{
			Name: "current-item",
			Type: v1beta1.ParamTypeString,
		}, {
			Name: "additional-parameter",
			Type: v1beta1.ParamTypeString,
		}},
		Steps: []v1beta1.Step{{
			Container: corev1.Container{Name: "foo", Image: "bar"},
		}},
	},
}

var aTaskLoop = &taskloopv1alpha1.TaskLoop{
	ObjectMeta: metav1.ObjectMeta{Name: "a-taskloop", Namespace: "foo"},
	Spec: taskloopv1alpha1.TaskLoopSpec{
		TaskRef:      &v1beta1.TaskRef{Name: "a-task"},
		IterateParam: "current-item",
	},
}

var aTaskLoopWithInlineTask = &taskloopv1alpha1.TaskLoop{
	ObjectMeta: metav1.ObjectMeta{Name: "a-taskloop-with-inline-task", Namespace: "foo"},
	Spec: taskloopv1alpha1.TaskLoopSpec{
		TaskSpec: &v1beta1.TaskSpec{
			Params: []v1beta1.ParamSpec{{
				Name: "current-item",
				Type: v1beta1.ParamTypeString,
			}, {
				Name: "additional-parameter",
				Type: v1beta1.ParamTypeString,
			}},
			Steps: []v1beta1.Step{{
				Container: corev1.Container{Name: "foo", Image: "bar"},
			}},
		},
		IterateParam: "current-item",
		Timeout:      &metav1.Duration{Duration: 5 * time.Minute},
	},
}

var runTaskLoop = &v1alpha1.Run{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "run-taskloop",
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
			Name:  "current-item",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeArray, ArrayVal: []string{"item1", "item2"}},
		}, {
			Name:  "additional-parameter",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "stuff"},
		}},
		Ref: &v1alpha1.TaskRef{
			APIVersion: taskloopv1alpha1.SchemeGroupVersion.String(),
			Kind:       taskloop.TaskLoopControllerName,
			Name:       "a-taskloop",
		},
	},
}

var runTaskLoopWithInlineTask = &v1alpha1.Run{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "run-taskloop-with-inline-task",
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
			Name:  "current-item",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeArray, ArrayVal: []string{"item1", "item2"}},
		}, {
			Name:  "additional-parameter",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "stuff"},
		}},
		Ref: &v1alpha1.TaskRef{
			APIVersion: taskloopv1alpha1.SchemeGroupVersion.String(),
			Kind:       taskloop.TaskLoopControllerName,
			Name:       "a-taskloop-with-inline-task",
		},
	},
}

var runWithMissingTaskLoopName = &v1alpha1.Run{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "bad-run-taskloop-missing",
		Namespace: "foo",
	},
	Spec: v1alpha1.RunSpec{
		Ref: &v1alpha1.TaskRef{
			APIVersion: taskloopv1alpha1.SchemeGroupVersion.String(),
			Kind:       taskloop.TaskLoopControllerName,
			// missing Name
		},
	},
}

var runWithNonexistentTaskLoop = &v1alpha1.Run{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "bad-run-taskloop-not-found",
		Namespace: "foo",
	},
	Spec: v1alpha1.RunSpec{
		Ref: &v1alpha1.TaskRef{
			APIVersion: taskloopv1alpha1.SchemeGroupVersion.String(),
			Kind:       taskloop.TaskLoopControllerName,
			Name:       "no-such-taskloop",
		},
	},
}

var runWithMissingIterateParam = &v1alpha1.Run{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "bad-run-missing-iterate-param",
		Namespace: "foo",
	},
	Spec: v1alpha1.RunSpec{
		// current-item, which is the iterate parameter, is missing from parameters
		Params: []v1beta1.Param{{
			Name:  "additional-parameter",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "stuff"},
		}},
		Ref: &v1alpha1.TaskRef{
			APIVersion: taskloopv1alpha1.SchemeGroupVersion.String(),
			Kind:       taskloop.TaskLoopControllerName,
			Name:       "a-taskloop",
		},
	},
}

var runWithIterateParamNotAnArray = &v1alpha1.Run{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "bad-run-iterate-param-not-an-array",
		Namespace: "foo",
	},
	Spec: v1alpha1.RunSpec{
		Params: []v1beta1.Param{{
			// Value of iteration parameter must be an array so this is an error.
			Name:  "current-item",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "item1"},
		}, {
			Name:  "additional-parameter",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "stuff"},
		}},
		Ref: &v1alpha1.TaskRef{
			APIVersion: taskloopv1alpha1.SchemeGroupVersion.String(),
			Kind:       taskloop.TaskLoopControllerName,
			Name:       "a-taskloop",
		},
	},
}

var expectedTaskRunIteration1 = &v1beta1.TaskRun{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "run-taskloop-00001-9l9zj",
		Namespace: "foo",
		OwnerReferences: []metav1.OwnerReference{{
			APIVersion:         "tekton.dev/v1alpha1",
			Kind:               "Run",
			Name:               "run-taskloop",
			Controller:         &trueB,
			BlockOwnerDeletion: &trueB,
		}},
		Labels: map[string]string{
			"custom.tekton.dev/taskLoop":          "a-taskloop",
			"tekton.dev/run":                      "run-taskloop",
			"custom.tekton.dev/taskLoopIteration": "1",
			"myTestLabel":                         "myTestLabelValue",
		},
		Annotations: map[string]string{
			"myTestAnnotation": "myTestAnnotationValue",
		},
	},
	Spec: v1beta1.TaskRunSpec{
		TaskRef: &v1beta1.TaskRef{Name: "a-task"},
		Params: []v1beta1.Param{{
			Name:  "current-item",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "item1"},
		}, {
			Name:  "additional-parameter",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "stuff"},
		}},
	},
}

// Note: The taskrun for the second iteration has the same random suffix as the first due to the resetting of the seed on each test.
var expectedTaskRunIteration2 = &v1beta1.TaskRun{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "run-taskloop-00002-9l9zj",
		Namespace: "foo",
		OwnerReferences: []metav1.OwnerReference{{
			APIVersion:         "tekton.dev/v1alpha1",
			Kind:               "Run",
			Name:               "run-taskloop",
			Controller:         &trueB,
			BlockOwnerDeletion: &trueB,
		}},
		Labels: map[string]string{
			"custom.tekton.dev/taskLoop":          "a-taskloop",
			"tekton.dev/run":                      "run-taskloop",
			"custom.tekton.dev/taskLoopIteration": "2",
			"myTestLabel":                         "myTestLabelValue",
		},
		Annotations: map[string]string{
			"myTestAnnotation": "myTestAnnotationValue",
		},
	},
	Spec: v1beta1.TaskRunSpec{
		TaskRef: &v1beta1.TaskRef{Name: "a-task"},
		Params: []v1beta1.Param{{
			Name:  "current-item",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "item2"},
		}, {
			Name:  "additional-parameter",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "stuff"},
		}},
	},
}

var expectedTaskRunWithInlineTaskIteration1 = &v1beta1.TaskRun{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "run-taskloop-with-inline-task-00001-9l9zj",
		Namespace: "foo",
		OwnerReferences: []metav1.OwnerReference{{
			APIVersion:         "tekton.dev/v1alpha1",
			Kind:               "Run",
			Name:               "run-taskloop-with-inline-task",
			Controller:         &trueB,
			BlockOwnerDeletion: &trueB,
		}},
		Labels: map[string]string{
			"custom.tekton.dev/taskLoop":          "a-taskloop-with-inline-task",
			"tekton.dev/run":                      "run-taskloop-with-inline-task",
			"custom.tekton.dev/taskLoopIteration": "1",
			"myTestLabel":                         "myTestLabelValue",
		},
		Annotations: map[string]string{
			"myTestAnnotation": "myTestAnnotationValue",
		},
	},
	Spec: v1beta1.TaskRunSpec{
		TaskSpec: &v1beta1.TaskSpec{
			Params: []v1beta1.ParamSpec{{
				Name: "current-item",
				Type: v1beta1.ParamTypeString,
			}, {
				Name: "additional-parameter",
				Type: v1beta1.ParamTypeString,
			}},
			Steps: []v1beta1.Step{{
				Container: corev1.Container{Name: "foo", Image: "bar"},
			}},
		},
		Params: []v1beta1.Param{{
			Name:  "current-item",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "item1"},
		}, {
			Name:  "additional-parameter",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "stuff"},
		}},
		Timeout: &metav1.Duration{Duration: 5 * time.Minute},
	},
}

func TestReconcileTaskLoopRun(t *testing.T) {

	testcases := []struct {
		name string
		// The following set of fields describe the resources on entry to reconcile.
		task     *v1beta1.Task
		taskloop *taskloopv1alpha1.TaskLoop
		run      *v1alpha1.Run
		taskruns []*v1beta1.TaskRun
		// The following set of fields describe the expected state after reconcile.
		expectedStatus   corev1.ConditionStatus
		expectedReason   taskloopv1alpha1.TaskLoopRunReason
		expectedTaskruns []*v1beta1.TaskRun
		expectedEvents   []string
	}{{
		name:             "Reconcile a new run with a taskloop that references a task",
		task:             aTask,
		taskloop:         aTaskLoop,
		run:              runTaskLoop,
		taskruns:         []*v1beta1.TaskRun{},
		expectedStatus:   corev1.ConditionUnknown,
		expectedReason:   taskloopv1alpha1.TaskLoopRunReasonRunning,
		expectedTaskruns: []*v1beta1.TaskRun{expectedTaskRunIteration1},
		expectedEvents:   []string{"Normal Started", "Normal Running Iterations completed: 0"},
	}, {
		name:             "Reconcile a new run with a taskloop that contains an inline task",
		taskloop:         aTaskLoopWithInlineTask,
		run:              runTaskLoopWithInlineTask,
		taskruns:         []*v1beta1.TaskRun{},
		expectedStatus:   corev1.ConditionUnknown,
		expectedReason:   taskloopv1alpha1.TaskLoopRunReasonRunning,
		expectedTaskruns: []*v1beta1.TaskRun{expectedTaskRunWithInlineTaskIteration1},
		expectedEvents:   []string{"Normal Started", "Normal Running Iterations completed: 0"},
	}, {
		name:             "Reconcile a run after the first TaskRun has succeeded.",
		task:             aTask,
		taskloop:         aTaskLoop,
		run:              loopRunning(runTaskLoop),
		taskruns:         []*v1beta1.TaskRun{successful(expectedTaskRunIteration1)},
		expectedStatus:   corev1.ConditionUnknown,
		expectedReason:   taskloopv1alpha1.TaskLoopRunReasonRunning,
		expectedTaskruns: []*v1beta1.TaskRun{successful(expectedTaskRunIteration1), expectedTaskRunIteration2},
		expectedEvents:   []string{"Normal Running Iterations completed: 1"},
	}, {
		name:             "Reconcile a run after all TaskRuns have succeeded",
		task:             aTask,
		taskloop:         aTaskLoop,
		run:              loopRunning(runTaskLoop),
		taskruns:         []*v1beta1.TaskRun{successful(expectedTaskRunIteration1), successful(expectedTaskRunIteration2)},
		expectedStatus:   corev1.ConditionTrue,
		expectedReason:   taskloopv1alpha1.TaskLoopRunReasonSucceeded,
		expectedTaskruns: []*v1beta1.TaskRun{successful(expectedTaskRunIteration1), successful(expectedTaskRunIteration2)},
		expectedEvents:   []string{"Normal Succeeded All TaskRuns completed successfully"},
	}, {
		name:             "Reconcile a run after the first TaskRun has failed",
		task:             aTask,
		taskloop:         aTaskLoop,
		run:              loopRunning(runTaskLoop),
		taskruns:         []*v1beta1.TaskRun{failed(expectedTaskRunIteration1)},
		expectedStatus:   corev1.ConditionFalse,
		expectedReason:   taskloopv1alpha1.TaskLoopRunReasonFailed,
		expectedTaskruns: []*v1beta1.TaskRun{failed(expectedTaskRunIteration1)},
		expectedEvents:   []string{"Warning Failed TaskRun " + expectedTaskRunIteration1.Name + " has failed"},
	}, {
		name:             "Reconcile a run after the first TaskRun has failed and retry is allowed",
		task:             aTask,
		taskloop:         allowRetry(aTaskLoop),
		run:              loopRunning(runTaskLoop),
		taskruns:         []*v1beta1.TaskRun{failed(expectedTaskRunIteration1)},
		expectedStatus:   corev1.ConditionUnknown,
		expectedReason:   taskloopv1alpha1.TaskLoopRunReasonRunning,
		expectedTaskruns: []*v1beta1.TaskRun{retrying(failed(expectedTaskRunIteration1))},
		expectedEvents:   []string{},
	}, {
		name:             "Reconcile a run after the first TaskRun has failed and retry failed as well",
		task:             aTask,
		taskloop:         allowRetry(aTaskLoop),
		run:              loopRunning(runTaskLoop),
		taskruns:         []*v1beta1.TaskRun{failed(retrying(failed(expectedTaskRunIteration1)))},
		expectedStatus:   corev1.ConditionFalse,
		expectedReason:   taskloopv1alpha1.TaskLoopRunReasonFailed,
		expectedTaskruns: []*v1beta1.TaskRun{failed(retrying(failed(expectedTaskRunIteration1)))},
		expectedEvents:   []string{"Warning Failed TaskRun " + expectedTaskRunIteration1.Name + " has failed"},
	}, {
		name:             "Reconcile a cancelled run while the first TaskRun is running",
		task:             aTask,
		taskloop:         aTaskLoop,
		run:              requestCancel(loopRunning(runTaskLoop)),
		taskruns:         []*v1beta1.TaskRun{running(expectedTaskRunIteration1)},
		expectedStatus:   corev1.ConditionUnknown,
		expectedReason:   taskloopv1alpha1.TaskLoopRunReasonRunning,
		expectedTaskruns: []*v1beta1.TaskRun{running(expectedTaskRunIteration1)},
		expectedEvents:   []string{"Normal Running Cancelling TaskRun " + expectedTaskRunIteration1.Name},
	}, {
		name:             "Reconcile a cancelled run after the first TaskRun has failed",
		task:             aTask,
		taskloop:         aTaskLoop,
		run:              requestCancel(loopRunning(runTaskLoop)),
		taskruns:         []*v1beta1.TaskRun{failed(expectedTaskRunIteration1)},
		expectedStatus:   corev1.ConditionFalse,
		expectedReason:   taskloopv1alpha1.TaskLoopRunReasonCancelled,
		expectedTaskruns: []*v1beta1.TaskRun{failed(expectedTaskRunIteration1)},
		expectedEvents:   []string{"Warning Failed Run " + runTaskLoop.Namespace + "/" + runTaskLoop.Name + " was cancelled"},
	}, {
		name:             "Reconcile a cancelled run after the first TaskRun has succeeded",
		task:             aTask,
		taskloop:         aTaskLoop,
		run:              requestCancel(loopRunning(runTaskLoop)),
		taskruns:         []*v1beta1.TaskRun{successful(expectedTaskRunIteration1)},
		expectedStatus:   corev1.ConditionFalse,
		expectedReason:   taskloopv1alpha1.TaskLoopRunReasonCancelled,
		expectedTaskruns: []*v1beta1.TaskRun{successful(expectedTaskRunIteration1)},
		expectedEvents:   []string{"Warning Failed Run " + runTaskLoop.Namespace + "/" + runTaskLoop.Name + " was cancelled"},
	}}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			names.TestingSeed()

			optionalTask := []*v1beta1.Task{tc.task}
			if tc.task == nil {
				optionalTask = nil
			}

			d := test.Data{
				Runs:     []*v1alpha1.Run{tc.run},
				Tasks:    optionalTask,
				TaskRuns: tc.taskruns,
			}

			testAssets, _ := getTaskLoopController(t, d, []*taskloopv1alpha1.TaskLoop{tc.taskloop})
			c := testAssets.Controller
			clients := testAssets.Clients

			if err := c.Reconciler.Reconcile(context.Background(), getRunName(tc.run)); err != nil {
				t.Fatalf("Error reconciling: %s", err)
			}

			// Fetch the updated Run
			reconciledRun, err := clients.Pipeline.TektonV1alpha1().Runs(tc.run.Namespace).Get(tc.run.Name, metav1.GetOptions{})
			if err != nil {
				t.Fatalf("Error getting reconciled run from fake client: %s", err)
			}

			// Verify that the Run has the expected status and reason.
			checkRunCondition(t, reconciledRun, tc.expectedStatus, tc.expectedReason)

			// Verify that a TaskRun was or was not created depending on the test.
			// If the number of expected TaskRuns is greater than the original number of TaskRuns
			// then the test expects a new TaskRun to be created.  The new TaskRun must be the
			// last one in the list of expected TaskRuns.
			createdTaskrun := getCreatedTaskrun(t, clients)
			if len(tc.expectedTaskruns) > len(tc.taskruns) {
				if createdTaskrun == nil {
					t.Errorf("A TaskRun should have been created but was not")
				} else {
					if d := cmp.Diff(tc.expectedTaskruns[len(tc.expectedTaskruns)-1], createdTaskrun); d != "" {
						t.Errorf("Expected TaskRun was not created. Diff %s", diff.PrintWantGot(d))
					}
				}
			} else {
				if createdTaskrun != nil {
					t.Errorf("A TaskRun was created which was not expected")
				}
			}

			// Verify Run status contains status for all TaskRuns.
			expectedTaskRuns := map[string]taskloopv1alpha1.TaskLoopTaskRunStatus{}
			for i, tr := range tc.expectedTaskruns {
				expectedTaskRuns[tr.Name] = taskloopv1alpha1.TaskLoopTaskRunStatus{Iteration: i + 1, Status: &tr.Status}
			}
			checkRunStatus(t, reconciledRun, expectedTaskRuns)

			// Verify expected events were created.
			if err := checkEvents(testAssets.Recorder, tc.name, tc.expectedEvents); err != nil {
				t.Errorf(err.Error())
			}
		})
	}
}

func TestReconcileTaskLoopRunFailures(t *testing.T) {
	testcases := []struct {
		name       string
		taskloop   *taskloopv1alpha1.TaskLoop
		run        *v1alpha1.Run
		reason     taskloopv1alpha1.TaskLoopRunReason
		wantEvents []string
	}{{
		name:   "missing TaskLoop name",
		run:    runWithMissingTaskLoopName,
		reason: taskloopv1alpha1.TaskLoopRunReasonCouldntGetTaskLoop,
		wantEvents: []string{
			"Normal Started ",
			"Warning Failed Missing spec.ref.name for Run",
		},
	}, {
		name:   "nonexistent TaskLoop",
		run:    runWithNonexistentTaskLoop,
		reason: taskloopv1alpha1.TaskLoopRunReasonCouldntGetTaskLoop,
		wantEvents: []string{
			"Normal Started ",
			"Warning Failed Error retrieving TaskLoop",
		},
	}, {
		name:     "missing iterate parameter",
		taskloop: aTaskLoop,
		run:      runWithMissingIterateParam,
		reason:   taskloopv1alpha1.TaskLoopRunReasonFailedValidation,
		wantEvents: []string{
			"Normal Started ",
			`Warning Failed Cannot determine number of iterations: The iterate parameter "current-item" was not found`,
		},
	}, {
		name:     "iterate parameter not an array",
		taskloop: aTaskLoop,
		run:      runWithIterateParamNotAnArray,
		reason:   taskloopv1alpha1.TaskLoopRunReasonFailedValidation,
		wantEvents: []string{
			"Normal Started ",
			`Warning Failed Cannot determine number of iterations: The value of the iterate parameter "current-item" is not an array`,
		},
	}}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {

			d := test.Data{
				Runs: []*v1alpha1.Run{tc.run},
			}

			optionalTaskLoop := []*taskloopv1alpha1.TaskLoop{tc.taskloop}
			if tc.taskloop == nil {
				optionalTaskLoop = nil
			}

			testAssets, _ := getTaskLoopController(t, d, optionalTaskLoop)
			c := testAssets.Controller
			clients := testAssets.Clients

			if err := c.Reconciler.Reconcile(context.Background(), getRunName(tc.run)); err != nil {
				t.Fatalf("Error reconciling: %s", err)
			}

			// Fetch the updated Run
			reconciledRun, err := clients.Pipeline.TektonV1alpha1().Runs(tc.run.Namespace).Get(tc.run.Name, metav1.GetOptions{})
			if err != nil {
				t.Fatalf("Error getting reconciled run from fake client: %s", err)
			}

			// Verify that the Run is in Failed status and both the start time and the completion time are set.
			checkRunCondition(t, reconciledRun, corev1.ConditionFalse, tc.reason)
			if reconciledRun.Status.StartTime == nil {
				t.Fatalf("Expected Run start time to be set but it wasn't")
			}
			if reconciledRun.Status.CompletionTime == nil {
				t.Fatalf("Expected Run completion time to be set but it wasn't")
			}

			if err := checkEvents(testAssets.Recorder, tc.name, tc.wantEvents); err != nil {
				t.Errorf(err.Error())
			}
		})
	}
}
