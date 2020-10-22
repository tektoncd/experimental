package reconciler

import (
	"context"
	"net"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	_ "github.com/mattn/go-sqlite3"
	"github.com/tektoncd/experimental/results/pkg/api/server"
	"github.com/tektoncd/experimental/results/pkg/watcher/convert"
	pb "github.com/tektoncd/experimental/results/proto/proto"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	ttesting "github.com/tektoncd/pipeline/pkg/reconciler/testing"
	test "github.com/tektoncd/pipeline/test"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/testing/protocmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/configmap"
)

const (
	port = ":0"
)

func getRunName(tr *v1beta1.TaskRun) string {
	return strings.Join([]string{tr.Namespace, tr.Name}, "/")
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

// reconcileTaskRun sends TaskRun data to reconciler and then retrieves completed TaskRun
func reconcileTaskRun(ctx context.Context, asset test.Assets, taskRun *v1beta1.TaskRun) (*v1beta1.TaskRun, error) {
	c := asset.Controller
	clients := asset.Clients
	if err := c.Reconciler.Reconcile(ctx, getRunName(taskRun)); err != nil {
		return nil, err
	}
	tr, err := clients.Pipeline.TektonV1beta1().TaskRuns(taskRun.Namespace).Get(taskRun.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return tr, nil
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

// TestReconcile tests if TaskRun in the client can be properly updated when sent to reconciler
func TestReconcile(t *testing.T) {
	client := newResultsClient(t)
	taskRun := &v1beta1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "Tekton",
			Namespace:   "default",
			Annotations: map[string]string{"demo": "demo"},
			UID:         "12345",
		},
	}

	asset, ctx := getFakeClients(t, []*v1beta1.TaskRun{taskRun}, client)
	// Create a new TaskRun in the fake client, completed TaskRun should be updated with a results id field in annotations.
	t.Run("test-create", func(t *testing.T) {
		tr, err := reconcileTaskRun(ctx, asset, taskRun)
		if err != nil {
			t.Fatalf("Failed to get completed TaskRun %s: %v", taskRun.Name, err)
		}
		if _, ok := tr.Annotations[idName]; !ok {
			t.Fatalf("Expected completed TaskRun %s should be updated with a results_id field in annotations", taskRun.Name)
		}
		if _, err := client.GetResult(ctx, &pb.GetResultRequest{Name: tr.Annotations[idName]}); err != nil {
			t.Fatalf("Expected completed TaskRun %s not created in api server", taskRun.Name)
		}
	})

	// Update a TaskRun with results id, completed TaskRun should remain unchanged.
	t.Run("test-unchange", func(t *testing.T) {
		tr, err := asset.Clients.Pipeline.TektonV1beta1().TaskRuns(taskRun.Namespace).Get(taskRun.Name, metav1.GetOptions{})
		if err != nil {
			t.Fatalf("Failed to get completed TaskRun %s: %v", taskRun.Name, err)
		}
		newtr, err := reconcileTaskRun(ctx, asset, tr)
		if err != nil {
			t.Fatalf("Failed to get completed TaskRun %s: %v", taskRun.Name, err)
		}
		if diff := cmp.Diff(tr, newtr); diff != "" {
			t.Fatalf("Expected completed TaskRun should remain unchanged when it has a results_id in annotations: %v", diff)
		}
	})

	// Update a TaskRun with new value and results id, completed TaskRun should be updated.
	t.Run("test-update", func(t *testing.T) {
		tr, err := reconcileTaskRun(ctx, asset, taskRun)
		if err != nil {
			t.Fatalf("Failed to get completed TaskRun %s: %v", taskRun.Name, err)
		}
		tr.UID = "234435"
		asset.Clients.Pipeline.TektonV1beta1().TaskRuns(taskRun.Namespace).Update(tr)
		updatetr, err := reconcileTaskRun(ctx, asset, tr)
		updatetr.ResourceVersion = tr.ResourceVersion
		if err != nil {
			t.Fatalf("Failed to get completed TaskRun %s: %v", taskRun.Name, err)
		}
		if diff := cmp.Diff(tr, updatetr); diff != "" {
			t.Fatalf("Expected completed TaskRun should be updated in cluster: %v", diff)
		}
		res, err := client.GetResult(ctx, &pb.GetResultRequest{Name: tr.Annotations[idName]})
		if err != nil {
			t.Fatalf("Expected completed TaskRun %s not created in api server", taskRun.Name)
		}
		p, err := convert.ToProto(updatetr)
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
	})
}
