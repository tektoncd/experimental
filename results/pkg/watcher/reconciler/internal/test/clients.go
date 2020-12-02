package test

import (
	"context"
	"fmt"
	"net"
	"testing"

	"github.com/tektoncd/experimental/results/pkg/api/server/test"
	server "github.com/tektoncd/experimental/results/pkg/api/server/v1alpha1"
	pb "github.com/tektoncd/experimental/results/proto/v1alpha1/results_go_proto"
	ttesting "github.com/tektoncd/pipeline/pkg/reconciler/testing"
	pipelinetest "github.com/tektoncd/pipeline/test"
	"google.golang.org/grpc"
	"knative.dev/pkg/configmap"
)

const (
	port = ":0"
)

func NewResultsClient(t *testing.T) pb.ResultsClient {
	srv, err := server.New(test.NewDB(t))
	if err != nil {
		t.Fatalf("Failed to create fake server: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterResultsServer(s, srv) // local test server
	lis, err := net.Listen("tcp", port)
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	go func() {
		if err := s.Serve(lis); err != nil {
			fmt.Printf("error starting result server: %v\n", err)
		}
	}()
	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		t.Fatalf("did not connect: %v", err)
	}
	t.Cleanup(func() {
		s.Stop()
		lis.Close()
		conn.Close()
	})
	return pb.NewResultsClient(conn)
}

func GetFakeClients(t *testing.T, d pipelinetest.Data, client pb.ResultsClient) (context.Context, pipelinetest.Clients, *configmap.InformedWatcher) {
	t.Helper()
	ctx, _ := ttesting.SetupFakeContext(t)
	clients, _ := pipelinetest.SeedTestData(t, ctx, d)
	cmw := configmap.NewInformedWatcher(clients.Kube, "")
	return ctx, clients, cmw
}
