package common

import (
	"context"
	"encoding/json"
	"net"
	"testing"

	"github.com/tektoncd/experimental/results/pkg/api/server"
	pb "github.com/tektoncd/experimental/results/proto/proto"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/test"
	"gomodules.xyz/jsonpatch/v2"
	"google.golang.org/grpc"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	port = ":0"
)

const (
	Path   = "/metadata/annotations/results.tekton.dev~1id"
	IDName = "results.tekton.dev/id"
)

func NewResultsClient(t *testing.T) pb.ResultsClient {
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

// AnnotationPath creates a jsonpatch path used for adding results_id to Result
// annotations field.
func AnnotationPath(resultID string, path string, op string) ([]byte, error) {
	patches := []jsonpatch.JsonPatchOperation{{
		Operation: op,
		Path:      path,
		Value:     resultID,
	}}
	return json.Marshal(patches)
}

func ReconcileTaskRun(ctx context.Context, asset test.Assets, taskRun *v1beta1.TaskRun) (*v1beta1.TaskRun, error) {
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

func ReconcilePipelineRun(ctx context.Context, asset test.Assets, pipelineRun *v1beta1.PipelineRun) (*v1beta1.PipelineRun, error) {
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
