// +build e2e

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

package test

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/tektoncd/experimental/task-loops/pkg/apis/taskloop"
	taskloopv1alpha1 "github.com/tektoncd/experimental/task-loops/pkg/apis/taskloop/v1alpha1"
	"github.com/tektoncd/experimental/task-loops/pkg/client/clientset/versioned"
	resourceversioned "github.com/tektoncd/experimental/task-loops/pkg/client/clientset/versioned/typed/taskloop/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/names"
	"github.com/tektoncd/pipeline/pkg/pod"
	"github.com/tektoncd/pipeline/test/diff"
	"gomodules.xyz/jsonpatch/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"knative.dev/pkg/apis"
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"
	knativetest "knative.dev/pkg/test"
)

var (
	numRetries              = 2 // number of task retries to test
	runTimeout              = 10 * time.Minute
	startedEventMessage     = "" // Run started event has no message
	taskTimeout             = &metav1.Duration{Duration: 10 * time.Second}
	ignoreReleaseAnnotation = func(k string, v string) bool {
		return k == pod.ReleaseAnnotation
	}
)

// commonTaskSpec is reused in Task, Cluster Task, and inline task
var commonTaskSpec = v1beta1.TaskSpec{
	Params: []v1beta1.ParamSpec{{
		Name: "current-item",
		Type: v1beta1.ParamTypeString,
	}, {
		Name:    "fail-on-item",
		Type:    v1beta1.ParamTypeString,
		Default: &v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: ""},
	}},
	Steps: []v1beta1.Step{{
		Container: corev1.Container{
			Name:    "passfail",
			Image:   "ubuntu",
			Command: []string{"/bin/bash"},
			Args:    []string{"-c", `[[ "$(params.fail-on-item)" == "" || "$(params.current-item)" != "$(params.fail-on-item)" ]]`},
		},
	}},
}

var aTask = &v1beta1.Task{
	ObjectMeta: metav1.ObjectMeta{Name: "a-task"},
	Spec:       commonTaskSpec,
}

// Create cluster task name with randomized suffix to avoid name clashes
var clusterTaskName = names.SimpleNameGenerator.RestrictLengthWithRandomSuffix("a-cluster-task")

var aClusterTask = &v1beta1.ClusterTask{
	ObjectMeta: metav1.ObjectMeta{Name: clusterTaskName},
	Spec:       commonTaskSpec,
}

var aTaskLoop = &taskloopv1alpha1.TaskLoop{
	ObjectMeta: metav1.ObjectMeta{
		Name: "a-taskloop",
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

var aTaskLoopUsingAnInlineTask = &taskloopv1alpha1.TaskLoop{
	ObjectMeta: metav1.ObjectMeta{
		Name: "a-taskloop-using-an-inline-task",
		// No labels or annotations in this one to test that case works
	},
	Spec: taskloopv1alpha1.TaskLoopSpec{
		TaskSpec:     &commonTaskSpec,
		IterateParam: "current-item",
	},
}

var aTaskLoopUsingAClusterTask = &taskloopv1alpha1.TaskLoop{
	ObjectMeta: metav1.ObjectMeta{
		Name: "a-taskloop-using-a-cluster-task",
		// No labels or annotations in this one to test that case works
	},
	Spec: taskloopv1alpha1.TaskLoopSpec{
		TaskRef:      &v1beta1.TaskRef{Name: clusterTaskName, Kind: "ClusterTask"},
		IterateParam: "current-item",
	},
}

var runTaskLoopSuccess = &v1alpha1.Run{
	ObjectMeta: metav1.ObjectMeta{
		Name: "run-taskloop",
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
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeArray, ArrayVal: []string{"item1", "item2"}},
		}},
		Ref: &v1alpha1.TaskRef{
			APIVersion: taskloopv1alpha1.SchemeGroupVersion.String(),
			Kind:       taskloop.TaskLoopControllerName,
			Name:       "a-taskloop",
		},
	},
}

