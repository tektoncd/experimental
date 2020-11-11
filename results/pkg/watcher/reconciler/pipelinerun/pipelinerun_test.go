package pipelinerun

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/experimental/results/pkg/watcher/convert"
	"github.com/tektoncd/experimental/results/pkg/watcher/reconciler/common"
	"github.com/tektoncd/experimental/results/pkg/watcher/reconciler/internal"
	pb "github.com/tektoncd/experimental/results/proto/proto"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/test"
	"google.golang.org/protobuf/testing/protocmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestReconcile(t *testing.T) {
	pipelineRunTest := NewPipelineRunTest(t)

	testFuncs := map[string]func(t *testing.T){
		"Create":   pipelineRunTest.testCreatePipelineRun,
		"Unchange": pipelineRunTest.testUnchangePipelineRun,
		"Update":   pipelineRunTest.testUpdatePipelineRun,
	}

	for name, testFunc := range testFuncs {
		t.Run(name, testFunc)
	}
}

type PipelineRunTest struct {
	pipelineRun *v1beta1.PipelineRun
	asset       test.Assets
	ctx         context.Context
	client      pb.ResultsClient
}

func NewPipelineRunTest(t *testing.T) PipelineRunTest {
	client := internal.NewResultsClient(t)
	pipelineRun := &v1beta1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "Tekton-PipelineRun",
			Namespace:   "default",
			Annotations: map[string]string{"demo": "pipelinerun_demo"},
			UID:         "54321",
		},
	}
	d := test.Data{
		PipelineRuns: []*v1beta1.PipelineRun{pipelineRun},
	}
	ctx, tclients, cmw := internal.GetFakeClients(t, d, client)
	pipelineRunTest := PipelineRunTest{
		pipelineRun: pipelineRun,
		asset: test.Assets{
			Controller: NewController(ctx, cmw, client),
			Clients:    tclients,
		},
		ctx:    ctx,
		client: client,
	}
	return pipelineRunTest
}

func (tt *PipelineRunTest) testCreatePipelineRun(t *testing.T) {
	pr, err := common.ReconcilePipelineRun(tt.ctx, tt.asset, tt.pipelineRun)
	if err != nil {
		t.Fatalf("Failed to get completed PipelineRun %s: %v", tt.pipelineRun.Name, err)
	}
	if _, ok := pr.Annotations[common.IDName]; !ok {
		t.Fatalf("Expected completed PipelineRun %s should be updated with a results_id field in annotations", tt.pipelineRun.Name)
	}
	if _, err := tt.client.GetResult(tt.ctx, &pb.GetResultRequest{Name: pr.Annotations[common.IDName]}); err != nil {
		t.Fatalf("Expected completed PipelineRun %s not created in api server", tt.pipelineRun.Name)
	}
}

func (tt *PipelineRunTest) testUnchangePipelineRun(t *testing.T) {
	pr, err := tt.asset.Clients.Pipeline.TektonV1beta1().PipelineRuns(tt.pipelineRun.Namespace).Get(tt.pipelineRun.Name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get completed PipelineRun %s: %v", tt.pipelineRun.Name, err)
	}
	newpr, err := common.ReconcilePipelineRun(tt.ctx, tt.asset, pr)
	if err != nil {
		t.Fatalf("Failed to get completed PipelineRun %s: %v", tt.pipelineRun.Name, err)
	}
	if diff := cmp.Diff(pr, newpr); diff != "" {
		t.Fatalf("Expected completed PipelineRun should remain unchanged when it has a results_id in annotations: %v", diff)
	}
}

func (tt *PipelineRunTest) testUpdatePipelineRun(t *testing.T) {
	pr, err := common.ReconcilePipelineRun(tt.ctx, tt.asset, tt.pipelineRun)
	if err != nil {
		t.Fatalf("Failed to get completed PipelineRun %s: %v", tt.pipelineRun.Name, err)
	}
	pr.UID = "234435"
	if _, err := tt.asset.Clients.Pipeline.TektonV1beta1().PipelineRuns(tt.pipelineRun.Namespace).Update(pr); err != nil {
		t.Fatalf("Failed to update PipelineRun %s: %v", tt.pipelineRun.Name, err)
	}
	updatepr, err := common.ReconcilePipelineRun(tt.ctx, tt.asset, pr)
	if err != nil {
		t.Fatalf("Failed to reconcile PipelineRun %s: %v", tt.pipelineRun.Name, err)
	}
	updatepr.ResourceVersion = pr.ResourceVersion
	if diff := cmp.Diff(pr, updatepr); diff != "" {
		t.Fatalf("Expected completed PipelineRun should be updated in cluster: %v", diff)
	}
	res, err := tt.client.GetResult(tt.ctx, &pb.GetResultRequest{Name: pr.Annotations[common.IDName]})
	if err != nil {
		t.Fatalf("Expected completed PipelineRun %s not created in api server", tt.pipelineRun.Name)
	}
	p, err := convert.ToPipelineRunProto(updatepr)
	if err != nil {
		t.Fatalf("failed to convert to proto: %v", err)
	}
	want := &pb.Result{
		Name: pr.Annotations[common.IDName],
		Executions: []*pb.Execution{{
			Execution: &pb.Execution_PipelineRun{p},
		}},
	}
	if diff := cmp.Diff(want, res, protocmp.Transform()); diff != "" {
		t.Fatalf("Expected completed PipelineRun should be upated in api server: %v", diff)
	}
}
