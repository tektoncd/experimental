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

package pipelinetotaskrun

import (
	"context"
	"fmt"
	"github.com/google/go-cmp/cmp/cmpopts"
	"io/ioutil"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/experimental/pipeline-to-taskrun/test"
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

func getController(t *testing.T, d test.Data) (test.Assets, func()) {
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

func checkRunCondition(t *testing.T, run *v1alpha1.Run, expectedStatus corev1.ConditionStatus, expectedReason string, expectedMessage string) error {
	failed := false
	condition := run.Status.GetCondition(apis.ConditionSucceeded)
	if condition == nil {
		t.Error("Condition missing in Run")
		failed = true
	} else {
		if condition.Status != expectedStatus {
			t.Errorf("Expected Run status to be %v but was %v", expectedStatus, condition)
			failed = true
		}
		if condition.Reason != expectedReason {
			t.Errorf("Expected reason to be %q but was %q", expectedReason, condition.Reason)
			failed = true
		}
		if condition.Message != expectedMessage {
			t.Errorf("Expected message to be %q but was %q", expectedMessage, condition.Message)
			failed = true
		}
	}
	if run.Status.StartTime == nil {
		t.Errorf("Expected Run start time to be set but it wasn't")
		failed = true
	}
	if expectedStatus == corev1.ConditionUnknown {
		if run.Status.CompletionTime != nil {
			t.Errorf("Expected Run completion time to not be set but it was")
			failed = true
		}
	} else if run.Status.CompletionTime == nil {
		t.Errorf("Expected Run completion time to be set but it wasn't")
		failed = true
	}
	if failed {
		return fmt.Errorf("run was invalid")
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

func getCreatedTaskRun(clients test.Clients) *v1beta1.TaskRun {
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
		Type:   apis.ConditionSucceeded,
		Status: corev1.ConditionTrue,
		Reason: v1beta1.TaskRunReasonSuccessful.String(),
	})
	return trWithStatus
}

func failed(tr *v1beta1.TaskRun) *v1beta1.TaskRun {
	trWithStatus := tr.DeepCopy()
	trWithStatus.Status.SetCondition(&apis.Condition{
		Type:   apis.ConditionSucceeded,
		Status: corev1.ConditionFalse,
		Reason: v1beta1.TaskRunReasonFailed.String(),
	})
	return trWithStatus
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
var tr = &v1beta1.TaskRun{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "run-with-pipeline",
		Namespace: "foo",
		Labels: map[string]string{
			"tekton.dev/run": "run-with-pipeline",
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
	Spec: v1beta1.TaskRunSpec{
		ServiceAccountName: "default",
		TaskSpec: &v1beta1.TaskSpec{
			Steps: []v1beta1.Step{{Container: corev1.Container{
				Image:   "ubuntu",
				Command: []string{"/bin/bash"},
				Args:    []string{"-c", "echo hello world"},
			}}},
		},
	},
}

var runWithPipeline = &v1alpha1.Run{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "run-with-pipeline",
		Namespace: "foo",
	},
	Spec: v1alpha1.RunSpec{
		Ref: &v1alpha1.TaskRef{
			APIVersion: "tekton.dev/v1alpha1",
			Kind:       "PipelineToTaskRun",
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
			APIVersion: "tekton.dev/v1alpha1",
			Kind:       "PipelineToTaskRun",
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
			APIVersion: "tekton.dev/v1alpha1",
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
			Kind: "PipelineToTaskRun",
			Name: "pipeline",
		},
	},
}

func TestReconcile(t *testing.T) {
	testcases := []struct {
		name            string
		pipeline        *v1beta1.Pipeline
		run             *v1alpha1.Run
		taskRun         *v1beta1.TaskRun
		expectedStatus  corev1.ConditionStatus
		expectedReason  v1beta1.TaskRunReason
		expectedMessage string
		expectedEvents  []string
		expectedTaskRun *v1beta1.TaskRun
	}{{
		name:            "Reconcile a new run that references a pipeline",
		pipeline:        p,
		run:             runWithPipeline,
		expectedTaskRun: tr,
		expectedStatus:  corev1.ConditionUnknown,
		expectedReason:  v1beta1.TaskRunReasonStarted,
		expectedEvents: []string{
			"Normal Started ",
		},
	}, {
		name:           "Reconcile a run with a running TaskRun",
		pipeline:       p,
		run:            runWithPipeline,
		taskRun:        running(tr),
		expectedStatus: corev1.ConditionUnknown,
		expectedReason: v1beta1.TaskRunReasonRunning,
		expectedEvents: []string{
			"Normal Started ",
			"Normal Running ",
		},
	}, {
		name:           "Reconcile a run with a failed PipelineRun",
		pipeline:       p,
		run:            runWithPipeline,
		taskRun:        failed(tr),
		expectedStatus: corev1.ConditionFalse,
		expectedReason: v1beta1.TaskRunReasonFailed,
		expectedEvents: []string{
			"Normal Started ",
			"Warning Failed ",
		},
	}, {
		name:           "Reconcile a run with a successful PipelineRun",
		pipeline:       p,
		run:            runWithPipeline,
		taskRun:        successful(tr),
		expectedStatus: corev1.ConditionTrue,
		expectedReason: v1beta1.TaskRunReasonSuccessful,
		expectedEvents: []string{
			"Normal Started ",
			"Normal Succeeded ",
		},
	},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			names.TestingSeed()

			optionalTaskRuns := []*v1beta1.TaskRun{tc.taskRun}
			if tc.taskRun == nil {
				optionalTaskRuns = nil
			}

			d := test.Data{
				Runs:      []*v1alpha1.Run{tc.run},
				Pipelines: []*v1beta1.Pipeline{tc.pipeline},
				TaskRuns:  optionalTaskRuns,
			}

			testAssets, _ := getController(t, d)

			if err := testAssets.Controller.Reconciler.Reconcile(ctx, getRunName(tc.run)); err != nil {
				t.Fatalf("couldn't reconcile run %v", err)
			}

			run, err := testAssets.Clients.Pipeline.TektonV1alpha1().Runs(tc.run.Namespace).Get(ctx, tc.run.Name, metav1.GetOptions{})
			if err != nil {
				t.Fatalf("Error getting reconciled run from fake client: %s", err)
			}

			createdTaskRun := getCreatedTaskRun(testAssets.Clients)
			if tc.expectedTaskRun != nil {
				if createdTaskRun == nil {
					t.Fatalf("A TaskRun should have been created but was not")
				} else {
					if d := cmp.Diff(tc.expectedTaskRun, createdTaskRun); d != "" {
						t.Errorf("Expected TaskRun was not created. Diff %s", diff.PrintWantGot(d))
					}
				}
			}

			if err := checkRunCondition(t, run, tc.expectedStatus, tc.expectedReason.String(), tc.expectedMessage); err != nil {
				t.Fatalf("run is invalid")
			}

			if err := checkEvents(testAssets.Recorder, tc.name, tc.expectedEvents); err != nil {
				t.Errorf(err.Error())
			}
		})
	}
}

func fromFile(t *testing.T, filename string) string {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Fatalf("couldn't load test data %s from file: %v", filename, err)
	}
	return string(b)
}

