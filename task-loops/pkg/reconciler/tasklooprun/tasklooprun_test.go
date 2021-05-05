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
	"github.com/tektoncd/experimental/task-loops/test"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/pod"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	ttesting "github.com/tektoncd/pipeline/pkg/reconciler/testing"
	"github.com/tektoncd/pipeline/pkg/system"
	"github.com/tektoncd/pipeline/test/diff"
	"github.com/tektoncd/pipeline/test/names"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ktesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/record"
	"knative.dev/pkg/apis"
	cminformer "knative.dev/pkg/configmap/informer"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/reconciler"
)

var (
	concurrencyLimit1  = 1
	concurrencyLimit2  = 2
	noConcurrencyLimit = 0
	myPodTemplate      = &pod.Template{
		NodeSelector: map[string]string{
			"workloadtype": "tekton",
		},
	}
	myServiceAccountName = "special-account"
	myWorkspaces         = []v1beta1.WorkspaceBinding{{
		Name:     "myworkspace",
		EmptyDir: &corev1.EmptyDirVolumeSource{},
	}}
	namespace = ""
	trueB     = true
)

func getRunName(run *v1alpha1.Run) string {
	return strings.Join([]string{run.Namespace, run.Name}, "/")
}

func loopRunning(run *v1alpha1.Run) *v1alpha1.Run {
	runWithStatus := run.DeepCopy()
	runWithStatus.Status.InitializeConditions()
	runWithStatus.Status.MarkRunRunning(taskloopv1alpha1.TaskLoopRunReasonRunning.String(), "")
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

func withConcurrencyLimit(tl *taskloopv1alpha1.TaskLoop, concurrencyLimit int) *taskloopv1alpha1.TaskLoop {
	taskLoopWithConcurrency := tl.DeepCopy()
	taskLoopWithConcurrency.Spec.Concurrency = &concurrencyLimit
	return taskLoopWithConcurrency
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

func withUpdatedRunSpec(run *v1alpha1.Run, serviceAccountName string, workspaces []v1beta1.WorkspaceBinding, podTemplate *pod.Template) *v1alpha1.Run {
	runWithUpdatedSpec := run.DeepCopy()
	runWithUpdatedSpec.Spec.ServiceAccountName = serviceAccountName
	runWithUpdatedSpec.Spec.Workspaces = workspaces
	runWithUpdatedSpec.Spec.PodTemplate = podTemplate
	return runWithUpdatedSpec
}

func withUpdatedTaskRunSpec(tr *v1beta1.TaskRun, serviceAccountName string, workspaces []v1beta1.WorkspaceBinding, podTemplate *pod.Template) *v1beta1.TaskRun {
	trWithUpdatedSpec := tr.DeepCopy()
	trWithUpdatedSpec.Spec.ServiceAccountName = serviceAccountName
	trWithUpdatedSpec.Spec.Workspaces = workspaces
	trWithUpdatedSpec.Spec.PodTemplate = podTemplate
	return trWithUpdatedSpec
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
		if _, err := client.CustomV1alpha1().TaskLoops(tl.Namespace).Create(ctx, tl, metav1.CreateOptions{}); err != nil {
			t.Fatal(err)
		}
	}

	configMapWatcher := cminformer.NewInformedWatcher(c.Kube, system.GetNamespace())
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

func getCreatedTaskRuns(t *testing.T, clients test.Clients) []*v1beta1.TaskRun {
	createdTaskRuns := []*v1beta1.TaskRun{}
	t.Log("actions", clients.Pipeline.Actions())
	for _, a := range clients.Pipeline.Actions() {
		if a.GetVerb() == "create" {
			obj := a.(ktesting.CreateAction).GetObject()
			if tr, ok := obj.(*v1beta1.TaskRun); ok {
				createdTaskRuns = append(createdTaskRuns, tr)
			}
		}
	}
	return createdTaskRuns
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
		found := false
		for actualTaskRunName, actualTaskRunStatus := range status.TaskRuns {
			if strings.HasPrefix(actualTaskRunName, expectedTaskRunName) {
				found = true
				if actualTaskRunStatus.Iteration != expectedTaskRunStatus.Iteration {
					t.Errorf("Run status for TaskRun %s has iteration number %d instead of %d",
						actualTaskRunName, actualTaskRunStatus.Iteration, expectedTaskRunStatus.Iteration)
				}
				if d := cmp.Diff(expectedTaskRunStatus.Status, actualTaskRunStatus.Status, cmpopts.IgnoreTypes(apis.Condition{}.LastTransitionTime.Inner.Time)); d != "" {
					t.Errorf("Run status for TaskRun %s is incorrect. Diff %s", actualTaskRunName, diff.PrintWantGot(d))
				}
				break
			}
		}
		if !found {
			t.Errorf("Expected TaskRun with prefix %s for Run %s/%s not found",
				expectedTaskRunName, run.Namespace, run.Name)
			continue
		}
	}
}

// commonTaskSpec is reused in Task and inline task
var commonTaskSpec = v1beta1.TaskSpec{
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
}

var aTask = &v1beta1.Task{
	ObjectMeta: metav1.ObjectMeta{Name: "a-task", Namespace: "foo"},
	Spec:       commonTaskSpec,
}

var aTaskLoop = &taskloopv1alpha1.TaskLoop{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "a-taskloop",
		Namespace: "foo",
		Labels: map[string]string{
			"myTaskLoopLabel": "myTaskLoopLabelValue",
		},
		Annotations: map[string]string{
			"myTaskLoopAnnotation": "myTaskLoopAnnotationValue",
		},
	},
	Spec: taskloopv1alpha1.TaskLoopSpec{
		TaskRef:      &v1beta1.TaskRef{Name: "a-task"},
		IterateParam: "current-item",
	},
}

