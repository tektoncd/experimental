package taskrun

import (
	"context"

	"github.com/tektoncd/experimental/metrics-operator/pkg/metrics"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"knative.dev/pkg/reconciler"
)

type Reconciler struct {
	manager *metrics.MetricManager
}

func (r *Reconciler) ReconcileKind(ctx context.Context, taskRun *pipelinev1beta1.TaskRun) reconciler.Event {
	if taskRun.IsDone() {
		return r.manager.RecordTaskRunDone(ctx, taskRun)
	}
	return r.manager.RecordTaskRunRunning(ctx, taskRun)
}

func (r *Reconciler) FinalizeKind(ctx context.Context, taskRun *pipelinev1beta1.TaskRun) reconciler.Event {
	if taskRun.IsDone() {
		r.manager.GetIndex().Clean(ctx, taskRun)
		return r.manager.RecordTaskRunDone(ctx, taskRun)
	}
	return r.manager.RecordTaskRunRunning(ctx, taskRun)
}