func createDataFromFiles(t *testing.T) test.Data {
	ns := "some-ns"
	taskFiles := []string{"testdata/gcs-upload.yaml", "testdata/git-clone.yaml", "testdata/go-test.yaml"}
	tasks := []*v1beta1.Task{}
	for i, taskFile := range taskFiles {
		tasks = append(tasks, test.MustParseTask(t, fromFile(t, taskFile)))
		tasks[i].Namespace = ns
	}
	examplePath := "../../../examples"
	pipeline := test.MustParsePipeline(t, fromFile(t, examplePath+"/clone-test-upload.yaml"))
	run := test.MustParseRun(t, fromFile(t, examplePath+"/pipeline-taskrun-run.yaml"))

	run.Name = "some-run"
	pipeline.Namespace = ns
	run.Namespace = ns

	return test.Data{
		Runs:      []*v1alpha1.Run{run},
		Pipelines: []*v1beta1.Pipeline{pipeline},
		Tasks:     tasks,
	}
}

func TestReconcileComplexPipeline(t *testing.T) {
	ctx := context.Background()
	names.TestingSeed()

	d := createDataFromFiles(t)
	run := d.Runs[0]

	testAssets, _ := getController(t, d)

	if err := testAssets.Controller.Reconciler.Reconcile(ctx, getRunName(run)); err != nil {
		t.Fatalf("couldn't reconcile run %v", err)
	}

	run, err := testAssets.Clients.Pipeline.TektonV1alpha1().Runs(run.Namespace).Get(ctx, run.Name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Error getting reconciled run from fake client: %s", err)
	}

	expectedTaskRun := test.MustParseTaskRun(t, fromFile(t, "testdata/expected-taskrun.yaml"))
	createdTaskRun := getCreatedTaskRun(testAssets.Clients)

	if createdTaskRun == nil {
		t.Fatalf("A TaskRun should have been created but was not")
	}

	if d := cmp.Diff(expectedTaskRun.ObjectMeta, createdTaskRun.ObjectMeta,
		// empty lists are defaulted differently between the yaml parsing and reconciling
		cmpopts.EquateEmpty()); d != "" {
		t.Errorf("TaskRun metadata was different from expected: %s", diff.PrintWantGot(d))
	}
	// string is the default type for params; the version loaded from yaml won't have this set explicitly
	ignoreString := cmpopts.IgnoreFields(v1beta1.ParamSpec{}, "Type")
	if d := cmp.Diff(expectedTaskRun.Spec, createdTaskRun.Spec, ignoreString); d != "" {
		t.Errorf("TaskRun spec was different from expected: %s", diff.PrintWantGot(d))
	}
	condition := run.Status.GetCondition(apis.ConditionSucceeded)
	if condition.Status != corev1.ConditionUnknown {
		t.Errorf("exepcted run to be marked executing but condition was %v", condition)
	}
}