var runTaskLoopFailure = &v1alpha1.Run{
	ObjectMeta: metav1.ObjectMeta{
		Name: "run-taskloop",
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
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeArray, ArrayVal: []string{"item1", "item2"}},
		}, {
			Name:  "fail-on-item",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "item1"},
		}},
		Ref: &v1alpha1.TaskRef{
			APIVersion: taskloopv1alpha1.SchemeGroupVersion.String(),
			Kind:       taskloop.TaskLoopControllerName,
			Name:       "a-taskloop",
		},
	},
}

var runTaskLoopUsingAnInlineTaskSuccess = &v1alpha1.Run{
	ObjectMeta: metav1.ObjectMeta{
		Name: "run-taskloop",
		// No labels or annotations in this one to test that case works
	},
	Spec: v1alpha1.RunSpec{
		Params: []v1beta1.Param{{
			Name:  "current-item",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeArray, ArrayVal: []string{"item1", "item2"}},
		}},
		Ref: &v1alpha1.TaskRef{
			APIVersion: taskloopv1alpha1.SchemeGroupVersion.String(),
			Kind:       taskloop.TaskLoopControllerName,
			Name:       "a-taskloop-using-an-inline-task",
		},
	},
}

var runTaskLoopUsingAClusterTaskSuccess = &v1alpha1.Run{
	ObjectMeta: metav1.ObjectMeta{
		Name: "run-taskloop",
		// No labels or annotations in this one to test that case works
	},
	Spec: v1alpha1.RunSpec{
		Params: []v1beta1.Param{{
			Name:  "current-item",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeArray, ArrayVal: []string{"item1", "item2"}},
		}},
		Ref: &v1alpha1.TaskRef{
			APIVersion: taskloopv1alpha1.SchemeGroupVersion.String(),
			Kind:       taskloop.TaskLoopControllerName,
			Name:       "a-taskloop-using-a-cluster-task",
		},
	},
}

var taskRunStatusSuccess = duckv1beta1.Status{
	Conditions: []apis.Condition{{
		Type:   apis.ConditionSucceeded,
		Status: corev1.ConditionTrue,
		Reason: v1beta1.TaskRunReasonSuccessful.String(),
	}},
}

var taskRunStatusFailed = duckv1beta1.Status{
	Conditions: []apis.Condition{{
		Type:   apis.ConditionSucceeded,
		Status: corev1.ConditionFalse,
		Reason: v1beta1.TaskRunReasonFailed.String(),
	}},
}

var taskRunStatusTimeout = duckv1beta1.Status{
	Conditions: []apis.Condition{{
		Type:   apis.ConditionSucceeded,
		Status: corev1.ConditionFalse,
		Reason: v1beta1.TaskRunReasonTimedOut.String(),
	}},
}

var expectedTaskRunIteration1Success = &v1beta1.TaskRun{
	ObjectMeta: metav1.ObjectMeta{
		Name: "run-taskloop-00001-", // does not include random suffix
		// Expected labels and annotations are added dynamically
	},
	Spec: v1beta1.TaskRunSpec{
		ServiceAccountName: "default", // default service account name
		TaskRef:            &v1beta1.TaskRef{Name: "a-task", Kind: "Task"},
		Timeout:            &metav1.Duration{Duration: 1 * time.Hour}, // default TaskRun timeout
		Params: []v1beta1.Param{{
			Name:  "current-item",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "item1"},
		}},
	},
	Status: v1beta1.TaskRunStatus{
		Status: taskRunStatusSuccess,
	},
}

var expectedTaskRunIteration2Success = &v1beta1.TaskRun{
	ObjectMeta: metav1.ObjectMeta{
		Name: "run-taskloop-00002-", // does not include random suffix
		// Expected labels and annotations are added dynamically
	},
	Spec: v1beta1.TaskRunSpec{
		ServiceAccountName: "default", // default service account name
		TaskRef:            &v1beta1.TaskRef{Name: "a-task", Kind: "Task"},
		Timeout:            &metav1.Duration{Duration: 1 * time.Hour}, // default TaskRun timeout
		Params: []v1beta1.Param{{
			Name:  "current-item",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "item2"},
		}},
	},
	Status: v1beta1.TaskRunStatus{
		Status: taskRunStatusSuccess,
	},
}

