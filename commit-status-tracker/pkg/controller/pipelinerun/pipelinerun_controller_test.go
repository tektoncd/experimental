// Copyright 2020 The Tekton Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pipelinerun

import (
	"reflect"
	"testing"

	"github.com/jenkins-x/go-scm/scm"
	fakescm "github.com/jenkins-x/go-scm/scm/driver/fake"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"knative.dev/pkg/apis"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	tb "github.com/tektoncd/experimental/commit-status-tracker/test/builder"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

var (
	testNamespace   = "test-namespace"
	pipelineRunName = "test-pipeline-run"
	testToken       = "abcdefghijklmnopqrstuvwxyz12345678901234"
)

var _ reconcile.Reconciler = &ReconcilePipelineRun{}

// TestPipelineRunControllerPendingState runs ReconcilePipelineRun.Reconcile() against a
// fake client that tracks PipelineRun objects.
func TestPipelineRunControllerPendingState(t *testing.T) {
	logf.SetLogger(logf.ZapLogger(true))
	pipelineRun := makePipelineRunWithResources(
		makeGitResourceBinding("https://github.com/tektoncd/triggers", "master"))
	applyOpts(
		pipelineRun,
		tb.PipelineRunAnnotation(notifiableName, "true"),
		tb.PipelineRunAnnotation(statusContextName, "test-context"),
		tb.PipelineRunAnnotation(statusDescriptionName, "testing"),
		tb.PipelineRunStatus(tb.PipelineRunStatusCondition(
			apis.Condition{Type: apis.ConditionSucceeded, Status: corev1.ConditionUnknown})))

	objs := []runtime.Object{
		pipelineRun,
		makeSecret(map[string][]byte{"token": []byte(testToken)}),
	}
	r, data := makeReconciler(t, "github", pipelineRun, objs...)

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      pipelineRunName,
			Namespace: testNamespace,
		},
	}
	res, err := r.Reconcile(req)
	fatalIfError(t, err, "reconcile: (%v)", err)
	if res.Requeue {
		t.Fatal("reconcile requeued request")
	}
	wanted := &scm.Status{State: scm.StatePending, Label: "test-context", Desc: "testing", Target: ""}
	status := data.Statuses["master"][0]
	if !reflect.DeepEqual(status, wanted) {
		t.Fatalf("commit-status notification got %#v, wanted %#v\n", status, wanted)
	}
}

// TestPipelineRunReconcileWithPreviousPending tests a PipelineRun that
// we've already sent a pending notification.
func TestPipelineRunReconcileWithPreviousPending(t *testing.T) {
	logf.SetLogger(logf.ZapLogger(true))
	pipelineRun := makePipelineRunWithResources(
		makeGitResourceBinding("https://github.com/tektoncd/triggers", "master"))
	applyOpts(
		pipelineRun,
		tb.PipelineRunAnnotation(notifiableName, "true"),
		tb.PipelineRunAnnotation(statusContextName, "test-context"),
		tb.PipelineRunAnnotation(statusDescriptionName, "testing"),
		tb.PipelineRunStatus(tb.PipelineRunStatusCondition(
			apis.Condition{Type: apis.ConditionSucceeded, Status: corev1.ConditionUnknown})))
	objs := []runtime.Object{
		pipelineRun,
		makeSecret(map[string][]byte{"token": []byte(testToken)}),
	}
	r, data := makeReconciler(t, "github", pipelineRun, objs...)

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      pipelineRunName,
			Namespace: testNamespace,
		},
	}
	// This runs Reconcile twice.
	res, err := r.Reconcile(req)
	fatalIfError(t, err, "reconcile: (%v)", err)
	if res.Requeue {
		t.Fatal("reconcile requeued request")
	}
	// This cleans out the existing date for the data, because the fake scm
	// client updates in-place, so there's no way to know if it received multiple
	// pending notifications.
	delete(data.Statuses, "master")
	res, err = r.Reconcile(req)
	fatalIfError(t, err, "reconcile: (%v)", err)
	if res.Requeue {
		t.Fatal("reconcile requeued request")
	}
	// There should be no recorded statuses, because the state is still pending
	// and the fake client's state was deleted above.
	assertNoStatusesRecorded(t, data)
}