var aTaskLoopWithBundle = &taskloopv1alpha1.TaskLoop{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "a-taskloop",
		Namespace: "foo",
		Labels: map[string]string{
			"myTaskLoopLabel": "myTaskLoopLabelValue",
		},
		Annotations: map[string]string{
			"myTaskLoopAnnotation": "myTaskLoopAnnotationValue",
		},
	},
	Spec: taskloopv1alpha1.TaskLoopSpec{
		TaskRef:      &v1beta1.TaskRef{Name: "a-task", Bundle: "a-bundle"},
		IterateParam: "current-item",
	},
}

var aTaskLoopWithInlineTask = &taskloopv1alpha1.TaskLoop{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "a-taskloop-with-inline-task",
		Namespace: "foo",
		// No labels or annotations in this one to test that case works
	},
	Spec: taskloopv1alpha1.TaskLoopSpec{
		TaskSpec:     &commonTaskSpec,
		IterateParam: "current-item",
		Timeout:      &metav1.Duration{Duration: 5 * time.Minute},
	},
}

var runTaskLoop = &v1alpha1.Run{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "run-taskloop",
		Namespace: "foo",
		Labels: map[string]string{
			"myRunLabel": "myRunLabelValue",
		},
		Annotations: map[string]string{
			"myRunAnnotation": "myRunAnnotationValue",
		},
	},
	Spec: v1alpha1.RunSpec{
		Params: []v1beta1.Param{{
			Name:  "current-item",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeArray, ArrayVal: []string{"item1", "item2", "item3"}},
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
		// No labels or annotations in this one to test that case works
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

var runWithIterateParamNotAnArray = &v1alpha1.Run{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "run-taskloop",
		Namespace: "foo",
		Labels: map[string]string{
			"myRunLabel": "myRunLabelValue",
		},
		Annotations: map[string]string{
			"myRunAnnotation": "myRunAnnotationValue",
		},
	},
	Spec: v1alpha1.RunSpec{
		Params: []v1beta1.Param{{
			// Value of iteration parameter must be an array so this is an error.
			Name:  "current-item",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "item1\nitem2"},
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

var expectedTaskRunIteration1 = &v1beta1.TaskRun{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "run-taskloop-00001-", // does not include random suffix
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
			"myTaskLoopLabel":                     "myTaskLoopLabelValue",
			"myRunLabel":                          "myRunLabelValue",
		},
		Annotations: map[string]string{
			"myTaskLoopAnnotation": "myTaskLoopAnnotationValue",
			"myRunAnnotation":      "myRunAnnotationValue",
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
		ServiceAccountName: "default",
	},
}

var expectedTaskRunWithBundle = &v1beta1.TaskRun{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "run-taskloop-00001-", // does not include random suffix
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
			"myTaskLoopLabel":                     "myTaskLoopLabelValue",
			"myRunLabel":                          "myRunLabelValue",
		},
		Annotations: map[string]string{
			"myTaskLoopAnnotation": "myTaskLoopAnnotationValue",
			"myRunAnnotation":      "myRunAnnotationValue",
		},
	},
	Spec: v1beta1.TaskRunSpec{
		TaskRef: &v1beta1.TaskRef{Name: "a-task", Bundle: "a-bundle"},
		Params: []v1beta1.Param{{
			Name:  "current-item",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "item1"},
		}, {
			Name:  "additional-parameter",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "stuff"},
		}},
		ServiceAccountName: "default",
	},
}