var expectedTaskRunIteration1Failure = &v1beta1.TaskRun{
	ObjectMeta: metav1.ObjectMeta{
		Name: "run-taskloop-00001-", // does not include random suffix
		// Expected labels and annotations are added dynamically
	},
	Spec: v1beta1.TaskRunSpec{
		ServiceAccountName: "default", // default service account name
		TaskRef:            &v1beta1.TaskRef{Name: "a-task", Kind: "Task"},
		Timeout:            &metav1.Duration{Duration: 1 * time.Hour}, // default TaskRun timeout
		Params: []v1beta1.Param{{
			Name:  "current-item",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "item1"},
		}, {
			Name:  "fail-on-item",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "item1"},
		}},
	},
	Status: v1beta1.TaskRunStatus{
		Status: taskRunStatusFailed,
	},
}

var sleepyTaskLoop = &taskloopv1alpha1.TaskLoop{
	ObjectMeta: metav1.ObjectMeta{Name: "sleepyloop"},
	Spec: taskloopv1alpha1.TaskLoopSpec{
		TaskSpec: &v1beta1.TaskSpec{
			Params: []v1beta1.ParamSpec{{
				Name: "sleep-time",
				Type: v1beta1.ParamTypeString,
			}},
			Steps: []v1beta1.Step{{
				Container: corev1.Container{
					Image: "busybox",
				},
				Script: "sleep $(params.sleep-time)",
			}},
		},
		IterateParam: "sleep-time",
	},
}

var runSleepyTaskLoop = &v1alpha1.Run{
	ObjectMeta: metav1.ObjectMeta{Name: "run-sleepy"},
	Spec: v1alpha1.RunSpec{
		Params: []v1beta1.Param{{
			Name:  "sleep-time",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeArray, ArrayVal: []string{"120", "120"}},
		}},
		Ref: &v1alpha1.TaskRef{
			APIVersion: taskloopv1alpha1.SchemeGroupVersion.String(),
			Kind:       taskloop.TaskLoopControllerName,
			Name:       "sleepyloop",
		},
	},
}

var expectedTaskRunIteration1Timeout = &v1beta1.TaskRun{
	ObjectMeta: metav1.ObjectMeta{
		Name: "run-sleepy-00001-", // does not include random suffix
		// Expected labels and annotations are added dynamically
	},
	Spec: v1beta1.TaskRunSpec{
		ServiceAccountName: "default", // default service account name
		Timeout:            taskTimeout,
		TaskSpec:           sleepyTaskLoop.Spec.TaskSpec,
		Params: []v1beta1.Param{{
			Name:  "sleep-time",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "120"},
		}},
	},
	Status: v1beta1.TaskRunStatus{
		Status: taskRunStatusTimeout,
	},
}

