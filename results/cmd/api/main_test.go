package main

import (
	"context"
	"database/sql"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/go-cmp/cmp"
	pb "github.com/tektoncd/experimental/results/proto/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	address = "localhost:50051"
)

// Test functionality of Server code
func TestCreateTaskRun(t *testing.T) {
	// Create a temporay database
	srv, err := setupTestDB(t)
	if err != nil {
		t.Fatalf("failed to create temp file for db: %v", err)
	}

	// connect to fake server and do testing
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := srv.CreateTaskRunResult(ctx, &pb.CreateTaskRunRequest{
		TaskRun: &pb.TaskRun{ApiVersion: "1",
			Metadata: &pb.ObjectMeta{
				Uid:       "123459",
				Name:      "mytaskrun",
				Namespace: "default"}}}); err != nil {
		t.Fatalf("could not create taskrun: %v", err)
	}
}

// setupTestDB set up a temporary database for testing
func setupTestDB(t *testing.T) (*server, error) {
	t.Helper()

	// Create a temporary file
	tmpfile, err := ioutil.TempFile("", "testdb")
	if err != nil {
		t.Fatalf("failed to create temp file for db: %v", err)
	}
	t.Cleanup(func() {
		os.Remove(tmpfile.Name())
	})

	// Connect to sqlite DB.
	db, err := sql.Open("sqlite3", tmpfile.Name())
	srv := &server{db: db}
	if err != nil {
		t.Fatalf("failed to open the results.db: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
	})

	schema, err := ioutil.ReadFile("results.sql")
	if err != nil {
		t.Fatalf("failed to read schema file: %v", err)
	}
	// Create taskrun table
	statement, err := db.Prepare(string(schema))
	if err != nil {
		t.Fatalf("failed to create taskrun table: %v", err)
	}
	if _, err := statement.Exec(); err != nil {
		t.Fatalf("failed to execute the taskrun table creation statement statement: %v", err)
	}
	return srv, nil
}

func TestGetTaskRun(t *testing.T) {
	// Create a temporary database
	srv, err := setupTestDB(t)
	if err != nil {
		t.Fatalf("failed to setup db: %v", err)
	}

	// Connect to fake server and create a new taskrun
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r, err := srv.CreateTaskRunResult(ctx, &pb.CreateTaskRunRequest{
		TaskRun: &pb.TaskRun{
			ApiVersion: "v1beta1",
			Metadata: &pb.ObjectMeta{
				Uid:       "31415926",
				Name:      "mytaskrun",
				Namespace: "default",
			}}})
	if err != nil {
		t.Fatalf("could not create taskrun: %v", err)
	}
	t.Logf("Created taskrun: %s", r.String())

	// Test if we can find inserted taskrun
	res, err := srv.GetTaskRunResult(ctx, &pb.GetTaskRunRequest{ResultsId: r.GetResultsId()})
	if err != nil {
		t.Fatalf("could not get taskrun: %v", err)
	}
	if diff := cmp.Diff(r.String(), res.String()); diff != "" {
		t.Fatalf("could not get the same taskrun: %v", diff)
	}
}

func TestUpdateTaskRun(t *testing.T) {
	// Create a temporary database
	srv, err := setupTestDB(t)
	if err != nil {
		t.Fatalf("failed to create temp file for db: %v", err)
	}

	// Connect to fake server and create a taskrun
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	objectmeta := &pb.ObjectMeta{Uid: "123459", Name: "mytaskrun", Namespace: "default"}
	r, err := srv.CreateTaskRunResult(ctx, &pb.CreateTaskRunRequest{TaskRun: &pb.TaskRun{ApiVersion: "1", Metadata: objectmeta}})
	if err != nil {
		t.Fatalf("could not create taskrun: %v", err)
	}

	// Update the created taskrun
	objectmeta = &pb.ObjectMeta{Uid: "123459", Name: "ziweifan", Namespace: "tomorrow"}
	r, err = srv.UpdateTaskRunResult(ctx, &pb.UpdateTaskRunRequest{TaskRun: &pb.TaskRun{ApiVersion: "v1alpha1", Metadata: objectmeta}, ResultsId: r.GetResultsId()})
	if err != nil {
		t.Fatalf("could not update taskrun: %v", err)
	}

	// Validate by checking if we can get the updated taskrun
	rows, err := srv.db.Query("SELECT taskrunlog FROM taskrun WHERE results_id = ?", r.GetResultsId())
	if err != nil {
		t.Fatalf("failed to query on database: %v", err)
	}
	for rows.Next() {
		var taskrunblob []byte
		taskrun := &pb.TaskRun{}
		rows.Scan(&taskrunblob)
		if err := proto.Unmarshal(taskrunblob, taskrun); err != nil {
			t.Fatal("unmarshaling error: ", err)
		}
		if diff := cmp.Diff(taskrun.String(), r.GetTaskRun().String()); diff != "" {
			t.Fatalf("Update Function not properly implemented: %v", diff)
		}
	}
}

func TestDeleteTaskRun(t *testing.T) {
	// Create a temporay database
	srv, err := setupTestDB(t)
	if err != nil {
		t.Fatalf("failed to create temp file for db: %v", err)
	}

	// Connect to fake server and insert a new taskrun
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r, err := srv.CreateTaskRunResult(ctx, &pb.CreateTaskRunRequest{TaskRun: &pb.TaskRun{
		ApiVersion: "1",
		Metadata: &pb.ObjectMeta{
			Uid:       "123459",
			Name:      "mytaskrun",
			Namespace: "default"}}})
	if err != nil {
		t.Fatalf("could not create taskrun: %v", err)
	}

	// Delete inserted taskrun
	if _, err := srv.DeleteTaskRunResult(ctx, &pb.DeleteTaskRunRequest{ResultsId: r.GetResultsId()}); err != nil {
		t.Fatalf("could not delete taskrun: %v", err)
	}

	// Check if the taskrun is deleted
	rows, err := srv.db.Query("SELECT taskrunlog FROM taskrun WHERE results_id = ?", r.GetResultsId())
	if err != nil {
		t.Fatalf("failed to query on database: %v", err)
	}
	if rows.Next() {
		t.Fatalf("failed to delete taskrun: %v", r.String())
	}

	// Check if a deleted taskrun can be delete again
	if _, err := srv.DeleteTaskRunResult(ctx, &pb.DeleteTaskRunRequest{ResultsId: r.GetResultsId()}); status.Code(err) != codes.NotFound {
		t.Fatalf("same taskrun not supposed to be deleted again: %v", r.String())
	}
}
