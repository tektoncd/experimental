//go:build e2e
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
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
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
	concurrencyLimit2         = 2
	noConcurrencyLimit        = 0
	numRetries                = 2 // number of task retries to test
	runTimeout                = 10 * time.Minute
	startedEventMessage       = ""        // Run started event has no message
	defaultServiceAccountName = "default" // default service account name
	defaultTaskRunTimeout     = &metav1.Duration{Duration: 1 * time.Hour}
	shortTaskRunTimeout       = &metav1.Duration{Duration: 10 * time.Second}
	ignoreReleaseAnnotation   = func(k string, v string) bool {
		return k == pod.ReleaseAnnotation
	}
)

// Expected events can be required or optional.
type ev struct {
	message  string
	required bool
}

// commonTaskSpec is reused in Task, Cluster Task, and inline task.
// This task is used to test TaskRun success and failure based on string comparison (current-item = fail-on-item).
// It is also used to test timeouts and cancellation by supporting a sleep parameter to control execution time.
var commonTaskSpec = v1beta1.TaskSpec{
	Params: []v1beta1.ParamSpec{{
		Name: "current-item",
		Type: v1beta1.ParamTypeString,
	}, {
		Name:    "fail-on-item",
		Type:    v1beta1.ParamTypeString,
		Default: &v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: ""},
	}, {
		Name:    "sleep-time",
		Type:    v1beta1.ParamTypeString,
		Default: &v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "0"},
	}},
	Steps: []v1beta1.Step{{
		Container: corev1.Container{
			Name:    "passfail",
			Image:   "ubuntu",
			Command: []string{"/bin/bash"},
			Args:    []string{"-c", `sleep $(params.sleep-time) && [[ "$(params.fail-on-item)" == "" || "$(params.current-item)" != "$(params.fail-on-item)" ]]`},
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
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeArray, ArrayVal: []string{"item1", "item2", "item3"}},
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
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeArray, ArrayVal: []string{"item1", "item2", "item3"}},
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
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeArray, ArrayVal: []string{"item1", "item2", "item3"}},
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
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeArray, ArrayVal: []string{"item1", "item2", "item3"}},
		}},
		Ref: &v1alpha1.TaskRef{
			APIVersion: taskloopv1alpha1.SchemeGroupVersion.String(),
			Kind:       taskloop.TaskLoopControllerName,
			Name:       "a-taskloop-using-a-cluster-task",
		},
	},
}

// This Run is used in the concurrency tests.  The tasks need to execute a short amount of time to make it
// unlikely that one could complete before the controller submits all of the TaskRuns.  If that happened, it would
// appear as if the desired concurrency wasn't obtained.  See checkConcurrency() for details.
var runTaskLoopWithShortSleep = &v1alpha1.Run{
	ObjectMeta: metav1.ObjectMeta{Name: "run-taskloop"},
	Spec: v1alpha1.RunSpec{
		Params: []v1beta1.Param{{
			Name:  "current-item",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeArray, ArrayVal: []string{"item1", "item2", "item3"}},
		}, {
			Name:  "sleep-time",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "10"},
		}},
		Ref: &v1alpha1.TaskRef{
			APIVersion: taskloopv1alpha1.SchemeGroupVersion.String(),
			Kind:       taskloop.TaskLoopControllerName,
			Name:       "a-taskloop",
		},
	},
}

// This Run is used in timeout and cancellation tests where we need the task to hang around a while.
var runTaskLoopWithLongSleep = &v1alpha1.Run{
	ObjectMeta: metav1.ObjectMeta{Name: "run-taskloop"},
	Spec: v1alpha1.RunSpec{
		Params: []v1beta1.Param{{
			Name:  "current-item",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeArray, ArrayVal: []string{"item1", "item2", "item3"}},
		}, {
			Name:  "sleep-time",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "120"},
		}},
		Ref: &v1alpha1.TaskRef{
			APIVersion: taskloopv1alpha1.SchemeGroupVersion.String(),
			Kind:       taskloop.TaskLoopControllerName,
			Name:       "a-taskloop",
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
		ServiceAccountName: defaultServiceAccountName,
		TaskRef:            &v1beta1.TaskRef{Name: "a-task", Kind: "Task"},
		Timeout:            defaultTaskRunTimeout,
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
		ServiceAccountName: defaultServiceAccountName,
		TaskRef:            &v1beta1.TaskRef{Name: "a-task", Kind: "Task"},
		Timeout:            defaultTaskRunTimeout,
		Params: []v1beta1.Param{{
			Name:  "current-item",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "item2"},
		}},
	},
	Status: v1beta1.TaskRunStatus{
		Status: taskRunStatusSuccess,
	},
}