func TestTaskLoopRun(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name string
		// The following set of fields describe the resources to create.
		task        *v1beta1.Task
		clustertask *v1beta1.ClusterTask
		taskloop    *taskloopv1alpha1.TaskLoop
		run         *v1alpha1.Run
		// The following set of fields describe the expected outcome.
		expectedStatus   corev1.ConditionStatus
		expectedReason   taskloopv1alpha1.TaskLoopRunReason
		expectedTaskRuns []*v1beta1.TaskRun
		expectedEvents   []string
		// This function can perform additional checks on the TaskRun.  It is passed the expected and actual TaskRuns.
		extraTaskRunChecks func(*testing.T, *v1beta1.TaskRun, *v1beta1.TaskRun)
	}{{
		name:             "successful TaskLoop",
		task:             aTask,
		taskloop:         aTaskLoop,
		run:              runTaskLoopSuccess,
		expectedStatus:   corev1.ConditionTrue,
		expectedReason:   taskloopv1alpha1.TaskLoopRunReasonSucceeded,
		expectedTaskRuns: []*v1beta1.TaskRun{expectedTaskRunIteration1Success, expectedTaskRunIteration2Success},
		expectedEvents:   []string{startedEventMessage, "Iterations completed: 0", "Iterations completed: 1", "All TaskRuns completed successfully"},
	}, {
		name:             "failed TaskLoop",
		task:             aTask,
		taskloop:         aTaskLoop,
		run:              runTaskLoopFailure,
		expectedStatus:   corev1.ConditionFalse,
		expectedReason:   taskloopv1alpha1.TaskLoopRunReasonFailed,
		expectedTaskRuns: []*v1beta1.TaskRun{expectedTaskRunIteration1Failure},
		expectedEvents:   []string{startedEventMessage, "Iterations completed: 0", "TaskRun run-taskloop-00001-.* has failed"},
	}, {
		name:               "failed TaskLoop with retries",
		task:               aTask,
		taskloop:           getTaskLoopWithRetries(aTaskLoop),
		run:                runTaskLoopFailure,
		expectedStatus:     corev1.ConditionFalse,
		expectedReason:     taskloopv1alpha1.TaskLoopRunReasonFailed,
		expectedTaskRuns:   []*v1beta1.TaskRun{expectedTaskRunIteration1Failure},
		expectedEvents:     []string{startedEventMessage, "Iterations completed: 0", "TaskRun run-taskloop-00001-.* has failed"},
		extraTaskRunChecks: checkTaskRunRetries,
	}, {
		name:             "failed TaskLoop due to timeout",
		taskloop:         getTaskLoopWithTimeout(sleepyTaskLoop),
		run:              runSleepyTaskLoop,
		expectedStatus:   corev1.ConditionFalse,
		expectedReason:   taskloopv1alpha1.TaskLoopRunReasonFailed,
		expectedTaskRuns: []*v1beta1.TaskRun{expectedTaskRunIteration1Timeout},
		expectedEvents:   []string{startedEventMessage, "Iterations completed: 0", "TaskRun run-sleepy-00001-.* has failed"},
	}, {
		name:           "successful TaskLoop using an inline task",
		taskloop:       aTaskLoopUsingAnInlineTask,
		run:            runTaskLoopUsingAnInlineTaskSuccess,
		expectedStatus: corev1.ConditionTrue,
		expectedReason: taskloopv1alpha1.TaskLoopRunReasonSucceeded,
		expectedTaskRuns: []*v1beta1.TaskRun{
			getExpectedTaskRunForInlineTask(expectedTaskRunIteration1Success),
			getExpectedTaskRunForInlineTask(expectedTaskRunIteration2Success),
		},
		expectedEvents: []string{startedEventMessage, "Iterations completed: 0", "Iterations completed: 1", "All TaskRuns completed successfully"},
	}, {
		name:           "successful TaskLoop using a cluster task",
		clustertask:    aClusterTask,
		taskloop:       aTaskLoopUsingAClusterTask,
		run:            runTaskLoopUsingAClusterTaskSuccess,
		expectedStatus: corev1.ConditionTrue,
		expectedReason: taskloopv1alpha1.TaskLoopRunReasonSucceeded,
		expectedTaskRuns: []*v1beta1.TaskRun{
			getExpectedTaskRunForClusterTask(expectedTaskRunIteration1Success),
			getExpectedTaskRunForClusterTask(expectedTaskRunIteration2Success),
		},
		expectedEvents: []string{startedEventMessage, "Iterations completed: 0", "Iterations completed: 1", "All TaskRuns completed successfully"},
	}}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			tc := tc // Copy current tc to local variable due to test parallelization
			t.Parallel()
			c, namespace := setup(t)
			taskLoopClient := getTaskLoopClient(t, namespace)

			knativetest.CleanupOnInterrupt(func() { tearDown(t, c, namespace) }, t.Logf)
			defer tearDown(t, c, namespace)

			if tc.task != nil {
				task := tc.task.DeepCopy()
				task.Namespace = namespace
				if _, err := c.TaskClient.Create(task); err != nil {
					t.Fatalf("Failed to create Task `%s`: %s", task.Name, err)
				}
			}

			if tc.clustertask != nil {
				if _, err := c.ClusterTaskClient.Create(tc.clustertask); err != nil {
					t.Fatalf("Failed to create ClusterTask `%s`: %s", tc.clustertask.Name, err)
				}
				knativetest.CleanupOnInterrupt(func() { deleteClusterTask(t, c, tc.clustertask.Name) }, t.Logf)
				defer deleteClusterTask(t, c, tc.clustertask.Name)
			}

			if tc.taskloop != nil {
				taskloop := tc.taskloop.DeepCopy()
				taskloop.Namespace = namespace
				if _, err := taskLoopClient.Create(taskloop); err != nil {
					t.Fatalf("Failed to create TaskLoop `%s`: %s", tc.taskloop.Name, err)
				}
			}

			run := tc.run.DeepCopy()
			run.Namespace = namespace
			run, err := c.RunClient.Create(tc.run)
			if err != nil {
				t.Fatalf("Failed to create Run `%s`: %s", run.Name, err)
			}

			t.Logf("Waiting for Run %s in namespace %s to complete", run.Name, run.Namespace)
			var inState ConditionAccessorFn
			var desc string
			if tc.expectedStatus == corev1.ConditionTrue {
				inState = Succeed(run.Name)
				desc = "RunSuccess"
			} else {
				inState = FailedWithReason(tc.expectedReason.String(), run.Name)
				desc = "RunFailed"
			}
			if err := WaitForRunState(c, run.Name, runTimeout, inState, desc); err != nil {
				t.Fatalf("Error waiting for Run %s/%s to finish: %s", run.Namespace, run.Name, err)
			}

			run, err = c.RunClient.Get(run.Name, metav1.GetOptions{})
			if err != nil {
				t.Fatalf("Couldn't get expected Run %s/%s: %s", run.Namespace, run.Name, err)
			}

			t.Logf("Making sure the expected TaskRuns were created")
			actualTaskRunList, err := c.TaskRunClient.List(metav1.ListOptions{LabelSelector: fmt.Sprintf("tekton.dev/run=%s", run.Name)})
			if err != nil {
				t.Fatalf("Error listing TaskRuns for Run %s/%s: %s", run.Namespace, run.Name, err)
			}

			if len(tc.expectedTaskRuns) != len(actualTaskRunList.Items) {
				t.Errorf("Expected %d TaskRuns for Run %s/%s but found %d",
					len(tc.expectedTaskRuns), run.Namespace, run.Name, len(actualTaskRunList.Items))
			}

			// Check TaskRun status in the Run's status.
			status := &taskloopv1alpha1.TaskLoopRunStatus{}
			if err := run.Status.DecodeExtraFields(status); err != nil {
				t.Errorf("DecodeExtraFields error: %v", err.Error())
			}
			for i, expectedTaskRun := range tc.expectedTaskRuns {
				expectedTaskRun = expectedTaskRun.DeepCopy()
				expectedTaskRun.ObjectMeta.Annotations = getExpectedTaskRunAnnotations(tc.taskloop, run)
				expectedTaskRun.ObjectMeta.Labels = getExpectedTaskRunLabels(tc.task, tc.clustertask, tc.taskloop, run, i+1)
				var actualTaskRun v1beta1.TaskRun
				found := false
				for _, actualTaskRun = range actualTaskRunList.Items {
					if strings.HasPrefix(actualTaskRun.Name, expectedTaskRun.Name) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected TaskRun with prefix %s for Run %s/%s not found",
						expectedTaskRun.Name, run.Namespace, run.Name)
					continue
				}
				if d := cmp.Diff(expectedTaskRun.Spec, actualTaskRun.Spec); d != "" {
					t.Errorf("TaskRun %s spec does not match expected spec. Diff %s", actualTaskRun.Name, diff.PrintWantGot(d))
				}
				if d := cmp.Diff(expectedTaskRun.ObjectMeta.Annotations, actualTaskRun.ObjectMeta.Annotations,
					cmpopts.IgnoreMapEntries(ignoreReleaseAnnotation)); d != "" {
					t.Errorf("TaskRun %s does not have expected annotations. Diff %s", actualTaskRun.Name, diff.PrintWantGot(d))
				}
				if d := cmp.Diff(expectedTaskRun.ObjectMeta.Labels, actualTaskRun.ObjectMeta.Labels); d != "" {
					t.Errorf("TaskRun %s does not have expected labels. Diff %s", actualTaskRun.Name, diff.PrintWantGot(d))
				}
				if d := cmp.Diff(expectedTaskRun.Status.Status.Conditions, actualTaskRun.Status.Status.Conditions,
					cmpopts.IgnoreTypes(apis.Condition{}.Message, apis.Condition{}.LastTransitionTime)); d != "" {
					t.Errorf("TaskRun %s does not have expected status condition. Diff %s", actualTaskRun.Name, diff.PrintWantGot(d))
				}

				if tc.extraTaskRunChecks != nil {
					tc.extraTaskRunChecks(t, expectedTaskRun, &actualTaskRun)
				}

				taskRunStatusInTaskLoopRun, exists := status.TaskRuns[actualTaskRun.Name]
				if !exists {
					t.Errorf("Run status does not include TaskRun status for TaskRun %s", actualTaskRun.Name)
				} else {
					if d := cmp.Diff(expectedTaskRun.Status.Status.Conditions, taskRunStatusInTaskLoopRun.Status.Status.Conditions,
						cmpopts.IgnoreTypes(apis.Condition{}.Message, apis.Condition{}.LastTransitionTime)); d != "" {
						t.Errorf("Run status for TaskRun %s does not have expected status condition. Diff %s",
							actualTaskRun.Name, diff.PrintWantGot(d))
					}
					if i+1 != taskRunStatusInTaskLoopRun.Iteration {
						t.Errorf("Run status for TaskRun %s has iteration number %d instead of %d",
							actualTaskRun.Name, taskRunStatusInTaskLoopRun.Iteration, i+1)
					}
				}
			}

			t.Logf("Checking events that were created from Run")
			matchKinds := map[string][]string{"Run": {run.Name}}
			events, err := collectMatchingEvents(c.KubeClient, namespace, matchKinds)
			if err != nil {
				t.Fatalf("Failed to collect matching events: %q", err)
			}
			for e, expectedEvent := range tc.expectedEvents {
				if e >= len(events) {
					t.Errorf("Expected %d events but got %d", len(tc.expectedEvents), len(events))
					break
				}
				if matched, _ := regexp.MatchString(expectedEvent, events[e].Message); !matched {
					t.Errorf("Expected event %q but got %q", expectedEvent, events[e].Message)
				}
			}
		})
	}
}

