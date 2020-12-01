package test

import (
	"context"
	"net"
	"testing"

	server "github.com/tektoncd/experimental/results/pkg/api/server/v1alpha1"
	pb "github.com/tektoncd/experimental/results/proto/v1alpha1/results_go_proto"
	ttesting "github.com/tektoncd/pipeline/pkg/reconciler/testing"
	"github.com/tektoncd/pipeline/test"
	"google.golang.org/grpc"
	"knative.dev/pkg/configmap"
)

const (
	port = ":0"
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

func GetFakeClients(t *testing.T, d test.Data, client pb.ResultsClient) (context.Context, test.Clients, *configmap.InformedWatcher) {
	t.Helper()
	ctx, _ := ttesting.SetupFakeContext(t)
	clients, _ := test.SeedTestData(t, ctx, d)
	cmw := configmap.NewInformedWatcher(clients.Kube, "")
	return ctx, clients, cmw
}
