/*
Copyright 2020 The Tekton Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"

	"github.com/golang/protobuf/proto"
	_ "github.com/mattn/go-sqlite3"
	pb "github.com/tektoncd/experimental/results/proto/proto"
	"google.golang.org/grpc"
)

const port = ":50051"

func main() {
	// Connect to sqlite DB.
	db, err := sql.Open("sqlite3", "./results.db")
	if err != nil {
		log.Fatalf("failed to open the results.db: %v", err)
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

// CreateTaskRun receives TaskRun from Watcher and save it to local Sqlite Server.
func (s *server) CreateTaskRun(ctx context.Context, req *pb.CreateTaskRunRequest) (*pb.TaskRun, error) {
	database := s.db
	statement, err := database.Prepare("INSERT INTO taskrun (taskrunlog, uid, name, namespace) VALUES (?, ?, ?, ?)")
	if err != nil {
		log.Printf("failed to insert a new taskrun: %v\n", err)
		return nil, fmt.Errorf("failed to insert a new taskrun: %v", err)
	}

	// serialize data and insert it into database.
	taskrunFromClient := req.GetTaskRun()
	blobData, err := proto.Marshal(taskrunFromClient)
	if err != nil {
		log.Println("taskrun marshaling error: ", err)
		return nil, fmt.Errorf("failed to marshal taskrun: %v", err)
	}
	taskrunMeta := taskrunFromClient.GetMetadata()
	if _, err := statement.Exec(blobData, taskrunMeta.GetUid(), taskrunMeta.GetName(), taskrunMeta.GetNamespace()); err != nil {
		log.Printf("failed to execute insertion of a new taskrun: %v\n", err)
		return nil, fmt.Errorf("failed to excute insertion a new taskrun: %v", err)

	}
	return taskrunFromClient, nil
}