func TestCancelTaskLoopRun(t *testing.T) {
	t.Run("cancel", func(t *testing.T) {
		c, namespace := setup(t)
		taskLoopClient := getTaskLoopClient(t, namespace)
		t.Parallel()

		knativetest.CleanupOnInterrupt(func() { tearDown(t, c, namespace) }, t.Logf)
		defer tearDown(t, c, namespace)

		t.Logf("Creating TaskLoop in namespace %s", namespace)
		if _, err := taskLoopClient.Create(sleepyTaskLoop); err != nil {
			t.Fatalf("Failed to create TaskLoop `%s`: %s", sleepyTaskLoop.Name, err)
		}

		t.Logf("Creating Run in namespace %s", namespace)
		run := runSleepyTaskLoop
		if _, err := c.RunClient.Create(run); err != nil {
			t.Fatalf("Failed to create Run `%s`: %s", run.Name, err)
		}

		t.Logf("Waiting for Run %s in namespace %s to be started", run.Name, namespace)
		if err := WaitForRunState(c, run.Name, runTimeout, Running(run.Name), "RunRunning"); err != nil {
			t.Fatalf("Error waiting for Run %s to be running: %s", run.Name, err)
		}

		// The current looping behavior is to run a single TaskRun at a time but the following code is generalized
		// to allow multiple TaskRuns in case that is added.
		taskrunList, err := c.TaskRunClient.List(metav1.ListOptions{LabelSelector: "tekton.dev/run=" + run.Name})
		if err != nil {
			t.Fatalf("Error listing TaskRuns for Run %s: %s", run.Name, err)
		}

		var wg sync.WaitGroup
		t.Logf("Waiting for TaskRuns from Run %s in namespace %s to be running", run.Name, namespace)
		for _, taskrunItem := range taskrunList.Items {
			wg.Add(1)
			go func(name string) {
				defer wg.Done()
				err := WaitForTaskRunState(c, name, Running(name), "TaskRunRunning")
				if err != nil {
					t.Errorf("Error waiting for TaskRun %s to be running: %v", name, err)
				}
			}(taskrunItem.Name)
		}
		wg.Wait()

		pr, err := c.RunClient.Get(run.Name, metav1.GetOptions{})
		if err != nil {
			t.Fatalf("Failed to get Run `%s`: %s", run.Name, err)
		}

		patches := []jsonpatch.JsonPatchOperation{{
			Operation: "add",
			Path:      "/spec/status",
			Value:     v1alpha1.RunSpecStatusCancelled,
		}}
		patchBytes, err := json.Marshal(patches)
		if err != nil {
			t.Fatalf("failed to marshal patch bytes in order to cancel")
		}
		if _, err := c.RunClient.Patch(pr.Name, types.JSONPatchType, patchBytes, ""); err != nil {
			t.Fatalf("Failed to patch Run `%s` with cancellation: %s", run.Name, err)
		}

		t.Logf("Waiting for Run %s in namespace %s to be cancelled", run.Name, namespace)
		if err := WaitForRunState(c, run.Name, runTimeout,
			FailedWithReason(taskloopv1alpha1.TaskLoopRunReasonCancelled.String(), run.Name), "RunCancelled"); err != nil {
			t.Errorf("Error waiting for Run %q to finished: %s", run.Name, err)
		}

		t.Logf("Waiting for TaskRuns in Run %s in namespace %s to be cancelled", run.Name, namespace)
		for _, taskrunItem := range taskrunList.Items {
			wg.Add(1)
			go func(name string) {
				defer wg.Done()
				err := WaitForTaskRunState(c, name, FailedWithReason("TaskRunCancelled", name), "TaskRunCancelled")
				if err != nil {
					t.Errorf("Error waiting for TaskRun %s to be finished: %v", name, err)
				}
			}(taskrunItem.Name)
		}
		wg.Wait()
	})
}

