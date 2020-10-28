package taskrun

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/experimental/results/pkg/watcher/convert"
	"github.com/tektoncd/experimental/results/pkg/watcher/reconciler/common"
	"github.com/tektoncd/experimental/results/pkg/watcher/reconciler/internal"
	pb "github.com/tektoncd/experimental/results/proto/v1alpha1/results_go_proto"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/test"
	"google.golang.org/protobuf/testing/protocmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestReconcile(t *testing.T) {
	taskRunTest := NewTaskRunTest(t)

	testFuncs := map[string]func(t *testing.T){
		"Create":   taskRunTest.testCreateTaskRun,
		"Unchange": taskRunTest.testUnchangeTaskRun,
		"Update":   taskRunTest.testUpdateTaskRun,
	}

	for name, testFunc := range testFuncs {
		t.Run(name, testFunc)
	}
}

type TaskRunTest struct {
	taskRun *v1beta1.TaskRun
	asset   test.Assets
	ctx     context.Context
	client  pb.ResultsClient
}

func NewTaskRunTest(t *testing.T) TaskRunTest {
	client := internal.NewResultsClient(t)
	taskRun := &v1beta1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "Tekton-TaskRun",
			Namespace:   "default",
			Annotations: map[string]string{"demo": "demo"},
			UID:         "12345",
		},
	}
	d := test.Data{
		TaskRuns: []*v1beta1.TaskRun{taskRun},
	}
	ctx, tclients, cmw := internal.GetFakeClients(t, d, client)
	taskRunTest := TaskRunTest{
		taskRun: taskRun,
		asset: test.Assets{
			Controller: NewController(ctx, cmw, client),
			Clients:    tclients,
		},
		ctx:    ctx,
		client: client,
	}
	return taskRunTest
}

func (tt *TaskRunTest) testCreateTaskRun(t *testing.T) {
	tr, err := common.ReconcileTaskRun(tt.ctx, tt.asset, tt.taskRun)
	if err != nil {
		t.Fatalf("Failed to get completed TaskRun %s: %v", tt.taskRun.Name, err)
	}
	if _, ok := tr.Annotations[common.IDName]; !ok {
		t.Fatalf("Expected completed TaskRun %s should be updated with a results_id field in annotations", tt.taskRun.Name)
	}
	if _, err := tt.client.GetResult(tt.ctx, &pb.GetResultRequest{Name: tr.Annotations[common.IDName]}); err != nil {
		t.Fatalf("Expected completed TaskRun %s not created in api server", tt.taskRun.Name)
	}
}

func (tt *TaskRunTest) testUnchangeTaskRun(t *testing.T) {
	tr, err := tt.asset.Clients.Pipeline.TektonV1beta1().TaskRuns(tt.taskRun.Namespace).Get(tt.taskRun.Name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get completed TaskRun %s: %v", tt.taskRun.Name, err)
	}
	newtr, err := common.ReconcileTaskRun(tt.ctx, tt.asset, tr)
	if err != nil {
		t.Fatalf("Failed to get completed TaskRun %s: %v", tt.taskRun.Name, err)
	}
	if diff := cmp.Diff(tr, newtr); diff != "" {
		t.Fatalf("Expected completed TaskRun should remain unchanged when it has a results_id in annotations: %v", diff)
	}
}

func (tt *TaskRunTest) testUpdateTaskRun(t *testing.T) {
	tr, err := common.ReconcileTaskRun(tt.ctx, tt.asset, tt.taskRun)
	if err != nil {
		t.Fatalf("Failed to get completed TaskRun %s: %v", tt.taskRun.Name, err)
	}
	tr.UID = "234435"
	if _, err := tt.asset.Clients.Pipeline.TektonV1beta1().TaskRuns(tt.taskRun.Namespace).Update(tr); err != nil {
		t.Fatalf("Failed to update TaskRun %s to Tekton Pipeline Client: %v", tt.taskRun.Name, err)
	}
	updatetr, err := common.ReconcileTaskRun(tt.ctx, tt.asset, tr)
	if err != nil {
		t.Fatalf("Failed to reconcile TaskRun %s: %v", tt.taskRun.Name, err)
	}
	updatetr.ResourceVersion = tr.ResourceVersion
	if diff := cmp.Diff(tr, updatetr); diff != "" {
		t.Fatalf("Expected completed TaskRun should be updated in cluster: %v", diff)
	}
	res, err := tt.client.GetResult(tt.ctx, &pb.GetResultRequest{Name: tr.Annotations[common.IDName]})
	if err != nil {
		t.Fatalf("Expected completed TaskRun %s not created in api server", tt.taskRun.Name)
	}
	p, err := convert.ToTaskRunProto(updatetr)
	if err != nil {
		t.Fatalf("failed to convert to proto: %v", err)
	}
	want := &pb.Result{
		Name: tr.Annotations[common.IDName],
		Executions: []*pb.Execution{{
			Execution: &pb.Execution_TaskRun{p},
		}},
	}
	if diff := cmp.Diff(want, res, protocmp.Transform()); diff != "" {
		t.Fatalf("Expected completed TaskRun should be upated in api server: %v", diff)
	}
}
