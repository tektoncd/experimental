package pipelinerun

import (
	"context"
	"net"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/experimental/results/pkg/api/server"
	"github.com/tektoncd/experimental/results/pkg/watcher/convert"
	pb "github.com/tektoncd/experimental/results/proto/proto"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	ttesting "github.com/tektoncd/pipeline/pkg/reconciler/testing"
	"github.com/tektoncd/pipeline/test"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/testing/protocmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/configmap"
)

const (
	port = ":0"
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
	client := newResultsClient(t)
	pipelineRun := &v1beta1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "Tekton-PipelineRun",
			Namespace:   "default",
			Annotations: map[string]string{"demo": "pipelinerun_demo"},
			UID:         "54321",
		},
	}
	asset, ctx := getFakeClients(t, []*v1beta1.PipelineRun{pipelineRun}, client)
	pipelineRunTest := PipelineRunTest{
		pipelineRun: pipelineRun,
		asset:       asset,
		ctx:         ctx,
		client:      client,
	}
	return pipelineRunTest
}

func (tt *PipelineRunTest) testCreatePipelineRun(t *testing.T) {
	pr, err := reconcile(tt.ctx, tt.asset, tt.pipelineRun)
	if err != nil {
		t.Fatalf("Failed to get completed PipelineRun %s: %v", tt.pipelineRun.Name, err)
	}
	if _, ok := pr.Annotations[idName]; !ok {
		t.Fatalf("Expected completed PipelineRun %s should be updated with a results_id field in annotations", tt.pipelineRun.Name)
	}
	if _, err := tt.client.GetResult(tt.ctx, &pb.GetResultRequest{Name: pr.Annotations[idName]}); err != nil {
		t.Fatalf("Expected completed PipelineRun %s not created in api server", tt.pipelineRun.Name)
	}
}

func (tt *PipelineRunTest) testUnchangePipelineRun(t *testing.T) {
	pr, err := tt.asset.Clients.Pipeline.TektonV1beta1().PipelineRuns(tt.pipelineRun.Namespace).Get(tt.pipelineRun.Name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get completed PipelineRun %s: %v", tt.pipelineRun.Name, err)
	}
	newpr, err := reconcile(tt.ctx, tt.asset, pr)
	if err != nil {
		t.Fatalf("Failed to get completed PipelineRun %s: %v", tt.pipelineRun.Name, err)
	}
	if diff := cmp.Diff(pr, newpr); diff != "" {
		t.Fatalf("Expected completed PipelineRun should remain unchanged when it has a results_id in annotations: %v", diff)
	}
}

func (tt *PipelineRunTest) testUpdatePipelineRun(t *testing.T) {
	pr, err := reconcile(tt.ctx, tt.asset, tt.pipelineRun)
	if err != nil {
		t.Fatalf("Failed to get completed PipelineRun %s: %v", tt.pipelineRun.Name, err)
	}
	pr.UID = "234435"
	_, err = tt.asset.Clients.Pipeline.TektonV1beta1().PipelineRuns(tt.pipelineRun.Namespace).Update(pr)
	if err != nil {
		t.Fatalf("Failed to update PipelineRun %s: %v", tt.pipelineRun.Name, err)
	}
	updatepr, err := reconcile(tt.ctx, tt.asset, pr)
	if err != nil {
		t.Fatalf("Failed to reconcile PipelineRun %s: %v", tt.pipelineRun.Name, err)
	}
	updatepr.ResourceVersion = pr.ResourceVersion
	if diff := cmp.Diff(pr, updatepr); diff != "" {
		t.Fatalf("Expected completed PipelineRun should be updated in cluster: %v", diff)
	}
	res, err := tt.client.GetResult(tt.ctx, &pb.GetResultRequest{Name: pr.Annotations[idName]})
	if err != nil {
		t.Fatalf("Expected completed PipelineRun %s not created in api server", tt.pipelineRun.Name)
	}
	p, err := convert.ToPipelineRunProto(updatepr)
	if err != nil {
		t.Fatalf("failed to convert to proto: %v", err)
	}
	want := &pb.Result{
		Name: pr.Annotations[idName],
		Executions: []*pb.Execution{{
			Execution: &pb.Execution_PipelineRun{p},
		}},
	}
	if diff := cmp.Diff(want, res, protocmp.Transform()); diff != "" {
		t.Fatalf("Expected completed PipelineRun should be upated in api server: %v", diff)
	}
}

func newResultsClient(t *testing.T) pb.ResultsClient {
	srv, err := server.SetupTestDB(t)
	if err != nil {
		t.Fatalf("Failed to create fake server: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterResultsServer(s, srv) // local test server
	lis, err := net.Listen("tcp", port)
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	go s.Serve(lis)
	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		t.Fatalf("did not connect: %v", err)
	}
	t.Cleanup(func() {
		lis.Close()
		s.Stop()
		conn.Close()
	})
	return pb.NewResultsClient(conn)
}

// getFakeClients create a fake client to send test data to reconciler
func getFakeClients(t *testing.T, pr []*v1beta1.PipelineRun, client pb.ResultsClient) (test.Assets, context.Context) {
	t.Helper()
	ctx, _ := ttesting.SetupFakeContext(t)
	d := test.Data{
		PipelineRuns: pr,
	}
	clients, _ := test.SeedTestData(t, ctx, d)
	cmw := configmap.NewInformedWatcher(clients.Kube, "")

	return test.Assets{
		Controller: NewController(ctx, cmw, client),
		Clients:    clients,
	}, ctx
}

func reconcile(ctx context.Context, asset test.Assets, pipelineRun *v1beta1.PipelineRun) (*v1beta1.PipelineRun, error) {
	c := asset.Controller
	clients := asset.Clients
	if err := c.Reconciler.Reconcile(ctx, pipelineRun.GetNamespacedName().String()); err != nil {
		return nil, err
	}
	pr, err := clients.Pipeline.TektonV1beta1().PipelineRuns(pipelineRun.Namespace).Get(pipelineRun.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return pr, err
}