func TestReconcileUnsupported(t *testing.T) {
	run := `
metadata:
  name: run-with-pipeline
  namespace: foo
spec:
  ref:
    apiVersion: tekton.dev/v1alpha1
    kind: PipelineToTaskRun
    name: pipeline
`
	testcases := []struct {
		name            string
		expectedErrText []string
		pipeline        *v1beta1.Pipeline
		run             *v1alpha1.Run
	}{{
		name:            "pipeline results - TODO (community#447)",
		expectedErrText: []string{"results"},
		pipeline: test.MustParsePipeline(t, `
metadata:
  name: pipeline
  namespace: foo
spec:
  tasks:
  - name: make-result
    taskSpec:
      steps:
      - image: ubuntu
      results:
      - name: amazing
  results:
  - name: amazing-result
    value: $(tasks.make-result.results.amazing)
`),
		run: test.MustParseRun(t, run),
	}, {
		name:            "array params - TODO (community#447)",
		expectedErrText: []string{"array"},
		pipeline: test.MustParsePipeline(t, `
metadata:
  name: pipeline
  namespace: foo
spec:
  tasks:
  - name: make-result
    taskSpec:
      params:
      - name: foo
        type: array
`),
		run: test.MustParseRun(t, run),
	}, {
		name:            "workspaces with subpaths  in run binding - TODO(community#447)",
		expectedErrText: []string{"spec.workspaces.subpath"},
		pipeline: test.MustParsePipeline(t, `
metadata:
  name: pipeline
  namespace: foo
spec:
  tasks:
  - name: use-workspace
    taskSpec:
      steps:
      - image: ubuntu
      workspaces:
      - name: task-workspace
    workspaces:
    - name: task-workspace
      workspace: pipeline-workspace
  workspaces:
  - name: pipeline-workspace
`),
		run: test.MustParseRun(t, run+`
  workspaces:
    - name: pipeline-workspace
      persistentVolumeClaim:
        claimName: pvc
      subPath: some/subdir
`),
	}, {
		name:            "workspaces with subpaths pipelinetasks - TODO(community#447)",
		expectedErrText: []string{"subpath"},
		pipeline: test.MustParsePipeline(t, `
metadata:
  name: pipeline
  namespace: foo
spec:
  tasks:
  - name: use-workspace
    taskSpec:
      steps:
      - image: ubuntu
      workspaces:
      - name: task-workspace
    workspaces:
    - name: task-workspace
      workspace: pipeline-workspace
      subPath: some/subdir
  workspaces:
  - name: pipeline-workspace
`),
		run: test.MustParseRun(t, run+`
  workspaces:
    - name: pipeline-workspace
      persistentVolumeClaim:
        claimName: pvc
`),
	}, {
		name:            "workspaces with mountpaths - TODO(community#447)",
		expectedErrText: []string{"mountPath"},
		pipeline: test.MustParsePipeline(t, `
metadata:
  name: pipeline
  namespace: foo
spec:
  tasks:
  - name: use-workspace
    taskSpec:
      steps:
      - image: ubuntu
      workspaces:
      - name: task-workspace
        mountPath: /foo/bar/
    workspaces:
    - name: task-workspace
      workspace: pipeline-workspace
  workspaces:
  - name: pipeline-workspace
`),
		run: test.MustParseRun(t, run+`
  workspaces:
    - name: pipeline-workspace
      persistentVolumeClaim:
        claimName: pvc
`),
	}, {
		name:            "workspaces with readonly - TODO(community#447)",
		expectedErrText: []string{"readOnly"},
		pipeline: test.MustParsePipeline(t, `
metadata:
  name: pipeline
  namespace: foo
spec:
  tasks:
  - name: use-workspace
    taskSpec:
      steps:
      - image: ubuntu
      workspaces:
      - name: task-workspace
        readOnly: true
    workspaces:
    - name: task-workspace
      workspace: pipeline-workspace
  workspaces:
  - name: pipeline-workspace
`),
		run: test.MustParseRun(t, run+`
  workspaces:
    - name: pipeline-workspace
      persistentVolumeClaim:
        claimName: pvc
`),
	}, {
		name:            "isolated workspaces - TODO(community#447)",
		expectedErrText: []string{"isolated"},
		pipeline: test.MustParsePipeline(t, `
metadata:
  name: pipeline
  namespace: foo
spec:
  tasks:
  - name: use-workspace
    taskSpec:
      steps:
      - image: ubuntu
        workspaces:
        - name: task-workspace
      workspaces:
      - name: task-workspace
    workspaces:
    - name: task-workspace
      workspace: pipeline-workspace
  workspaces:
  - name: pipeline-workspace
`),
		run: test.MustParseRun(t, run+`
  workspaces:
    - name: pipeline-workspace
      persistentVolumeClaim:
        claimName: pvc
`),
	}, {
		name:            "embedded tasks with labels and annotations - TODO(community#447)",
		expectedErrText: []string{"label", "annotation"},
		pipeline: test.MustParsePipeline(t, `
metadata:
  name: pipeline
  namespace: foo
spec:
  tasks:
  - name: some-task
    taskSpec:
      metadata:
        labels:
          some-key: some-value
        annotations:
          description: this embedded task uses labels and annotations
      steps:
      - image: ubuntu
`),
		run: test.MustParseRun(t, run),
	}, {
		name:            "results between tasks",
		expectedErrText: []string{"result"},
		pipeline: test.MustParsePipeline(t, `
metadata:
  name: pipeline
  namespace: foo
spec:
  tasks:
  - name: make-result
    taskSpec:
      steps:
      - image: ubuntu
      results:
      - name: amazing
  - name: use-result
    params:
    - name: foo
      value: $(tasks.make-result.results.amazing)
    taskSpec:
      params:
      - name: foo
      steps:
      - image: ubuntu
`),
		run: test.MustParseRun(t, run),
	}, {
		name:            "task with sidecar",
		expectedErrText: []string{"sidecar"},
		pipeline: test.MustParsePipeline(t, `
metadata:
  name: pipeline
  namespace: foo
spec:
  tasks:
  - name: use-sidecar
    taskSpec:
      steps:
      - image: ubuntu
      sidecars:
      - image: ubuntu
`),
		run: test.MustParseRun(t, run),
	}, {
		name:            "tasks as bundles",
		expectedErrText: []string{"bundle"},
		pipeline: test.MustParsePipeline(t, `
metadata:
  name: pipeline
  namespace: foo
spec:
  tasks:
  - name: use-bundle
    taskRef:
      name: echo-task
      bundle: docker.com/myrepo/mycatalog
`),
		run: test.MustParseRun(t, run),
	}, {
		name:            "pipeline task timeouts",
		expectedErrText: []string{"timeout"},
		pipeline: test.MustParsePipeline(t, `
metadata:
  name: pipeline
  namespace: foo
spec:
  tasks:
  - name: some-task
    timeout: "0h1m30s"
    taskSpec:
      steps:
      - image: ubuntu
`),
		run: test.MustParseRun(t, run),
	}, {
		name:            "retries",
		expectedErrText: []string{"retries"},
		pipeline: test.MustParsePipeline(t, `
metadata:
  name: pipeline
  namespace: foo
spec:
  tasks:
  - name: some-task
    retries: 1
    taskSpec:
      steps:
      - image: ubuntu
`),
		run: test.MustParseRun(t, run),
	}, {
		name:            "when expressions - new approach required to support",
		expectedErrText: []string{"when"},
		pipeline: test.MustParsePipeline(t, `
metadata:
  name: pipeline
  namespace: foo
spec:
  tasks:
  - name: some-task
    when:
      - input: "foo"
        operator: in
        values: ["bar"]    
    taskSpec:
      steps:
      - image: ubuntu
`),
		run: test.MustParseRun(t, run),
	}, {
		name:            "finally tasks - new approach required to support",
		expectedErrText: []string{"finally"},
		pipeline: test.MustParsePipeline(t, `
metadata:
  name: pipeline
  namespace: foo
spec:
  tasks:
  - name: some-task
    taskSpec:
      steps:
      - image: ubuntu
  finally:
  - name: some-finally-task
    taskSpec:
      steps:
      - image: ubuntu
`),
		run: test.MustParseRun(t, run),
	}, {
		name:            "parallel tasks (two, starting immediately) - new approach required to support",
		expectedErrText: []string{"parallel"},
		pipeline: test.MustParsePipeline(t, `
metadata:
  name: pipeline
  namespace: foo
spec:
  tasks:
  - name: a-parallel-task
    taskSpec:
      steps:
      - image: ubuntu
  - name: another-parallel-task
    taskSpec:
      steps:
      - image: ubuntu
`),
		run: test.MustParseRun(t, run),
	}, {
		name:            "parallel tasks (branch after first task) - new approach required to support",
		expectedErrText: []string{"parallel"},
		pipeline: test.MustParsePipeline(t, `
metadata:
  name: pipeline
  namespace: foo
spec:
  tasks:
  - name: first-task
    taskSpec:
      steps:
      - image: ubuntu
  - name: a-parallel-task
    runAfter: [first-task]
    taskSpec:
      steps:
      - image: ubuntu
  - name: another-parallel-task
    runAfter: [first-task]
    taskSpec:
      steps:
      - image: ubuntu
`),
		run: test.MustParseRun(t, run),
	}, {
		name:            "custom tasks - new approach required to support",
		expectedErrText: []string{"custom task"},
		pipeline: test.MustParsePipeline(t, `
metadata:
  name: pipeline
  namespace: foo
spec:
  tasks:
  - name: use-custom-task
    taskRef:
      apiVersion: tekton.dev/v1alpha1
      kind: SomeCustomTask
      name: some-custom-task
`),
		run: test.MustParseRun(t, run),
	}, {
		name:            "conditions - unlikely to support",
		expectedErrText: []string{"condition"},
		pipeline: test.MustParsePipeline(t, `
metadata:
  name: pipeline
  namespace: foo
spec:
  tasks:
  - name: some-task
    conditions:
    - conditionRef: some-condition
      params:
      - name: foo
        value: bar
    taskSpec:
      steps:
      - image: ubuntu
`),
		run: test.MustParseRun(t, run),
	}, {
		name:            "pipelineresources - unlikely to support",
		expectedErrText: []string{"pipelineresources"},
		pipeline: test.MustParsePipeline(t, `
metadata:
  name: pipeline
  namespace: foo
spec:
  resources:
  - name: source-repo
    type: git
    optional: true
  tasks:
  - name: some-task
    resources:
      inputs:
      - name: some-source
        resource: source-repo
    taskSpec:
      resources:
        inputs:
        - name: some-source
          type: git
      steps:
      - image: ubuntu
`),
		run: test.MustParseRun(t, run),
	}, {
		name: "volumes and volume mounts - unlikely to support",
		pipeline: test.MustParsePipeline(t, `
metadata:
  name: pipeline
  namespace: foo
spec:
  tasks:
  - name: use-volume
    taskSpec:
      steps:
      - image: ubuntu
        volumeMounts:
        - name: my-volume-mount
          mountPath: /foo/bar/baz
      volumes:
      - name: my-volume-mount
        empty-dir: {}
`),
		run: test.MustParseRun(t, run),
	}, {
		name:            "missing name",
		expectedErrText: []string{"name"},
		pipeline:        p,
		run: test.MustParseRun(t, `
metadata:
  name: run-with-pipeline
  namespace: foo
spec:
  ref:
    apiVersion: tekton.dev/v1alpha1
    kind: PipelineToTaskRun
`),
	}}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			names.TestingSeed()

			d := test.Data{
				Runs:      []*v1alpha1.Run{tc.run},
				Pipelines: []*v1beta1.Pipeline{tc.pipeline},
			}

			testAssets, _ := getController(t, d)

			err := testAssets.Controller.Reconciler.Reconcile(ctx, getRunName(tc.run))
			if err == nil {
				t.Errorf("expected permanenent error for invalid error (i.e. indication not to requeue) but got none")
			} else {
				if !controller.IsPermanentError(err) {
					t.Errorf("invalid run should return permanent error but instead got %v", err)
				}
			}

			run, err := testAssets.Clients.Pipeline.TektonV1alpha1().Runs(tc.run.Namespace).Get(ctx, tc.run.Name, metav1.GetOptions{})
			if err != nil {
				t.Fatalf("Error getting reconciled run from fake client: %s", err)
			}
			condition := run.Status.GetCondition(apis.ConditionSucceeded)
			if condition.Status != corev1.ConditionFalse {
				t.Errorf("exepcted invalid run to be marked failed but condition was %v", condition)
			}

			for _, text := range tc.expectedErrText {
				if !strings.Contains(condition.Message, text) {
					t.Errorf("excepected failure message to container %q but was %q", text, condition.Message)
				}
			}

		})
	}
}