// TestPipelineRunControllerSuccessState runs ReconcilePipelineRun.Reconcile() against a
// fake client that tracks PipelineRun objects.
func TestPipelineRunControllerSuccessState(t *testing.T) {
	logf.SetLogger(logf.ZapLogger(true))
	pipelineRun := makePipelineRunWithResources(
		makeGitResourceBinding("https://github.com/tektoncd/triggers", "master"))
	applyOpts(
		pipelineRun,
		tb.PipelineRunAnnotation(notifiableName, "true"),
		tb.PipelineRunAnnotation(statusContextName, "test-context"),
		tb.PipelineRunAnnotation(statusDescriptionName, "testing"),
		tb.PipelineRunStatus(tb.PipelineRunStatusCondition(
			apis.Condition{Type: apis.ConditionSucceeded, Status: corev1.ConditionTrue})))
	objs := []runtime.Object{
		pipelineRun,
		makeSecret(map[string][]byte{"token": []byte(testToken)}),
	}
	r, data := makeReconciler(t, "github", pipelineRun, objs...)

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      pipelineRunName,
			Namespace: testNamespace,
		},
	}
	res, err := r.Reconcile(req)
	fatalIfError(t, err, "reconcile: (%v)", err)
	if res.Requeue {
		t.Fatal("reconcile requeued request")
	}
	wanted := &scm.Status{State: scm.StateSuccess, Label: "test-context", Desc: "testing", Target: ""}
	status := data.Statuses["master"][0]
	if !reflect.DeepEqual(status, wanted) {
		t.Fatalf("commit-status notification got %#v, wanted %#v\n", status, wanted)
	}
}

// TestPipelineRunControllerFailedState runs ReconcilePipelineRun.Reconcile() against a
// fake client that tracks PipelineRun objects.
func TestPipelineRunControllerFailedState(t *testing.T) {
	logf.SetLogger(logf.ZapLogger(true))
	pipelineRun := makePipelineRunWithResources(
		makeGitResourceBinding("https://github.com/tektoncd/triggers", "master"))
	applyOpts(
		pipelineRun,
		tb.PipelineRunAnnotation(notifiableName, "true"),
		tb.PipelineRunAnnotation(statusContextName, "test-context"),
		tb.PipelineRunAnnotation(statusDescriptionName, "testing"),
		tb.PipelineRunStatus(tb.PipelineRunStatusCondition(
			apis.Condition{Type: apis.ConditionSucceeded, Status: corev1.ConditionFalse})))
	objs := []runtime.Object{
		pipelineRun,
		makeSecret(map[string][]byte{"token": []byte(testToken)}),
	}
	r, data := makeReconciler(t, "github", pipelineRun, objs...)

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      pipelineRunName,
			Namespace: testNamespace,
		},
	}
	res, err := r.Reconcile(req)
	fatalIfError(t, err, "reconcile: (%v)", err)
	if res.Requeue {
		t.Fatal("reconcile requeued request")
	}
	wanted := &scm.Status{State: scm.StateFailure, Label: "test-context", Desc: "testing", Target: ""}
	status := data.Statuses["master"][0]
	if !reflect.DeepEqual(status, wanted) {
		t.Fatalf("commit-status notification got %#v, wanted %#v\n", status, wanted)
	}
}

// TestPipelineRunReconcileWithNoGitCredentials tests a non-notifable
// PipelineRun.
func TestPipelineRunReconcileNonNotifiable(t *testing.T) {
	logf.SetLogger(logf.ZapLogger(true))
	pipelineRun := makePipelineRunWithResources(
		makeGitResourceBinding("https://github.com/tektoncd/triggers", "master"))
	applyOpts(
		pipelineRun,
		tb.PipelineRunStatus(tb.PipelineRunStatusCondition(
			apis.Condition{Type: apis.ConditionSucceeded, Status: corev1.ConditionUnknown})))
	objs := []runtime.Object{
		pipelineRun,
		makeSecret(map[string][]byte{"token": []byte(testToken)}),
	}
	r, data := makeReconciler(t, "github", pipelineRun, objs...)

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      pipelineRunName,
			Namespace: testNamespace,
		},
	}
	res, err := r.Reconcile(req)
	fatalIfError(t, err, "reconcile: (%v)", err)
	if res.Requeue {
		t.Fatal("reconcile requeued request")
	}
	assertNoStatusesRecorded(t, data)
}

// TestPipelineRunReconcileWithNoGitCredentials tests a notifable PipelineRun
// with no "git" resource.
func TestPipelineRunReconcileWithNoGitRepository(t *testing.T) {
	logf.SetLogger(logf.ZapLogger(true))
	pipelineRun := makePipelineRunWithResources()
	applyOpts(
		pipelineRun,
		tb.PipelineRunAnnotation(notifiableName, "true"),
		tb.PipelineRunAnnotation(statusContextName, "test-context"),
		tb.PipelineRunAnnotation(statusDescriptionName, "testing"),
		tb.PipelineRunStatus(tb.PipelineRunStatusCondition(
			apis.Condition{Type: apis.ConditionSucceeded, Status: corev1.ConditionUnknown})))
	objs := []runtime.Object{
		pipelineRun,
		makeSecret(map[string][]byte{"token": []byte(testToken)}),
	}
	r, data := makeReconciler(t, "", pipelineRun, objs...)

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      pipelineRunName,
			Namespace: testNamespace,
		},
	}
	res, err := r.Reconcile(req)
	fatalIfError(t, err, "reconcile: (%v)", err)
	if res.Requeue {
		t.Fatal("reconcile requeued request")
	}
	assertNoStatusesRecorded(t, data)
}

