package reconciler

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/experimental/results/pkg/watcher/convert"
	"github.com/tektoncd/experimental/results/pkg/watcher/reconciler/common"
	"github.com/tektoncd/experimental/results/pkg/watcher/reconciler/pipelinerun"
	"github.com/tektoncd/experimental/results/pkg/watcher/reconciler/taskrun"
	pb "github.com/tektoncd/experimental/results/proto/proto"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	ttesting "github.com/tektoncd/pipeline/pkg/reconciler/testing"
	"github.com/tektoncd/pipeline/test"
	"google.golang.org/protobuf/testing/protocmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"knative.dev/pkg/configmap"
)

func TestReconciler(t *testing.T) {
	var reconcilerTest ReconcilerTest

	testFuncs := map[string]func(*testing.T){
		"Update PipelineRun to the existed Result": reconcilerTest.testUpdatePipelineRunToTheExistedResult,
		"Update TaskRun to the existed Result":     reconcilerTest.testUpdateTaskRunToTheExistedResult,
	}

	for name, testFunc := range testFuncs {
		t.Run(name, testFunc)
	}
}

type ReconcilerTest struct {
	taskRun *v1beta1.TaskRun
	trAsset test.Assets

	pipelineRun *v1beta1.PipelineRun
	prAsset     test.Assets
	ctx         context.Context
	client      pb.ResultsClient
}

func newReconcilerTest(t *testing.T) *ReconcilerTest {
	client := common.NewResultsClient(t)
	tr := &v1beta1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "Tekton-Result",
			Namespace:   "default",
			Annotations: map[string]string{"demo": "demo"},
			UID:         "1",
		},
	}
	pr := &v1beta1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "Tekton-Result",
			Namespace:   "default",
			Annotations: map[string]string{"demo": "demo"},
			UID:         "2",
		},
	}
	trAsset, prAsset, ctx := getFakeClients(t, []*v1beta1.TaskRun{tr}, []*v1beta1.PipelineRun{pr}, client)
	return &ReconcilerTest{
		taskRun:     tr,
		trAsset:     trAsset,
		pipelineRun: pr,
		prAsset:     prAsset,

		ctx:    ctx,
		client: client,
	}
}

func getFakeClients(t *testing.T, tr []*v1beta1.TaskRun, pr []*v1beta1.PipelineRun, client pb.ResultsClient) (test.Assets, test.Assets, context.Context) {
	t.Helper()
	ctx, _ := ttesting.SetupFakeContext(t)
	d := test.Data{
		TaskRuns:     tr,
		PipelineRuns: pr,
	}
	clients, _ := test.SeedTestData(t, ctx, d)
	cmw := configmap.NewInformedWatcher(clients.Kube, "")

	return test.Assets{
			Controller: taskrun.NewController(ctx, cmw, client),
			Clients:    clients,
		}, test.Assets{
			Controller: pipelinerun.NewController(ctx, cmw, client),
			Clients:    clients,
		}, ctx
}