func getTaskLoopClient(t *testing.T, namespace string) resourceversioned.TaskLoopInterface {
	configPath := knativetest.Flags.Kubeconfig
	clusterName := knativetest.Flags.Cluster
	cfg, err := knativetest.BuildClientConfig(configPath, clusterName)
	if err != nil {
		t.Fatalf("failed to create configuration obj from %s for cluster %s: %s", configPath, clusterName, err)
	}
	cs, err := versioned.NewForConfig(cfg)
	if err != nil {
		t.Fatalf("failed to create taskloop clientset from config file at %s: %s", configPath, err)
	}
	return cs.CustomV1alpha1().TaskLoops(namespace)
}

func getExpectedTaskRunForInlineTask(expectedTaskRun *v1beta1.TaskRun) *v1beta1.TaskRun {
	// Change expected TaskRun for inline task case.  The only difference is there is a task spec instead of a task reference.
	expectedTaskRun = expectedTaskRun.DeepCopy()
	expectedTaskRun.Spec.TaskRef = nil
	expectedTaskRun.Spec.TaskSpec = &commonTaskSpec
	return expectedTaskRun
}

func getExpectedTaskRunForClusterTask(expectedTaskRun *v1beta1.TaskRun) *v1beta1.TaskRun {
	// Change expected TaskRun for cluster task case.  The only difference is the task reference is to a ClusterTask rather than a Task.
	expectedTaskRun = expectedTaskRun.DeepCopy()
	expectedTaskRun.Spec.TaskRef = &v1beta1.TaskRef{Name: clusterTaskName, Kind: "ClusterTask"}
	return expectedTaskRun
}

