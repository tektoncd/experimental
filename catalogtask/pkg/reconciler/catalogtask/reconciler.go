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

package catalogtask

import (
	context "context"
	"fmt"

	catalog "github.com/tektoncd/experimental/catalogtask/pkg/catalog"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	clientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"github.com/tektoncd/pipeline/pkg/client/injection/reconciler/pipeline/v1alpha1/run"
	listers "github.com/tektoncd/pipeline/pkg/client/listers/pipeline/v1beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
	logging "knative.dev/pkg/logging"
	reconciler "knative.dev/pkg/reconciler"
)

// Ensure reconciler implements Interface
var _ run.Interface = (*Reconciler)(nil)

type Reconciler struct {
	catalog           catalog.Catalog
	pipelineClientSet clientset.Interface
	taskRunLister     listers.TaskRunLister
}

func debugLog(format string, args ...interface{}) {
	wrappedFormat := fmt.Sprintf("\n\n\n%s\n\n", format)
	fmt.Printf(wrappedFormat, args...)
}

func (r *Reconciler) ReconcileKind(ctx context.Context, run *v1alpha1.Run) reconciler.Event {
	logger := logging.FromContext(ctx)

	namespace := run.ObjectMeta.Namespace
	name := run.ObjectMeta.Name

	logger.Infof("Reconciling Run %s/%s", namespace, name)

	if !run.HasStarted() {
		return r.OnNewRun(ctx, run)
	}

	if !run.IsDone() {
		return r.reconcile(ctx, run)
	}

	return nil
}

func (r *Reconciler) OnNewRun(ctx context.Context, run *v1alpha1.Run) reconciler.Event {
	run.Status.InitializeConditions()
	r.syncTime(ctx, run)

	// TODO: Should relly only put Run into running status once TaskRun is in running status.
	return r.tryRunTask(ctx, run)
}

// syncTime checks in case node time was not synchronized
// when controller has been scheduled to other nodes.
func (r *Reconciler) syncTime(ctx context.Context, run *v1alpha1.Run) {
	logger := logging.FromContext(ctx)
	if run.Status.StartTime.Sub(run.CreationTimestamp.Time) < 0 {
		logger.Warnf("Run %s/%s createTimestamp %s is after the Run started %s", run.Namespace, run.Name, run.CreationTimestamp, run.Status.StartTime)
		run.Status.StartTime = &run.CreationTimestamp
	}
}

func (r *Reconciler) reconcile(ctx context.Context, run *v1alpha1.Run) reconciler.Event {
	// logger := logging.FromContext(ctx)
	tr, err := r.taskRunLister.TaskRuns(run.Namespace).Get(run.Name)
	if err != nil {
		debugLog("tried getting taskrun but error: %v", err)
		run.Status.MarkRunFailed("ErrorListingCatalogTaskRun", "%v", err)
		return nil
	}

	// Reflect all the taskrun status conditions into the run
	for _, cond := range tr.Status.Conditions {
		run.Status.SetCondition(&apis.Condition{
			Type:               cond.Type,
			Status:             cond.Status,
			Severity:           cond.Severity,
			LastTransitionTime: cond.LastTransitionTime,
			Reason:             cond.Reason,
			Message:            cond.Message,
		})
	}

	if tr.IsDone() && len(tr.Status.TaskRunResults) != 0 {
		for _, result := range tr.Status.TaskRunResults {
			run.Status.Results = append(run.Status.Results, v1alpha1.RunResult{
				Name:  result.Name,
				Value: result.Value,
			})
		}
	}

	return reconciler.NewEvent(v1.EventTypeNormal, "RunReconciled", "Run reconciled: \"%s/%s\"", run.Namespace, run.Name)
}

func (r *Reconciler) tryRunTask(ctx context.Context, run *v1alpha1.Run) reconciler.Event {
	namespace := run.ObjectMeta.Namespace
	name := run.ObjectMeta.Name

	if run.Spec.Ref.Name == "" {
		run.Status.MarkRunFailed("MissingCatalogTaskRefName", "spec.ref.name missing")
		return nil
	}
	taskName := run.Spec.Ref.Name

	task, err := r.catalog.Get(taskName)

	if err != nil {
		run.Status.MarkRunFailed("ErrorResolvingCatalogTask", "%v", err)
		return nil
	}

	tr := toTaskRun(task, run)
	if _, err = r.pipelineClientSet.TektonV1beta1().TaskRuns(tr.Namespace).Create(ctx, tr, metav1.CreateOptions{}); err != nil {
		debugLog("error creating taskrun from %q: %v", name, err)
		run.Status.MarkRunFailed("TaskRunCreationError", "task=%q: %v", name, err)
		return nil
	}

	return reconciler.NewEvent(v1.EventTypeNormal, "RunReconciled", "Run reconciled: \"%s/%s\"", namespace, name)
}

func toTaskRun(task *v1beta1.Task, r *v1alpha1.Run) *v1beta1.TaskRun {
	tr := &v1beta1.TaskRun{
		TypeMeta: metav1.TypeMeta{
			Kind:       "taskrun",
			APIVersion: "tekton.dev/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.ObjectMeta.Name,
			Namespace: r.ObjectMeta.Namespace,
		},
	}

	// set owner of taskrun to run so it gets cleaned up on delete
	isController := true
	tr.ObjectMeta.OwnerReferences = []metav1.OwnerReference{{
		Kind:       "Run",
		APIVersion: "tekton.dev/v1alpha1",
		Name:       r.ObjectMeta.Name,
		UID:        r.ObjectMeta.UID,
		Controller: &isController,
	}}

	tr.Spec = v1beta1.TaskRunSpec{}

	tr.Spec.TaskSpec = task.Spec.DeepCopy()

	for _, p := range r.Spec.Params {
		tr.Spec.Params = append(tr.Spec.Params, p)
	}

	for _, w := range r.Spec.Workspaces {
		tr.Spec.Workspaces = append(tr.Spec.Workspaces, *w.DeepCopy())
	}

	return tr
}

func getParam(run *v1alpha1.Run, name string) string {
	if run != nil {
		for _, p := range run.Spec.Params {
			if p.Name == name {
				return p.Value.StringVal
			}
		}
	}
	return ""
}
