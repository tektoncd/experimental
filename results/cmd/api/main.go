package main

import (
	"context"
	"database/sql"
	"log"
	"net"

	_ "github.com/mattn/go-sqlite3"
	pb "github.com/tektoncd/experimental/results/proto/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const port = ":50051"

func main() {
	// Connect to sqlite DB.
	db, err := sql.Open("sqlite3", "./results.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	srv := &server{db: db}

	// Listen for gRPC requests.
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterResultsServer(s, srv)
	log.Printf("Listening on %s...", port)
	log.Fatal(s.Serve(lis))
}

type server struct {
	pb.UnimplementedResultsServer

	db *sql.DB
}

func (s *server) InsertTaskRun(ctx context.Context, req *pb.InsertTaskRunRequest) (*pb.TaskRun, error) {
	// TODO: implement this.
	return nil, status.Errorf(codes.Unimplemented, "method InsertTaskRun not implemented")
}