var expectedTaskRunIteration3Success = &v1beta1.TaskRun{
	ObjectMeta: metav1.ObjectMeta{
		Name: "run-taskloop-00003-", // does not include random suffix
		// Expected labels and annotations are added dynamically
	},
	Spec: v1beta1.TaskRunSpec{
		ServiceAccountName: defaultServiceAccountName,
		TaskRef:            &v1beta1.TaskRef{Name: "a-task", Kind: "Task"},
		Timeout:            defaultTaskRunTimeout,
		Params: []v1beta1.Param{{
			Name:  "current-item",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "item3"},
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
		ServiceAccountName: defaultServiceAccountName,
		TaskRef:            &v1beta1.TaskRef{Name: "a-task", Kind: "Task"},
		Timeout:            defaultTaskRunTimeout,
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

var expectedTaskRunShortSleepIteration1Success = &v1beta1.TaskRun{
	ObjectMeta: metav1.ObjectMeta{
		Name: "run-taskloop-00001-", // does not include random suffix
		// Expected labels and annotations are added dynamically
	},
	Spec: v1beta1.TaskRunSpec{
		ServiceAccountName: defaultServiceAccountName,
		TaskRef:            &v1beta1.TaskRef{Name: "a-task", Kind: "Task"},
		Timeout:            defaultTaskRunTimeout,
		Params: []v1beta1.Param{{
			Name:  "current-item",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "item1"},
		}, {
			Name:  "sleep-time",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "10"},
		}},
	},
	Status: v1beta1.TaskRunStatus{
		Status: taskRunStatusSuccess,
	},
}

var expectedTaskRunShortSleepIteration2Success = &v1beta1.TaskRun{
	ObjectMeta: metav1.ObjectMeta{
		Name: "run-taskloop-00002-", // does not include random suffix
		// Expected labels and annotations are added dynamically
	},
	Spec: v1beta1.TaskRunSpec{
		ServiceAccountName: defaultServiceAccountName,
		TaskRef:            &v1beta1.TaskRef{Name: "a-task", Kind: "Task"},
		Timeout:            defaultTaskRunTimeout,
		Params: []v1beta1.Param{{
			Name:  "current-item",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "item2"},
		}, {
			Name:  "sleep-time",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "10"},
		}},
	},
	Status: v1beta1.TaskRunStatus{
		Status: taskRunStatusSuccess,
	},
}

var expectedTaskRunShortSleepIteration3Success = &v1beta1.TaskRun{
	ObjectMeta: metav1.ObjectMeta{
		Name: "run-taskloop-00003-", // does not include random suffix
		// Expected labels and annotations are added dynamically
	},
	Spec: v1beta1.TaskRunSpec{
		ServiceAccountName: defaultServiceAccountName,
		TaskRef:            &v1beta1.TaskRef{Name: "a-task", Kind: "Task"},
		Timeout:            defaultTaskRunTimeout,
		Params: []v1beta1.Param{{
			Name:  "current-item",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "item3"},
		}, {
			Name:  "sleep-time",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "10"},
		}},
	},
	Status: v1beta1.TaskRunStatus{
		Status: taskRunStatusSuccess,
	},
}