func (tt *ReconcilerTest) testUpdatePipelineRunToTheExistedResult(t *testing.T) {
	tt = newReconcilerTest(t)
	// Create a TaskRun
	tr, err := common.ReconcileTaskRun(tt.ctx, tt.trAsset, tt.taskRun)
	if err != nil {
		t.Fatalf("Failed to get completed TaskRun %s: %v", tt.taskRun.Name, err)
	}
	resultID, ok := tr.Annotations[common.IDName]
	if !ok {
		t.Fatalf("Expected completed TaskRun %s should be updated with a results_id field in annotations", tt.taskRun.Name)
	}
	trResult, err := tt.client.GetResult(tt.ctx, &pb.GetResultRequest{Name: resultID})
	if err != nil {
		t.Fatalf("Expected completed TaskRun %s not created in api server", tt.taskRun.Name)
	}

	path, err := common.AnnotationPath(trResult.GetName(), common.Path, "add")
	if err != nil {
		t.Fatalf("Error jsonpatch for TaskRun Result %s: %v", trResult.GetName(), err)
	}
	// Give the PipelineRun the same Result ID as the TaskRun
	pr, err := tt.prAsset.Clients.Pipeline.TektonV1beta1().PipelineRuns(tt.pipelineRun.Namespace).Patch(tt.pipelineRun.Name, types.JSONPatchType, path)
	if err != nil {
		t.Fatalf("Failed to apply result patch to PipelineRun: %v", err)
	}

	// Update the PipelineRun to the same Result
	pr, err = common.ReconcilePipelineRun(tt.ctx, tt.prAsset, tt.pipelineRun)
	if err != nil {
		t.Fatalf("Failed to reconcile PipelineRun: %v", err)
	}
	prResult, err := tt.client.GetResult(tt.ctx, &pb.GetResultRequest{Name: resultID})
	if err != nil {
		t.Fatalf("Expected completed PipelineRun %s not updated in api server: %v", tt.pipelineRun.Name, err)
	}
	prProto, err := convert.ToPipelineRunProto(pr)
	if err != nil {
		t.Fatalf("Failed to convert to proto: %v", err)
	}

	want := trResult
	want.Executions = append(want.Executions, &pb.Execution{Execution: &pb.Execution_PipelineRun{prProto}})
	if diff := cmp.Diff(want, prResult, protocmp.Transform()); diff != "" {
		t.Fatalf("Expected completed PipelineRun should be upated in api server: %v", diff)
	}
}

func (tt *ReconcilerTest) testUpdateTaskRunToTheExistedResult(t *testing.T) {
	tt = newReconcilerTest(t)
	// Create a PipelineRun
	pr, err := common.ReconcilePipelineRun(tt.ctx, tt.prAsset, tt.pipelineRun)
	if err != nil {
		t.Fatalf("Failed to get completed PipelineRun %s: %v", tt.pipelineRun.Name, err)
	}
	resultID, ok := pr.Annotations[common.IDName]
	if !ok {
		t.Fatalf("Expected completed PipelineRun %s should be updated with a results_id field in annotations", tt.pipelineRun.Name)
	}
	prResult, err := tt.client.GetResult(tt.ctx, &pb.GetResultRequest{Name: resultID})
	if err != nil {
		t.Fatalf("Expected completed PipelineRun %s not created in api server", tt.pipelineRun.Name)
	}

	path, err := common.AnnotationPath(prResult.GetName(), common.Path, "add")
	if err != nil {
		t.Fatalf("Error jsonpatch for PipelineRun Result %s: %v", prResult.GetName(), err)
	}
	// Give the TaskRun the same Result ID as the PipelineRun
	tr, err := tt.trAsset.Clients.Pipeline.TektonV1beta1().TaskRuns(tt.taskRun.Namespace).Patch(tt.taskRun.Name, types.JSONPatchType, path)
	if err != nil {
		t.Fatalf("Failed to apply result patch to TaskRun: %v", err)
	}

	// Update the TaskRun to the same Result
	tr, err = common.ReconcileTaskRun(tt.ctx, tt.trAsset, tt.taskRun)
	if err != nil {
		t.Fatalf("Failed to reconcile TaskRun: %v", err)
	}
	trResult, err := tt.client.GetResult(tt.ctx, &pb.GetResultRequest{Name: resultID})
	if err != nil {
		t.Fatalf("Expected completed TaskRun %s not updated in api server: %v", tt.taskRun.Name, err)
	}
	trProto, err := convert.ToTaskRunProto(tr)
	if err != nil {
		t.Fatalf("Failed to convert to proto: %v", err)
	}

	want := prResult
	want.Executions = append(want.Executions, &pb.Execution{Execution: &pb.Execution_TaskRun{trProto}})
	if diff := cmp.Diff(want, trResult, protocmp.Transform()); diff != "" {
		t.Fatalf("Expected completed TaskRun should be upated in api server: %v", diff)
	}
}