// TestPipelineRunReconcileWithNoGitCredentials tests a notifable PipelineRun
// with multiple "git" resources.
func TestPipelineRunReconcileWithGitRepositories(t *testing.T) {
	logf.SetLogger(logf.ZapLogger(true))
	pipelineRun := makePipelineRunWithResources(
		makeGitResourceBinding("https://github.com/tektoncd/triggers", "master"),
		makeGitResourceBinding("https://github.com/tektoncd/pipeline", "master"))
	applyOpts(
		pipelineRun,
		tb.PipelineRunAnnotation(notifiableName, "true"),
		tb.PipelineRunAnnotation(statusContextName, "test-context"),
		tb.PipelineRunAnnotation(statusDescriptionName, "testing"),
		tb.PipelineRunStatus(tb.PipelineRunStatusCondition(
			apis.Condition{Type: apis.ConditionSucceeded, Status: corev1.ConditionUnknown})))
	objs := []runtime.Object{
		pipelineRun,
		makeSecret(map[string][]byte{"token": []byte(testToken)}),
	}
	r, data := makeReconciler(t, "", pipelineRun, objs...)

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      pipelineRunName,
			Namespace: testNamespace,
		},
	}
	res, err := r.Reconcile(req)
	fatalIfError(t, err, "reconcile: (%v)", err)
	if res.Requeue {
		t.Fatal("reconcile requeued request")
	}
	assertNoStatusesRecorded(t, data)
}

// TestPipelineRunReconcileWithNoGitCredentials tests a notifable PipelineRun
// with a "git" resource, but with no Git credentials.
func TestPipelineRunReconcileWithNoGitCredentials(t *testing.T) {
	logf.SetLogger(logf.ZapLogger(true))
	pipelineRun := makePipelineRunWithResources(
		makeGitResourceBinding("https://github.com/tektoncd/triggers", "master"),
		makeGitResourceBinding("https://github.com/tektoncd/pipeline", "master"))
	applyOpts(
		pipelineRun,
		tb.PipelineRunAnnotation(notifiableName, "true"),
		tb.PipelineRunAnnotation(statusContextName, "test-context"),
		tb.PipelineRunAnnotation(statusDescriptionName, "testing"),
		tb.PipelineRunStatus(tb.PipelineRunStatusCondition(
			apis.Condition{Type: apis.ConditionSucceeded, Status: corev1.ConditionUnknown})))
	objs := []runtime.Object{pipelineRun}
	r, data := makeReconciler(t, "", pipelineRun, objs...)

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      pipelineRunName,
			Namespace: testNamespace,
		},
	}
	res, err := r.Reconcile(req)
	fatalIfError(t, err, "reconcile: (%v)", err)
	if res.Requeue {
		t.Fatal("reconcile requeued request")
	}
	assertNoStatusesRecorded(t, data)

}

func TestKeyForCommit(t *testing.T) {
	inputTests := []struct {
		repo string
		sha  string
		want string
	}{
		{"tekton/triggers", "e1466db56110fa1b813277c1647e20283d3370c3",
			"7b2841ab8791fece7acdc0b3bb6e398c7a184273"},
	}

	for _, tt := range inputTests {
		if v := keyForCommit(tt.repo, tt.sha); v != tt.want {
			t.Errorf("keyForCommit(%#v, %#v) got %#v, want %#v", tt.repo, tt.sha, v, tt.want)
		}
	}
}

func applyOpts(pr *pipelinev1.PipelineRun, opts ...tb.PipelineRunOp) {
	for _, o := range opts {
		o(pr)
	}
}

func makeReconciler(t *testing.T, wantType string, pr *pipelinev1.PipelineRun, objs ...runtime.Object) (*ReconcilePipelineRun, *fakescm.Data) {
	t.Helper()
	s := scheme.Scheme
	s.AddKnownTypes(pipelinev1.SchemeGroupVersion, pr)
	cl := fake.NewFakeClient(objs...)
	client, data := fakescm.NewDefault()
	fakeClientFactory := func(s, gotType string) *scm.Client {
		if wantType != gotType {
			t.Fatalf("repository type mismatch: got type %s, want %s", gotType, wantType)
		}
		return client
	}
	return &ReconcilePipelineRun{
		client:       cl,
		scheme:       s,
		scmFactory:   fakeClientFactory,
		pipelineRuns: make(pipelineRunTracker),
	}, data
}

func fatalIfError(t *testing.T, err error, format string, a ...interface{}) {
	t.Helper()
	if err != nil {
		t.Fatalf(format, a...)
	}
}

func assertNoStatusesRecorded(t *testing.T, d *fakescm.Data) {
	if l := len(d.Statuses["master"]); l != 0 {
		t.Fatalf("too many statuses recorded, got %v, wanted 0", l)
	}
}