func getExpectedTaskRunAnnotations(taskloop *taskloopv1alpha1.TaskLoop, run *v1alpha1.Run) map[string]string {
	annotations := make(map[string]string, len(taskloop.ObjectMeta.Annotations)+len(run.ObjectMeta.Annotations))
	for key, value := range taskloop.ObjectMeta.Labels {
		run.ObjectMeta.Labels[key] = value
	}
	for key, val := range run.ObjectMeta.Annotations {
		annotations[key] = val
	}
	return annotations
}

func getExpectedTaskRunLabels(task *v1beta1.Task, clustertask *v1beta1.ClusterTask, taskloop *taskloopv1alpha1.TaskLoop, run *v1alpha1.Run, iteration int) map[string]string {
	labels := map[string]string{
		"app.kubernetes.io/managed-by":        "tekton-pipelines",
		"tekton.dev/run":                      run.Name,
		"custom.tekton.dev/taskLoop":          taskloop.Name,
		"custom.tekton.dev/taskLoopIteration": strconv.Itoa(iteration),
	}
	if task != nil {
		labels["tekton.dev/task"] = task.Name
	} else if clustertask != nil {
		labels["tekton.dev/task"] = clustertask.Name
		labels["tekton.dev/clusterTask"] = clustertask.Name
	}
	for key, value := range taskloop.ObjectMeta.Labels {
		labels[key] = value
	}
	for key, value := range run.ObjectMeta.Labels {
		labels[key] = value
	}
	return labels
}

