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
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/google/uuid"
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

// CreateTaskRunResult receives CreateTaskRunRequest from clients and save it to local Sqlite Server.
func (s *server) CreateTaskRunResult(ctx context.Context, req *pb.CreateTaskRunRequest) (*pb.TaskRunResult, error) {
	database := s.db
	statement, err := database.Prepare("INSERT INTO taskrun (taskrunlog, results_id, name, namespace) VALUES (?, ?, ?, ?)")
	if err != nil {
		log.Printf("failed to insert a new taskrun: %v\n", err)
		return nil, fmt.Errorf("failed to insert a new taskrun: %w", err)
	}
	resultsID := uuid.New()

	// serialize data and insert it into database.
	taskrunFromClient := req.GetTaskRun()
	taskrunRes := pb.TaskRunResult{TaskRun: taskrunFromClient, ResultsId: resultsID.String()}
	blobData, err := proto.Marshal(taskrunFromClient)
	if err != nil {
		log.Println("taskrun marshaling error: ", err)
		return nil, fmt.Errorf("failed to marshal taskrun: %w", err)
	}
	taskrunMeta := taskrunFromClient.GetMetadata()
	if _, err := statement.Exec(blobData, resultsID, taskrunMeta.GetName(), taskrunMeta.GetNamespace()); err != nil {
		log.Printf("failed to execute insertion of a new taskrun: %v\n", err)
		return nil, status.Errorf(codes.AlreadyExists, "Try to create taskrun again")
	}
	return &taskrunRes, nil
}

// GetTaskRun received GetTaskRunRequest from users and return TaskRunResult back to users
func (s *server) GetTaskRunResult(ctx context.Context, req *pb.GetTaskRunRequest) (*pb.TaskRunResult, error) {
	resultsID, err := uuid.Parse(req.GetResultsId())
	if err != nil {
		log.Fatal("failed to parse resultID string into resultsID UUID", err)
		return nil, fmt.Errorf("failed to find a taskrun: %w", err)
	}
	rows, err := s.db.Query("SELECT taskrunlog FROM taskrun WHERE results_id = ?", resultsID)
	if err != nil {
		log.Fatalf("failed to query on database: %v", err)
		return nil, fmt.Errorf("failed to query on a taskrun: %w", err)
	}
	taskrun := &pb.TaskRun{}
	rowNum := 0
	for rows.Next() {
		var taskrunblob []byte
		rowNum++
		if rowNum >= 2 {
			log.Println("Warning: multiple rows found")
			break
		}
		rows.Scan(&taskrunblob)
		if err := proto.Unmarshal(taskrunblob, taskrun); err != nil {
			log.Fatal("unmarshaling error: ", err)
			return nil, fmt.Errorf("failed to unmarshal taskrun: %w", err)
		}
	}
	return &pb.TaskRunResult{TaskRun: taskrun, ResultsId: req.GetResultsId()}, nil
}

// UpdateTaskRun receives TaskRun and FieldMask from client and uses them to update records in local Sqlite Server.
func (s *server) UpdateTaskRunResult(ctx context.Context, req *pb.UpdateTaskRunRequest) (*pb.TaskRunResult, error) {
	// Update the entire row in database based on uid of taskrun.
	statement, err := s.db.Prepare("UPDATE taskrun SET name = ?, namespace = ?, taskrunlog = ? WHERE results_id = ?")
	if err != nil {
		log.Printf("failed to update a existing taskrun: %v\n", err)
		return nil, fmt.Errorf("failed to update a exsiting taskrun: %w", err)
	}
	taskrunFromClient := req.GetTaskRun()
	blobData, err := proto.Marshal(taskrunFromClient)
	if err != nil {
		log.Println("taskrun marshaling error: ", err)
		return nil, fmt.Errorf("taskrun marshaling error: %w", err)
	}
	taskrunMeta := taskrunFromClient.GetMetadata()
	if _, err := statement.Exec(taskrunMeta.GetName(), taskrunMeta.GetNamespace(), blobData, req.GetResultsId()); err != nil {
		log.Printf("failed to execute update of a new taskrun: %v\n", err)
		return nil, fmt.Errorf("failed to execute update of a new taskrun: %w", err)
	}
	return &pb.TaskRunResult{TaskRun: taskrunFromClient, ResultsId: req.GetResultsId()}, nil
}

// DeleteTaskRun receives DeleteTaskRun request from users and delete TaskRun in local Sqlite Server.
func (s *server) DeleteTaskRunResult(ctx context.Context, req *pb.DeleteTaskRunRequest) (*empty.Empty, error) {
	statement, err := s.db.Prepare("DELETE FROM taskrun WHERE results_id = ?")
	if err != nil {
		log.Fatalf("failed to create delete statement: %v", err)
		return nil, fmt.Errorf("failed to create delete statement: %w", err)
	}
	results, err := statement.Exec(req.GetResultsId())
	if err != nil {
		log.Fatalf("failed to execute delete statement: %v", err)
		return nil, fmt.Errorf("failed to execute delete statement: %w", err)
	}
	affect, err := results.RowsAffected()
	if err != nil {
		log.Fatalf("failed to retrieve results: %v", err)
		return nil, fmt.Errorf("failed to retrieve results: %w", err)
	}
	if affect == 0 {
		return nil, status.Errorf(codes.NotFound, "TaskRun not found")
	}
	return nil, nil
}
