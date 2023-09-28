package pipelinerun

import (
	"context"

	"github.com/tektoncd/experimental/metrics-operator/pkg/metrics"
	"github.com/tektoncd/experimental/metrics-operator/pkg/metrics/recorder"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"knative.dev/pkg/reconciler"
)

type Reconciler struct {
	manager *metrics.MetricManager
}

func (r *Reconciler) ReconcileKind(ctx context.Context, pipelineRun *pipelinev1beta1.PipelineRun) reconciler.Event {
	if pipelineRun.IsDone() {
		return r.manager.RecordPipelineRunDone(ctx, pipelineRun)
	}
	return r.manager.RecordPipelineRunRunning(ctx, pipelineRun)
}

func (r *Reconciler) FinalizeKind(ctx context.Context, pipelineRun *pipelinev1beta1.PipelineRun) reconciler.Event {
	run := recorder.PipelineRunDimensions(pipelineRun)
	if pipelineRun.IsDone() {
		r.manager.GetIndex().Clean(ctx, run)
		return r.manager.RecordPipelineRunDone(ctx, pipelineRun)
	}
	return r.manager.RecordPipelineRunRunning(ctx, pipelineRun)
}