var expectedTaskRunIteration2 = &v1beta1.TaskRun{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "run-taskloop-00002-", // does not include random suffix
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
			"myTaskLoopLabel":                     "myTaskLoopLabelValue",
			"myRunLabel":                          "myRunLabelValue",
		},
		Annotations: map[string]string{
			"myTaskLoopAnnotation": "myTaskLoopAnnotationValue",
			"myRunAnnotation":      "myRunAnnotationValue",
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
		ServiceAccountName: "default",
	},
}

var expectedTaskRunIteration3 = &v1beta1.TaskRun{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "run-taskloop-00003-", // does not include random suffix
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
			"custom.tekton.dev/taskLoopIteration": "3",
			"myTaskLoopLabel":                     "myTaskLoopLabelValue",
			"myRunLabel":                          "myRunLabelValue",
		},
		Annotations: map[string]string{
			"myTaskLoopAnnotation": "myTaskLoopAnnotationValue",
			"myRunAnnotation":      "myRunAnnotationValue",
		},
	},
	Spec: v1beta1.TaskRunSpec{
		TaskRef: &v1beta1.TaskRef{Name: "a-task"},
		Params: []v1beta1.Param{{
			Name:  "current-item",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "item3"},
		}, {
			Name:  "additional-parameter",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "stuff"},
		}},
		ServiceAccountName: "default",
	},
}

