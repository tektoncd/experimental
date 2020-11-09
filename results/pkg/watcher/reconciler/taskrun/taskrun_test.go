package taskrun

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
	client := newResultsClient(t)
	taskRun := &v1beta1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "Tekton-TaskRun",
			Namespace:   "default",
			Annotations: map[string]string{"demo": "demo"},
			UID:         "12345",
		},
	}
	asset, ctx := getFakeClients(t, []*v1beta1.TaskRun{taskRun}, client)
	taskRunTest := TaskRunTest{
		taskRun: taskRun,
		asset:   asset,
		ctx:     ctx,
		client:  client,
	}
	return taskRunTest
}

func (tt *TaskRunTest) testCreateTaskRun(t *testing.T) {
	tr, err := reconcile(tt.ctx, tt.asset, tt.taskRun)
	if err != nil {
		t.Fatalf("Failed to get completed TaskRun %s: %v", tt.taskRun.Name, err)
	}
	if _, ok := tr.Annotations[idName]; !ok {
		t.Fatalf("Expected completed TaskRun %s should be updated with a results_id field in annotations", tt.taskRun.Name)
	}
	if _, err := tt.client.GetResult(tt.ctx, &pb.GetResultRequest{Name: tr.Annotations[idName]}); err != nil {
		t.Fatalf("Expected completed TaskRun %s not created in api server", tt.taskRun.Name)
	}
}

func (tt *TaskRunTest) testUnchangeTaskRun(t *testing.T) {
	tr, err := tt.asset.Clients.Pipeline.TektonV1beta1().TaskRuns(tt.taskRun.Namespace).Get(tt.taskRun.Name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get completed TaskRun %s: %v", tt.taskRun.Name, err)
	}
	newtr, err := reconcile(tt.ctx, tt.asset, tr)
	if err != nil {
		t.Fatalf("Failed to get completed TaskRun %s: %v", tt.taskRun.Name, err)
	}
	if diff := cmp.Diff(tr, newtr); diff != "" {
		t.Fatalf("Expected completed TaskRun should remain unchanged when it has a results_id in annotations: %v", diff)
	}
}

func (tt *TaskRunTest) testUpdateTaskRun(t *testing.T) {
	tr, err := reconcile(tt.ctx, tt.asset, tt.taskRun)
	if err != nil {
		t.Fatalf("Failed to get completed TaskRun %s: %v", tt.taskRun.Name, err)
	}
	tr.UID = "234435"
	_, err = tt.asset.Clients.Pipeline.TektonV1beta1().TaskRuns(tt.taskRun.Namespace).Update(tr)
	if err != nil {
		t.Fatalf("Failed to update TaskRun %s to Tekton Pipeline Client: %v", tt.taskRun.Name, err)
	}
	updatetr, err := reconcile(tt.ctx, tt.asset, tr)
	if err != nil {
		t.Fatalf("Failed to reconcile TaskRun %s: %v", tt.taskRun.Name, err)
	}
	updatetr.ResourceVersion = tr.ResourceVersion
	if diff := cmp.Diff(tr, updatetr); diff != "" {
		t.Fatalf("Expected completed TaskRun should be updated in cluster: %v", diff)
	}
	res, err := tt.client.GetResult(tt.ctx, &pb.GetResultRequest{Name: tr.Annotations[idName]})
	if err != nil {
		t.Fatalf("Expected completed TaskRun %s not created in api server", tt.taskRun.Name)
	}
	p, err := convert.ToTaskRunProto(updatetr)
	if err != nil {
		t.Fatalf("failed to convert to proto: %v", err)
	}
	want := &pb.Result{
		Name: tr.Annotations[idName],
		Executions: []*pb.Execution{{
			Execution: &pb.Execution_TaskRun{p},
		}},
	}
	if diff := cmp.Diff(want, res, protocmp.Transform()); diff != "" {
		t.Fatalf("Expected completed TaskRun should be upated in api server: %v", diff)
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
func getFakeClients(t *testing.T, tr []*v1beta1.TaskRun, client pb.ResultsClient) (test.Assets, context.Context) {
	t.Helper()
	ctx, _ := ttesting.SetupFakeContext(t)
	d := test.Data{
		TaskRuns: tr,
	}
	clients, _ := test.SeedTestData(t, ctx, d)
	cmw := configmap.NewInformedWatcher(clients.Kube, "")

	return test.Assets{
		Controller: NewController(ctx, cmw, client),
		Clients:    clients,
	}, ctx
}

func reconcile(ctx context.Context, asset test.Assets, taskRun *v1beta1.TaskRun) (*v1beta1.TaskRun, error) {
	c := asset.Controller
	clients := asset.Clients
	if err := c.Reconciler.Reconcile(ctx, taskRun.GetNamespacedName().String()); err != nil {
		return nil, err
	}
	tr, err := clients.Pipeline.TektonV1beta1().TaskRuns(taskRun.Namespace).Get(taskRun.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return tr, err
}