var expectedTaskRunLongSleepIteration1Timeout = &v1beta1.TaskRun{
	ObjectMeta: metav1.ObjectMeta{
		Name: "run-taskloop-00001-", // does not include random suffix
		// Expected labels and annotations are added dynamically
	},
	Spec: v1beta1.TaskRunSpec{
		ServiceAccountName: defaultServiceAccountName,
		Timeout:            shortTaskRunTimeout,
		TaskRef:            &v1beta1.TaskRef{Name: "a-task", Kind: "Task"},
		Params: []v1beta1.Param{{
			Name:  "current-item",
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "item1"},
		}, {
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
		expectedStatus      corev1.ConditionStatus
		expectedReason      taskloopv1alpha1.TaskLoopRunReason
		expectedTaskRuns    []*v1beta1.TaskRun
		expectedConcurrency *int // default is sequential
		expectedEvents      []ev
		// This function can perform additional checks on the TaskRun.  It is passed the expected and actual TaskRuns.
		extraTaskRunChecks func(*testing.T, *v1beta1.TaskRun, *v1beta1.TaskRun)
	}{{
		name:           "successful TaskLoop",
		task:           aTask,
		taskloop:       aTaskLoop,
		run:            runTaskLoopSuccess,
		expectedStatus: corev1.ConditionTrue,
		expectedReason: taskloopv1alpha1.TaskLoopRunReasonSucceeded,
		expectedTaskRuns: []*v1beta1.TaskRun{
			expectedTaskRunIteration1Success,
			expectedTaskRunIteration2Success,
			expectedTaskRunIteration3Success},
		expectedEvents: []ev{
			ev{startedEventMessage, true},
			ev{"Iterations completed: 0", true},
			ev{"Iterations completed: 1", true},
			ev{"Iterations completed: 2", true},
			ev{"All TaskRuns completed successfully", true}},
	}, {
		name:             "failed TaskLoop",
		task:             aTask,
		taskloop:         aTaskLoop,
		run:              runTaskLoopFailure,
		expectedStatus:   corev1.ConditionFalse,
		expectedReason:   taskloopv1alpha1.TaskLoopRunReasonFailed,
		expectedTaskRuns: []*v1beta1.TaskRun{expectedTaskRunIteration1Failure},
		expectedEvents: []ev{
			ev{startedEventMessage, true},
			ev{"Iterations completed: 0", true},
			ev{"One or more TaskRuns have failed", true}},
	}, {
		name:             "failed TaskLoop with retries",
		task:             aTask,
		taskloop:         getTaskLoopWithRetries(aTaskLoop),
		run:              runTaskLoopFailure,
		expectedStatus:   corev1.ConditionFalse,
		expectedReason:   taskloopv1alpha1.TaskLoopRunReasonFailed,
		expectedTaskRuns: []*v1beta1.TaskRun{expectedTaskRunIteration1Failure},
		expectedEvents: []ev{
			ev{startedEventMessage, true},
			ev{"Iterations completed: 0", true},
			ev{"One or more TaskRuns have failed", true}},
		extraTaskRunChecks: checkTaskRunRetries,
	}, {
		name:             "failed TaskLoop due to taskrun timeout",
		task:             aTask,
		taskloop:         getTaskLoopWithTimeout(aTaskLoop),
		run:              runTaskLoopWithLongSleep,
		expectedStatus:   corev1.ConditionFalse,
		expectedReason:   taskloopv1alpha1.TaskLoopRunReasonFailed,
		expectedTaskRuns: []*v1beta1.TaskRun{expectedTaskRunLongSleepIteration1Timeout},
		expectedEvents: []ev{
			ev{startedEventMessage, true},
			ev{"Iterations completed: 0", true},
			ev{"One or more TaskRuns have failed", true}},
	}, {
		name:           "successful TaskLoop using an inline task",
		taskloop:       aTaskLoopUsingAnInlineTask,
		run:            runTaskLoopUsingAnInlineTaskSuccess,
		expectedStatus: corev1.ConditionTrue,
		expectedReason: taskloopv1alpha1.TaskLoopRunReasonSucceeded,
		expectedTaskRuns: []*v1beta1.TaskRun{
			getExpectedTaskRunForInlineTask(expectedTaskRunIteration1Success),
			getExpectedTaskRunForInlineTask(expectedTaskRunIteration2Success),
			getExpectedTaskRunForInlineTask(expectedTaskRunIteration3Success),
		},
		expectedEvents: []ev{
			ev{startedEventMessage, true},
			ev{"Iterations completed: 0", true},
			ev{"Iterations completed: 1", true},
			ev{"Iterations completed: 2", true},
			ev{"All TaskRuns completed successfully", true}},
	}, {
		name:           "successful TaskLoop with concurrency limit",
		task:           aTask,
		taskloop:       getTaskLoopWithConcurrency(aTaskLoop, concurrencyLimit2),
		run:            runTaskLoopWithShortSleep,
		expectedStatus: corev1.ConditionTrue,
		expectedReason: taskloopv1alpha1.TaskLoopRunReasonSucceeded,
		expectedTaskRuns: []*v1beta1.TaskRun{
			expectedTaskRunShortSleepIteration1Success,
			expectedTaskRunShortSleepIteration2Success,
			expectedTaskRunShortSleepIteration3Success},
		expectedConcurrency: &concurrencyLimit2,
		expectedEvents: []ev{
			ev{startedEventMessage, true},
			ev{"Iterations completed: 0", true},
			ev{"Iterations completed: 1", false}, // Optional event depending on timing
			ev{"Iterations completed: 2", false}, // Optional event depending on timing
			ev{"All TaskRuns completed successfully", true}},
	}, {
		name:           "successful TaskLoop with unlimited concurrency",
		task:           aTask,
		taskloop:       getTaskLoopWithConcurrency(aTaskLoop, noConcurrencyLimit),
		run:            runTaskLoopWithShortSleep,
		expectedStatus: corev1.ConditionTrue,
		expectedReason: taskloopv1alpha1.TaskLoopRunReasonSucceeded,
		expectedTaskRuns: []*v1beta1.TaskRun{
			expectedTaskRunShortSleepIteration1Success,
			expectedTaskRunShortSleepIteration2Success,
			expectedTaskRunShortSleepIteration3Success},
		expectedConcurrency: &noConcurrencyLimit,
		expectedEvents: []ev{
			ev{startedEventMessage, true},
			ev{"Iterations completed: 0", true},
			ev{"Iterations completed: 1", false}, // Optional event depending on timing
			ev{"Iterations completed: 2", false}, // Optional event depending on timing
			ev{"All TaskRuns completed successfully", true}},
	}}
	// TODO: TaskLoop tests using a ClusterTask

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			tc := tc // Copy current tc to local variable due to test parallelization
			t.Parallel()
			ctx := context.Background()
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()
			c, namespace := setup(ctx, t)
			taskLoopClient := getTaskLoopClient(t, namespace)

			knativetest.CleanupOnInterrupt(func() { tearDown(ctx, t, c, namespace) }, t.Logf)
			defer tearDown(ctx, t, c, namespace)

			if tc.task != nil {
				task := tc.task.DeepCopy()
				task.Namespace = namespace
				if _, err := c.TaskClient.Create(ctx, task, metav1.CreateOptions{}); err != nil {
					t.Fatalf("Failed to create Task `%s`: %s", task.Name, err)
				}
			}

			if tc.clustertask != nil {
				if _, err := c.ClusterTaskClient.Create(ctx, tc.clustertask, metav1.CreateOptions{}); err != nil {
					t.Fatalf("Failed to create ClusterTask `%s`: %s", tc.clustertask.Name, err)
				}
				knativetest.CleanupOnInterrupt(func() { deleteClusterTask(ctx, t, c, tc.clustertask.Name) }, t.Logf)
				defer deleteClusterTask(ctx, t, c, tc.clustertask.Name)
			}

			if tc.taskloop != nil {
				taskloop := tc.taskloop.DeepCopy()
				taskloop.Namespace = namespace
				if _, err := taskLoopClient.Create(ctx, taskloop, metav1.CreateOptions{}); err != nil {
					t.Fatalf("Failed to create TaskLoop `%s`: %s", tc.taskloop.Name, err)
				}
			}

			run := tc.run.DeepCopy()
			run.Namespace = namespace
			run, err := c.RunClient.Create(ctx, run, metav1.CreateOptions{})
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
			if err := WaitForRunState(ctx, c, run.Name, runTimeout, inState, desc); err != nil {
				t.Fatalf("Error waiting for Run %s/%s to finish: %s", run.Namespace, run.Name, err)
			}

			run, err = c.RunClient.Get(ctx, run.Name, metav1.GetOptions{})
			if err != nil {
				t.Fatalf("Couldn't get expected Run %s/%s: %s", run.Namespace, run.Name, err)
			}

			t.Logf("Making sure the expected TaskRuns were created")
			actualTaskRunList, err := c.TaskRunClient.List(ctx, metav1.ListOptions{LabelSelector: fmt.Sprintf("tekton.dev/run=%s", run.Name)})
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
					cmpopts.IgnoreFields(apis.Condition{}, "Message", "LastTransitionTime")); d != "" {
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
						cmpopts.IgnoreFields(apis.Condition{}, "Message", "LastTransitionTime")); d != "" {
						t.Errorf("Run status for TaskRun %s does not have expected status condition. Diff %s",
							actualTaskRun.Name, diff.PrintWantGot(d))
					}
					if i+1 != taskRunStatusInTaskLoopRun.Iteration {
						t.Errorf("Run status for TaskRun %s has iteration number %d instead of %d",
							actualTaskRun.Name, taskRunStatusInTaskLoopRun.Iteration, i+1)
					}
				}
			}

			// Check for concurrency limit violation.
			t.Logf("Checking for concurrency limit violation")
			checkConcurrency(t, tc.expectedConcurrency, run, tc.expectedTaskRuns, actualTaskRunList)

			t.Logf("Checking events that were created from Run")
			checkEvents(ctx, t, c, run, namespace, tc.expectedEvents)
		})
	}
}

