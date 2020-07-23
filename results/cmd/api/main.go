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
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	pb "github.com/tektoncd/experimental/results/proto/proto"
	mask "go.chromium.org/luci/common/proto/mask"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
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
	// Create cel enviroment for filter
	srv, err := new(db)
	// Listen for gRPC requests.
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterResultsServer(s, srv)
	reflection.Register(s)
	log.Printf("Listening on %s...", port)
	log.Fatal(s.Serve(lis))
}

type server struct {
	pb.UnimplementedResultsServer
	env *cel.Env
	db  *sql.DB
}

// CreateTaskRunResult receives CreateTaskRunRequest from clients and save it to local Sqlite Server.
func (s *server) CreateTaskRunResult(ctx context.Context, req *pb.CreateTaskRunRequest) (*pb.TaskRunResult, error) {
	statement, err := s.db.Prepare("INSERT INTO taskrun (taskrunlog, results_id, name, namespace) VALUES (?, ?, ?, ?)")
	if err != nil {
		log.Printf("failed to insert a new taskrun: %v", err)
		return nil, fmt.Errorf("failed to insert a new taskrun: %w", err)
	}
	resultsID := uuid.New()

	// serialize data and insert it into database.
	taskrunFromClient := req.GetTaskRun()
	taskrunRes := pb.TaskRunResult{TaskRun: taskrunFromClient, ResultsId: resultsID.String()}
	blobData, err := proto.Marshal(taskrunFromClient)
	if err != nil {
		log.Printf("taskrun marshaling error: %v", err)
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
	taskrun, err := s.getTaskRunByID(req.GetResultsId())
	if err != nil {
		return nil, fmt.Errorf("failed to find a taskrun: %w", err)
	}
	return &pb.TaskRunResult{TaskRun: taskrun, ResultsId: req.GetResultsId()}, nil
}

// UpdateTaskRun receives TaskRun and FieldMask from client and uses them to update records in local Sqlite Server.
func (s *server) UpdateTaskRunResult(ctx context.Context, req *pb.UpdateTaskRunRequest) (*pb.TaskRunResult, error) {
	// Find corresponding TaskRun in the database according to results_id.
	tx, err := s.db.Begin()
	if err != nil {
		log.Printf("failed to begin a transaction: %v", err)
		return nil, fmt.Errorf("failed to update a taskrun: %w", err)
	}
	updateTaskRun, err := s.getTaskRunByID(req.GetResultsId())
	if err != nil {
		return nil, fmt.Errorf("failed to find a taskrun: %w", err)
	}
	// Merge TaskRun from client into existing TaskRun based on fieldmask.
	taskrunFromClient := req.GetTaskRun()
	fieldMask := req.GetUpdateMask()
	// Update entire taskrun if user do not specify paths
	if fieldMask == nil {
		updateTaskRun = taskrunFromClient
	} else {
		msk, err := mask.FromFieldMask(fieldMask, updateTaskRun, false, true)
		// Return NotFound error to client field is invalid
		if err != nil {
			log.Printf("failed to convert fieldmask to mask: %v", err)
			return nil, status.Errorf(codes.NotFound, "field in fieldmask not found in taskrun")
		}
		if err := msk.Merge(taskrunFromClient, updateTaskRun); err != nil {
			log.Printf("failed to merge new taskrun into old taskrun: %v", err)
			return nil, fmt.Errorf("failed to update taskrun: %w", err)
		}
	}
	blobData, err := proto.Marshal(updateTaskRun)
	if err != nil {
		log.Println("taskrun marshaling error: ", err)
		return nil, fmt.Errorf("taskrun marshaling error: %w", err)
	}
	statement, err := s.db.Prepare("UPDATE taskrun SET name = ?, namespace = ?, taskrunlog = ? WHERE results_id = ?")
	if err != nil {
		log.Printf("failed to update a existing taskrun: %v", err)
		return nil, fmt.Errorf("failed to update a exsiting taskrun: %w", err)
	}
	taskrunMeta := updateTaskRun.GetMetadata()
	if _, err := statement.Exec(taskrunMeta.GetName(), taskrunMeta.GetNamespace(), blobData, req.GetResultsId()); err != nil {
		tx.Rollback()
		log.Printf("failed to execute update of a new taskrun: %v", err)
		return nil, fmt.Errorf("failed to execute update of a new taskrun: %w", err)
	}
	tx.Commit()
	return &pb.TaskRunResult{TaskRun: taskrunFromClient, ResultsId: req.GetResultsId()}, nil
}

// DeleteTaskRun receives DeleteTaskRun request from users and delete TaskRun in local Sqlite Server.
func (s *server) DeleteTaskRunResult(ctx context.Context, req *pb.DeleteTaskRunRequest) (*empty.Empty, error) {
	statement, err := s.db.Prepare("DELETE FROM taskrun WHERE results_id = ?")
	if err != nil {
		log.Printf("failed to create delete statement: %v", err)
		return nil, fmt.Errorf("failed to create delete statement: %w", err)
	}
	results, err := statement.Exec(req.GetResultsId())
	if err != nil {
		log.Printf("failed to execute delete statement: %v", err)
		return nil, fmt.Errorf("failed to execute delete statement: %w", err)
	}
	affect, err := results.RowsAffected()
	if err != nil {
		log.Printf("failed to retrieve results: %v", err)
		return nil, fmt.Errorf("failed to retrieve results: %w", err)
	}
	if affect == 0 {
		return nil, status.Errorf(codes.NotFound, "TaskRun not found")
	}
	return nil, nil
}

// ListTaskRunsResult receives a ListTaskRunRequest from users and return to users a list of TaskRuns according to the query
func (s *server) ListTaskRunsResult(ctx context.Context, req *pb.ListTaskRunsRequest) (*pb.ListTaskRunsResponse, error) {
	// Set up environment for cel and check if filter is empty string
	ast, issues := s.env.Compile(req.GetFilter())
	if issues != nil && issues.Err() != nil && req.GetFilter() != "" {
		log.Printf("type-check error: %s", issues.Err())
		return nil, status.Errorf(codes.InvalidArgument, "Error occurred during filter parse step, no TaskRuns found for the query string due to invalid field, invalid function to evaluate filter or missing double quotes around field value, please try to enter a query with correct type again: %v", issues.Err())
	}
	// get all taskruns from database
	rows, err := s.db.Query("SELECT taskrunlog FROM taskrun")
	if err != nil {
		log.Printf("failed to query on database: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to query results: %v", err)
	}
	var taskRunList []*pb.TaskRun
	for rows.Next() {
		var taskrunblob []byte
		if err := rows.Scan(&taskrunblob); err != nil {
			log.Printf("failed to scan a row in query results: %v", err)
			return nil, status.Errorf(codes.Internal, "failed to read result data: %v", err)
		}
		taskrun := &pb.TaskRun{}
		if err := proto.Unmarshal(taskrunblob, taskrun); err != nil {
			log.Printf("unmarshaling error: %v", err)
			return nil, status.Errorf(codes.Internal, "failed to parse result data: %v", err)
		}
		taskRunList = append(taskRunList, taskrun)
	}
	// return all taskruns back to users if empty query is given
	if req.GetFilter() == "" {
		return &pb.ListTaskRunsResponse{Items: taskRunList}, nil
	}
	// filter from all taskruns
	prg, err := s.env.Program(ast)
	if err != nil {
		log.Printf("program construction error: %s", err)
		return nil, status.Errorf(codes.InvalidArgument, "Error occurred during filter checking step, no TaskRuns found for the query string due to invalid field, invalid function to evaluate filter or missing double quotes around field value, please try to enter a query with correct type again: %v", err)
	}
	var resList []*pb.TaskRun
	for _, taskrun := range taskRunList {
		out, _, err := prg.Eval(map[string]interface{}{
			"taskrun": taskrun,
		})
		if err != nil {
			log.Printf("failed to evaluate the expression: %v", err)
			return nil, status.Errorf(codes.InvalidArgument, "Error occurred during filter evaluation step, no TaskRuns found for the query string due to invalid field, invalid function to evaluate filter or missing double quotes around field value, please try to enter a query with correct type again: %v", err)
		}
		if out.Value() == true {
			resList = append(resList, taskrun)
		}
	}
	return &pb.ListTaskRunsResponse{Items: resList}, nil
}

// GetTaskRunByID is the helper function to get a TaskRun by results_id
func (s *server) getTaskRunByID(id string) (*pb.TaskRun, error) {
	resultsID, err := uuid.Parse(id)
	if err != nil {
		log.Printf("failed to parse resultID string into resultsID UUID: %v", err)
		return nil, fmt.Errorf("failed to find a taskrun: %w", err)
	}
	rows, err := s.db.Query("SELECT taskrunlog FROM taskrun WHERE results_id = ?", resultsID)
	if err != nil {
		log.Printf("failed to query on database: %v", err)
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
			log.Printf("unmarshaling error: %v", err)
			return nil, fmt.Errorf("failed to unmarshal taskrun: %w", err)
		}
	}
	return taskrun, nil
}

// New set up environment for the api server
func new(db *sql.DB) (*server, error) {
	env, err := cel.NewEnv(
		cel.Types(&pb.TaskRun{}),
		cel.Declarations(decls.NewIdent("taskrun", decls.NewObjectType("tekton.TaskRun"), nil)),
	)
	if err != nil {
		log.Fatalf("failed to create environment for filter: %v", err)
	}
	srv := &server{db: db, env: env}
	return srv, nil
}