var expectedTaskRunWithInlineTaskIteration1 = &v1beta1.TaskRun{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "run-taskloop-with-inline-task-00001-", // does not include random suffix
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
		},
		Annotations: map[string]string{},
	},
	Spec: v1beta1.TaskRunSpec{
		TaskSpec: &commonTaskSpec,
		Params: []v1beta1.Param{{
			Name:  "current-item",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "item1"},
		}, {
			Name:  "additional-parameter",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "stuff"},
		}},
		ServiceAccountName: "default",
		Timeout:            &metav1.Duration{Duration: 5 * time.Minute},
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
		name:             "Reconcile a new run with a taskloop that references a task with Bundle",
		task:             aTask,
		taskloop:         aTaskLoopWithBundle,
		run:              runTaskLoop,
		taskruns:         []*v1beta1.TaskRun{},
		expectedStatus:   corev1.ConditionUnknown,
		expectedReason:   taskloopv1alpha1.TaskLoopRunReasonRunning,
		expectedTaskruns: []*v1beta1.TaskRun{expectedTaskRunWithBundle},
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
		name:             "Reconcile a new run that uses all the bells and whistles (tests propagation to TaskRun)",
		task:             aTask,
		taskloop:         aTaskLoop,
		run:              withUpdatedRunSpec(runTaskLoop, myServiceAccountName, myWorkspaces, myPodTemplate),
		taskruns:         []*v1beta1.TaskRun{},
		expectedStatus:   corev1.ConditionUnknown,
		expectedReason:   taskloopv1alpha1.TaskLoopRunReasonRunning,
		expectedTaskruns: []*v1beta1.TaskRun{withUpdatedTaskRunSpec(expectedTaskRunIteration1, myServiceAccountName, myWorkspaces, myPodTemplate)},
		expectedEvents:   []string{"Normal Started", "Normal Running Iterations completed: 0"},
	}, {
		name:             "Reconcile a run after the first TaskRun has succeeded",
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
		taskruns:         []*v1beta1.TaskRun{successful(expectedTaskRunIteration1), successful(expectedTaskRunIteration2), successful(expectedTaskRunIteration3)},
		expectedStatus:   corev1.ConditionTrue,
		expectedReason:   taskloopv1alpha1.TaskLoopRunReasonSucceeded,
		expectedTaskruns: []*v1beta1.TaskRun{successful(expectedTaskRunIteration1), successful(expectedTaskRunIteration2), successful(expectedTaskRunIteration3)},
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
		expectedEvents:   []string{"Warning Failed One or more TaskRuns have failed"},
	}, {
		name:             "Reconcile a run after the first TaskRun has failed and retry is allowed",
		task:             aTask,
		taskloop:         allowRetry(aTaskLoop),
		run:              loopRunning(runTaskLoop),
		taskruns:         []*v1beta1.TaskRun{failed(expectedTaskRunIteration1)},
		expectedStatus:   corev1.ConditionUnknown,
		expectedReason:   taskloopv1alpha1.TaskLoopRunReasonRunning,
		expectedTaskruns: []*v1beta1.TaskRun{retrying(failed(expectedTaskRunIteration1))},
		expectedEvents:   []string{"Normal Running Iterations completed: 0"},
	}, {
		name:             "Reconcile a run after the first TaskRun has failed and retry failed as well",
		task:             aTask,
		taskloop:         allowRetry(aTaskLoop),
		run:              loopRunning(runTaskLoop),
		taskruns:         []*v1beta1.TaskRun{failed(retrying(failed(expectedTaskRunIteration1)))},
		expectedStatus:   corev1.ConditionFalse,
		expectedReason:   taskloopv1alpha1.TaskLoopRunReasonFailed,
		expectedTaskruns: []*v1beta1.TaskRun{failed(retrying(failed(expectedTaskRunIteration1)))},
		expectedEvents:   []string{"Warning Failed One or more TaskRuns have failed"},
	}, {
		name:             "Reconcile a cancelled run while the first TaskRun is running",
		task:             aTask,
		taskloop:         aTaskLoop,
		run:              requestCancel(loopRunning(runTaskLoop)),
		taskruns:         []*v1beta1.TaskRun{running(expectedTaskRunIteration1)},
		expectedStatus:   corev1.ConditionUnknown,
		expectedReason:   taskloopv1alpha1.TaskLoopRunReasonRunning,
		expectedTaskruns: []*v1beta1.TaskRun{running(expectedTaskRunIteration1)},
		expectedEvents:   []string{"Normal Running Cancelling TaskRuns"},
	}, {
		name:             "Reconcile a cancelled run after the first TaskRun has failed",
		task:             aTask,
		taskloop:         aTaskLoop,
		run:              requestCancel(loopRunning(runTaskLoop)),
		taskruns:         []*v1beta1.TaskRun{failed(expectedTaskRunIteration1)},
		expectedStatus:   corev1.ConditionFalse,
		expectedReason:   v1alpha1.RunReasonCancelled,
		expectedTaskruns: []*v1beta1.TaskRun{failed(expectedTaskRunIteration1)},
		expectedEvents:   []string{"Warning Failed Run " + runTaskLoop.Namespace + "/" + runTaskLoop.Name + " was cancelled"},
	}, {
		name:             "Reconcile a cancelled run after the first TaskRun has succeeded",
		task:             aTask,
		taskloop:         aTaskLoop,
		run:              requestCancel(loopRunning(runTaskLoop)),
		taskruns:         []*v1beta1.TaskRun{successful(expectedTaskRunIteration1)},
		expectedStatus:   corev1.ConditionFalse,
		expectedReason:   v1alpha1.RunReasonCancelled,
		expectedTaskruns: []*v1beta1.TaskRun{successful(expectedTaskRunIteration1)},
		expectedEvents:   []string{"Warning Failed Run " + runTaskLoop.Namespace + "/" + runTaskLoop.Name + " was cancelled"},
	}, {
		name:             "Reconcile a new run with a taskloop that explicitly requests sequential execution",
		task:             aTask,
		taskloop:         withConcurrencyLimit(aTaskLoop, concurrencyLimit1),
		run:              runTaskLoop,
		taskruns:         []*v1beta1.TaskRun{},
		expectedStatus:   corev1.ConditionUnknown,
		expectedReason:   taskloopv1alpha1.TaskLoopRunReasonRunning,
		expectedTaskruns: []*v1beta1.TaskRun{expectedTaskRunIteration1},
		expectedEvents:   []string{"Normal Started", "Normal Running Iterations completed: 0"},
	}, {
		name:             "Reconcile a new run with a taskloop that allows limited concurrency",
		task:             aTask,
		taskloop:         withConcurrencyLimit(aTaskLoop, concurrencyLimit2),
		run:              runTaskLoop,
		taskruns:         []*v1beta1.TaskRun{},
		expectedStatus:   corev1.ConditionUnknown,
		expectedReason:   taskloopv1alpha1.TaskLoopRunReasonRunning,
		expectedTaskruns: []*v1beta1.TaskRun{expectedTaskRunIteration1, expectedTaskRunIteration2},
		expectedEvents:   []string{"Normal Started", "Normal Running Iterations completed: 0"},
	}, {
		name:             "Reconcile a new run with a taskloop that allows unlimited concurrency",
		task:             aTask,
		taskloop:         withConcurrencyLimit(aTaskLoop, noConcurrencyLimit),
		run:              runTaskLoop,
		taskruns:         []*v1beta1.TaskRun{},
		expectedStatus:   corev1.ConditionUnknown,
		expectedReason:   taskloopv1alpha1.TaskLoopRunReasonRunning,
		expectedTaskruns: []*v1beta1.TaskRun{expectedTaskRunIteration1, expectedTaskRunIteration2, expectedTaskRunIteration3},
		expectedEvents:   []string{"Normal Started", "Normal Running Iterations completed: 0"},
	}, {
		name:             "Reconcile a run that allows limited concurrency after the first TaskRun has succeeded",
		task:             aTask,
		taskloop:         withConcurrencyLimit(aTaskLoop, concurrencyLimit2),
		run:              loopRunning(runTaskLoop),
		taskruns:         []*v1beta1.TaskRun{successful(expectedTaskRunIteration1), running(expectedTaskRunIteration2)},
		expectedStatus:   corev1.ConditionUnknown,
		expectedReason:   taskloopv1alpha1.TaskLoopRunReasonRunning,
		expectedTaskruns: []*v1beta1.TaskRun{successful(expectedTaskRunIteration1), running(expectedTaskRunIteration2), expectedTaskRunIteration3},
		expectedEvents:   []string{"Normal Running Iterations completed: 1"},
	}, {
		name:             "Reconcile a run that allows limited concurrency after the first TaskRun has failed but the other is still running",
		task:             aTask,
		taskloop:         withConcurrencyLimit(aTaskLoop, concurrencyLimit2),
		run:              loopRunning(runTaskLoop),
		taskruns:         []*v1beta1.TaskRun{failed(expectedTaskRunIteration1), running(expectedTaskRunIteration2)},
		expectedStatus:   corev1.ConditionUnknown,
		expectedReason:   taskloopv1alpha1.TaskLoopRunReasonRunning,
		expectedTaskruns: []*v1beta1.TaskRun{failed(expectedTaskRunIteration1), running(expectedTaskRunIteration2)}, // no new TaskRun
		expectedEvents:   []string{"Normal Running Iterations completed: 1"},
	}, {
		name:             "Reconcile a run where the iterate parameter is not an array",
		task:             aTask,
		taskloop:         aTaskLoop,
		run:              runWithIterateParamNotAnArray,
		expectedStatus:   corev1.ConditionUnknown,
		expectedReason:   taskloopv1alpha1.TaskLoopRunReasonRunning,
		expectedTaskruns: []*v1beta1.TaskRun{expectedTaskRunIteration1}, // The first taskrun was started
		expectedEvents:   []string{"Normal Started", "Normal Running Iterations completed: 0"},
	}, {
		name:             "Reconcile a run that allows limited concurrency where the iterate parameter is not an array",
		task:             aTask,
		taskloop:         withConcurrencyLimit(aTaskLoop, concurrencyLimit2),
		run:              runWithIterateParamNotAnArray,
		expectedStatus:   corev1.ConditionUnknown,
		expectedReason:   taskloopv1alpha1.TaskLoopRunReasonRunning,
		expectedTaskruns: []*v1beta1.TaskRun{expectedTaskRunIteration1, expectedTaskRunIteration2}, // The first taskrun was started
		expectedEvents:   []string{"Normal Started", "Normal Running Iterations completed: 0"},
	}}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
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

			if err := c.Reconciler.Reconcile(ctx, getRunName(tc.run)); err != nil {
				t.Fatalf("Error reconciling: %s", err)
			}

			// Fetch the updated Run
			reconciledRun, err := clients.Pipeline.TektonV1alpha1().Runs(tc.run.Namespace).Get(ctx, tc.run.Name, metav1.GetOptions{})
			if err != nil {
				t.Fatalf("Error getting reconciled run from fake client: %s", err)
			}

			// Verify that the Run has the expected status and reason.
			checkRunCondition(t, reconciledRun, tc.expectedStatus, tc.expectedReason)

			// Verify that TaskRun(s) were or were not created depending on the test.
			// If the number of expected TaskRuns is greater than the original number of TaskRuns
			// then the test expects new TaskRuns to be created.  The new TaskRuns must be at the
			// end of the list of expected TaskRuns.
			createdTaskRuns := getCreatedTaskRuns(t, clients)
			numberofExpectedNewTaskRuns := len(tc.expectedTaskruns) - len(tc.taskruns)
			numberOfActualNewTaskRuns := len(createdTaskRuns)
			n := numberofExpectedNewTaskRuns
			if numberOfActualNewTaskRuns > numberofExpectedNewTaskRuns {
				n = numberOfActualNewTaskRuns
			}
			for i := 0; i < n; i++ {
				if i >= numberofExpectedNewTaskRuns {
					t.Errorf("A TaskRun %s was created which was not expected", createdTaskRuns[i].ObjectMeta.Name)
				} else {
					expectedNewTaskRun := tc.expectedTaskruns[len(tc.taskruns)+i]
					if i >= numberOfActualNewTaskRuns {
						t.Errorf("A TaskRun with prefix %s should have been created but was not", expectedNewTaskRun.ObjectMeta.Name)
					} else {
						if !strings.HasPrefix(createdTaskRuns[i].ObjectMeta.Name, expectedNewTaskRun.ObjectMeta.Name) {
							t.Errorf("A TaskRun %s was created but the name does not have expected prefix %s",
								createdTaskRuns[i].ObjectMeta.Name, expectedNewTaskRun.ObjectMeta.Name)
						}
						if d := cmp.Diff(expectedNewTaskRun, createdTaskRuns[i],
							cmpopts.IgnoreFields(metav1.ObjectMeta{}, "Name")); d != "" {
							t.Errorf("Expected TaskRun was not created. Diff %s", diff.PrintWantGot(d))
						}
					}
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
	}}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

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

			if err := c.Reconciler.Reconcile(ctx, getRunName(tc.run)); err != nil {
				t.Fatalf("Error reconciling: %s", err)
			}

			// Fetch the updated Run
			reconciledRun, err := clients.Pipeline.TektonV1alpha1().Runs(tc.run.Namespace).Get(ctx, tc.run.Name, metav1.GetOptions{})
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
