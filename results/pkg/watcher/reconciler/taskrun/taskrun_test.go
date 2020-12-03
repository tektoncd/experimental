package taskrun

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/experimental/results/pkg/watcher/convert"
	"github.com/tektoncd/experimental/results/pkg/watcher/reconciler/annotation"
	"github.com/tektoncd/experimental/results/pkg/watcher/reconciler/internal/test"
	pb "github.com/tektoncd/experimental/results/proto/v1alpha1/results_go_proto"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	pipelinetest "github.com/tektoncd/pipeline/test"
	"google.golang.org/protobuf/testing/protocmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type taskRunTest struct {
	taskRun *v1beta1.TaskRun
	asset   pipelinetest.Assets
	ctx     context.Context
	client  pb.ResultsClient
}

func newTaskRunTest(t *testing.T) *taskRunTest {
	client := test.NewResultsClient(t)
	taskRun := &v1beta1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "Tekton-TaskRun",
			Namespace:   "default",
			Annotations: map[string]string{"demo": "demo"},
			UID:         "12345",
		},
	}
	d := pipelinetest.Data{
		TaskRuns: []*v1beta1.TaskRun{taskRun},
	}
	ctx, tclients, cmw := test.GetFakeClients(t, d, client)
	taskRunTest := &taskRunTest{
		taskRun: taskRun,
		asset: pipelinetest.Assets{
			Controller: NewController(ctx, cmw, client),
			Clients:    tclients,
		},
		ctx:    ctx,
		client: client,
	}
	return taskRunTest
}

func TestReconcile_CreateTaskRun(t *testing.T) {
	tt := newTaskRunTest(t)
	tr, err := test.ReconcileTaskRun(tt.ctx, tt.asset, tt.taskRun)
	if err != nil {
		t.Fatalf("Failed to get completed TaskRun %s: %v", tt.taskRun.Name, err)
	}
	if _, ok := tr.Annotations[annotation.ResultID]; !ok {
		t.Fatalf("Expected completed TaskRun %s should be updated with a results_id field in annotations", tt.taskRun.Name)
	}
	if _, err := tt.client.GetResult(tt.ctx, &pb.GetResultRequest{Name: tr.Annotations[annotation.ResultID]}); err != nil {
		t.Fatalf("Expected completed TaskRun %s not created in api server", tt.taskRun.Name)
	}
}

func TestReconcile_UnchangeTaskRun(t *testing.T) {
	tt := newTaskRunTest(t)

	// Reconcile once to get IDs, etc.
	tr, err := test.ReconcileTaskRun(tt.ctx, tt.asset, tt.taskRun)
	if err != nil {
		t.Fatalf("failed to get completed TaskRun %s: %v", tt.taskRun.Name, err)
	}

	// Reconcile again to verify nothing changes.
	newtr, err := test.ReconcileTaskRun(tt.ctx, tt.asset, tt.taskRun)
	if err != nil {
		t.Fatalf("failed to get completed TaskRun %s: %v", tt.taskRun.Name, err)
	}
	if diff := cmp.Diff(tr, newtr); diff != "" {
		t.Error(diff)
	}
}

func TestReconcile_UpdateTaskRun(t *testing.T) {
	tt := newTaskRunTest(t)
	tr, err := test.ReconcileTaskRun(tt.ctx, tt.asset, tt.taskRun)
	if err != nil {
		t.Fatalf("Failed to get completed TaskRun %s: %v", tt.taskRun.Name, err)
	}
	tr.UID = "234435"
	if _, err := tt.asset.Clients.Pipeline.TektonV1beta1().TaskRuns(tt.taskRun.Namespace).Update(tr); err != nil {
		t.Fatalf("Failed to update TaskRun %s to Tekton Pipeline Client: %v", tt.taskRun.Name, err)
	}
	updatetr, err := test.ReconcileTaskRun(tt.ctx, tt.asset, tr)
	if err != nil {
		t.Fatalf("Failed to reconcile TaskRun %s: %v", tt.taskRun.Name, err)
	}
	updatetr.ResourceVersion = tr.ResourceVersion
	if diff := cmp.Diff(tr, updatetr); diff != "" {
		t.Fatalf("Expected completed TaskRun should be updated in cluster: %v", diff)
	}
	res, err := tt.client.GetResult(tt.ctx, &pb.GetResultRequest{Name: tr.Annotations[annotation.ResultID]})
	if err != nil {
		t.Fatalf("Expected completed TaskRun %s not created in api server", tt.taskRun.Name)
	}
	p, err := convert.ToTaskRunProto(updatetr)
	if err != nil {
		t.Fatalf("failed to convert to proto: %v", err)
	}
	want := &pb.Result{
		Name: tr.Annotations[annotation.ResultID],
		Executions: []*pb.Execution{{
			Execution: &pb.Execution_TaskRun{p},
		}},
	}
	if diff := cmp.Diff(want, res, protocmp.Transform()); diff != "" {
		t.Fatalf("Expected completed TaskRun should be upated in api server: %v", diff)
	}
}