func getTaskLoopWithRetries(taskloop *taskloopv1alpha1.TaskLoop) *taskloopv1alpha1.TaskLoop {
	taskloop = taskloop.DeepCopy()
	taskloop.Spec.Retries = numRetries
	return taskloop
}

func getTaskLoopWithTimeout(taskloop *taskloopv1alpha1.TaskLoop) *taskloopv1alpha1.TaskLoop {
	taskloop = taskloop.DeepCopy()
	taskloop.Spec.Timeout = taskTimeout
	return taskloop
}

func checkTaskRunRetries(t *testing.T, expectedTaskRun *v1beta1.TaskRun, actualTaskRun *v1beta1.TaskRun) {
	if len(actualTaskRun.Status.RetriesStatus) != numRetries {
		t.Errorf("Expected TaskRun %s to be retried %d times but it was retried %d times",
			actualTaskRun.Name, numRetries, len(actualTaskRun.Status.RetriesStatus))
	}
}

// collectMatchingEvents collects a list of events under 5 seconds that match certain objects by kind and name.
// This is copied from pipelinerun_test and modified to drop the reason parameter.
func collectMatchingEvents(kubeClient *knativetest.KubeClient, namespace string, kinds map[string][]string) ([]*corev1.Event, error) {
	var events []*corev1.Event

	watchEvents, err := kubeClient.Kube.CoreV1().Events(namespace).Watch(metav1.ListOptions{})
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

// deleteClusterTask removes a single clustertask by name using provided
// clientset. Test state is used for logging. deleteClusterTask does not wait
// for the clustertask to be deleted, so it is still possible to have name
// conflicts during test
func deleteClusterTask(t *testing.T, c *clients, name string) {
	t.Logf("Deleting clustertask %s", name)
	if err := c.ClusterTaskClient.Delete(name, &metav1.DeleteOptions{}); err != nil {
		t.Fatalf("Failed to delete clustertask: %v", err)
	}
}