func TestCancelTaskLoopRun(t *testing.T) {
	t.Run("cancel", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		c, namespace := setup(ctx, t)
		taskLoopClient := getTaskLoopClient(t, namespace)

		knativetest.CleanupOnInterrupt(func() { tearDown(ctx, t, c, namespace) }, t.Logf)
		defer tearDown(ctx, t, c, namespace)

		t.Logf("Creating Task %s in namespace %s", aTask.Name, namespace)
		if _, err := c.TaskClient.Create(ctx, aTask, metav1.CreateOptions{}); err != nil {
			t.Fatalf("Failed to create TaskLoop `%s`: %s", aTaskLoop.Name, err)
		}

		t.Logf("Creating TaskLoop %s in namespace %s", aTaskLoop.Name, namespace)
		if _, err := taskLoopClient.Create(ctx, getTaskLoopWithConcurrency(aTaskLoop, concurrencyLimit2), metav1.CreateOptions{}); err != nil {
			t.Fatalf("Failed to create TaskLoop `%s`: %s", aTaskLoop.Name, err)
		}

		run := runTaskLoopWithLongSleep
		t.Logf("Creating Run %s in namespace %s", run.Name, namespace)
		if _, err := c.RunClient.Create(ctx, run, metav1.CreateOptions{}); err != nil {
			t.Fatalf("Failed to create Run `%s`: %s", run.Name, err)
		}

		t.Logf("Waiting for Run %s in namespace %s to be started", run.Name, namespace)
		if err := WaitForRunState(ctx, c, run.Name, runTimeout, Running(run.Name), "RunRunning"); err != nil {
			t.Fatalf("Error waiting for Run %s to be running: %s", run.Name, err)
		}

		taskrunList, err := c.TaskRunClient.List(ctx, metav1.ListOptions{LabelSelector: "tekton.dev/run=" + run.Name})
		if err != nil {
			t.Fatalf("Error listing TaskRuns for Run %s: %s", run.Name, err)
		}

		var wg sync.WaitGroup
		for _, taskrunItem := range taskrunList.Items {
			t.Logf("Waiting for TaskRun %s from Run %s in namespace %s to be running", taskrunItem.Name, run.Name, namespace)
			wg.Add(1)
			go func(name string) {
				defer wg.Done()
				err := WaitForTaskRunState(ctx, c, name, Running(name), "TaskRunRunning")
				if err != nil {
					t.Errorf("Error waiting for TaskRun %s to be running: %v", name, err)
				}
			}(taskrunItem.Name)
		}
		wg.Wait()

		pr, err := c.RunClient.Get(ctx, run.Name, metav1.GetOptions{})
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
		if _, err := c.RunClient.Patch(ctx, pr.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{}); err != nil {
			t.Fatalf("Failed to patch Run `%s` with cancellation: %s", run.Name, err)
		}

		t.Logf("Waiting for Run %s in namespace %s to be cancelled", run.Name, namespace)
		if err := WaitForRunState(ctx, c, run.Name, runTimeout,
			FailedWithReason(v1alpha1.RunReasonCancelled, run.Name), "RunCancelled"); err != nil {
			t.Errorf("Error waiting for Run %q to finished: %s", run.Name, err)
		}

		for _, taskrunItem := range taskrunList.Items {
			t.Logf("Waiting for TaskRun %s in Run %s in namespace %s to be cancelled", taskrunItem.Name, run.Name, namespace)
			wg.Add(1)
			go func(name string) {
				defer wg.Done()
				err := WaitForTaskRunState(ctx, c, name, FailedWithReason("TaskRunCancelled", name), "TaskRunCancelled")
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

func getTaskLoopWithConcurrency(taskloop *taskloopv1alpha1.TaskLoop, concurrency int) *taskloopv1alpha1.TaskLoop {
	taskloop = taskloop.DeepCopy()
	taskloop.Spec.Concurrency = &concurrency
	return taskloop
}

func getTaskLoopWithRetries(taskloop *taskloopv1alpha1.TaskLoop) *taskloopv1alpha1.TaskLoop {
	taskloop = taskloop.DeepCopy()
	taskloop.Spec.Retries = numRetries
	return taskloop
}

func getTaskLoopWithTimeout(taskloop *taskloopv1alpha1.TaskLoop) *taskloopv1alpha1.TaskLoop {
	taskloop = taskloop.DeepCopy()
	taskloop.Spec.Timeout = shortTaskRunTimeout
	return taskloop
}

func checkTaskRunRetries(t *testing.T, expectedTaskRun *v1beta1.TaskRun, actualTaskRun *v1beta1.TaskRun) {
	if len(actualTaskRun.Status.RetriesStatus) != numRetries {
		t.Errorf("Expected TaskRun %s to be retried %d times but it was retried %d times",
			actualTaskRun.Name, numRetries, len(actualTaskRun.Status.RetriesStatus))
	}
}

// checkConcurrency collects and sorts the creation and completion times of each TaskRun.
// It then processes the times in sequence to determine the maximum number of TaskRuns were alive at a time.
func checkConcurrency(t *testing.T, expectedConcurrency *int, run *v1alpha1.Run,
	expectedTaskRuns []*v1beta1.TaskRun, actualTaskRunList *v1beta1.TaskRunList) {

	type taskRunEventType int
	const (
		trCreated   taskRunEventType = iota
		trCompleted taskRunEventType = iota
	)

	type taskRunEvent struct {
		eventTime metav1.Time
		eventType taskRunEventType
		trName    string
	}

	events := []taskRunEvent{}
	for _, actualTaskRun := range actualTaskRunList.Items {
		// This shouldn't happen unless something really went wrong but we need to test it to avoid a panic dereferencing the pointer.
		if actualTaskRun.Status.CompletionTime == nil {
			t.Errorf("TaskRun %s does not have a completion time", actualTaskRun.Name)
			continue
		}
		events = append(events,
			taskRunEvent{eventTime: actualTaskRun.ObjectMeta.CreationTimestamp, eventType: trCreated, trName: actualTaskRun.ObjectMeta.Name},
			taskRunEvent{eventTime: *actualTaskRun.Status.CompletionTime, eventType: trCompleted, trName: actualTaskRun.ObjectMeta.Name},
		)
	}

	sort.Slice(events, func(i, j int) bool {
		// Unfortunately the timestamp resolution is only 1 second which makes event ordering imprecise.
		// This could trigger a false limit failure if TaskRun B was created a few milliseconds after TaskRun A completed
		// but the events were sorted the other way.  In order to address this, the sort places TaskRun completion
		// before TaskRun creation when the times are equal and the TaskRun names are different.  There is a small
		// chance this could mask a problem but it's the best that can be done with the limited timestamp resolution.
		return events[i].eventTime.Before(&events[j].eventTime) ||
			(events[i].eventTime.Equal(&events[j].eventTime) &&
				((events[i].trName == events[j].trName && events[i].eventType == trCreated) ||
					(events[i].trName != events[j].trName && events[i].eventType == trCompleted)))
	})

	t.Logf("Sorted taskrun event table: %v", events)

	// Determine how many TaskRuns were "alive" at any given time, where "alive" means the TaskRun was created
	// and thus eligible to run and not yet completed.
	concurrency := 0
	maxConcurrency := 0
	concurrencyLimit := 1
	if expectedConcurrency != nil {
		concurrencyLimit = *expectedConcurrency
	}
	offender := ""
	var firstCreationTime, lastCompletionTime metav1.Time
	firstCreationTime = metav1.Unix(1<<63-62135596801, 999999999) // max time
	for _, event := range events {
		if event.eventType == trCreated {
			concurrency++
		} else {
			concurrency--
		}
		// If the concurrency limit was breached, record the first TaskRun where it happened.
		if concurrencyLimit > 0 && concurrency > concurrencyLimit && offender == "" {
			offender = event.trName
		}
		// Track the peak number of active TaskRuns.
		if concurrency > maxConcurrency {
			maxConcurrency = concurrency
		}
		if event.eventType == trCreated {
			if event.eventTime.Before(&firstCreationTime) {
				firstCreationTime = event.eventTime
			}
		} else {
			if lastCompletionTime.Before(&event.eventTime) {
				lastCompletionTime = event.eventTime
			}
		}
	}

	t.Logf("maxConcurrency=%v", maxConcurrency)

	if concurrencyLimit <= 0 {
		// There is no limit so all of the expected TaskRuns should have been created at once.
		// This check assumes that the controller can create all of the TaskRuns before any of them complete.
		// It would take a very fast TaskRun and a very slow controller to violate that but using a sleep in
		// the task helps to make that unlikely.
		if maxConcurrency < len(expectedTaskRuns) {
			t.Errorf("Concurrency is unlimited so all %d expected TaskRuns should have been active at once but only %d were.",
				len(expectedTaskRuns), maxConcurrency)
		}
	} else {
		// There is a limit so there shouldn't be more TaskRuns than that alive at any moment.
		if maxConcurrency > concurrencyLimit {
			t.Errorf("Concurrency limit %d was broken. "+
				"%d TaskRuns were running or eligible to run at one point. "+
				"The limit was first crossed when TaskRun %s was created.",
				concurrencyLimit, maxConcurrency, offender)
		} else {
			// The limit should be equaled when TaskRuns are created for the first set of iterations,
			// unless there are fewer TaskRuns than the limit.
			expectedPeak := concurrencyLimit
			if len(expectedTaskRuns) < concurrencyLimit {
				expectedPeak = len(expectedTaskRuns)
			}
			if maxConcurrency < expectedPeak {
				t.Errorf("Concurrency limit %d was not reached. "+
					"At most only %d TaskRuns were running or eligible to run.",
					concurrencyLimit, maxConcurrency)
			}
		}
	}

	// Check that the Run's start and completion times are set appropriately.
	if run.Status.StartTime == nil {
		t.Errorf("The Run start time is not set!")
	} else if firstCreationTime.Before(run.Status.StartTime) {
		t.Errorf("The Run start time %v is after the first TaskRun's creation time %v", run.Status.StartTime, firstCreationTime)
	}
	if run.Status.CompletionTime == nil {
		t.Errorf("The Run completion time is not set!")
	} else if run.Status.CompletionTime.Before(&lastCompletionTime) {
		t.Errorf("The Run completion time %v is before the last TaskRun's completion time %v", run.Status.CompletionTime, lastCompletionTime)
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

// deleteClusterTask removes a single clustertask by name using provided
// clientset. Test state is used for logging. deleteClusterTask does not wait
// for the clustertask to be deleted, so it is still possible to have name
// conflicts during test
func deleteClusterTask(ctx context.Context, t *testing.T, c *clients, name string) {
	t.Logf("Deleting clustertask %s", name)
	if err := c.ClusterTaskClient.Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		t.Fatalf("Failed to delete clustertask: %v", err)
	}
}
