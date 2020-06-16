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
)

const (
	address = "localhost:50051"
)

// Test functionality of Server code
func TestCreateTaskRun(t *testing.T) {
	// Create a temporay database
	srv, err := setupTestDB("testdb", t)
	if err != nil {
		t.Fatalf("failed to create temp file for db: %v", err)
	}

	// connect to fake server and do testing
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := srv.CreateTaskRun(ctx, &pb.CreateTaskRunRequest{
		TaskRun: &pb.TaskRun{ApiVersion: "1",
			Metadata: &pb.ObjectMeta{
				Uid:       "123459",
				Name:      "mytaskrun",
				Namespace: "default"}}}); err != nil {
		t.Fatalf("could not create taskrun: %v", err)
	}
}

// setupTestDB set up a temporary database for testing
func setupTestDB(dbName string, t *testing.T) (*server, error) {
	// Create a temporary file
	tmpfile, err := ioutil.TempFile("", "testdb")
	if err != nil {
		t.Fatalf("failed to create temp file for db: %v", err)
	}

	// Connect to sqlite DB.
	db, err := sql.Open("sqlite3", tmpfile.Name())
	srv := &server{db: db}
	if err != nil {
		t.Fatalf("failed to open the results.db: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
		os.Remove(tmpfile.Name())
	})

	// Create taskrun table
	statement, err := db.Prepare("CREATE TABLE IF NOT EXISTS taskrun (logid binary(16) PRIMARY KEY, taskrunlog BLOB, uid TEXT, name TEXT, namespace TEXT)")
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
	srv, err := setupTestDB("testdb", t)
	if err != nil {
		t.Fatalf("failed to setup db: %v", err)
	}

	// Connect to fake server and create a new taskrun
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r, err := srv.CreateTaskRun(ctx, &pb.CreateTaskRunRequest{
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
	res, err := srv.GetTaskRun(ctx, &pb.GetTaskRunRequest{Uid: r.GetMetadata().GetUid()})
	if err != nil {
		t.Fatalf("could not get taskrun: %v", err)
	}
	if diff := cmp.Diff(r.String(), res.String()); diff != "" {
		t.Fatalf("could not get the same taskrun: %v", diff)
	}
}
func TestUpdateTaskRun(t *testing.T) {
	// Create a temporary database
	srv, err := setupTestDB("testdb", t)
	if err != nil {
		t.Fatalf("failed to create temp file for db: %v", err)
	}

	// Connect to fake server and create a taskrun
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	objectmeta := &pb.ObjectMeta{Uid: "123459", Name: "mytaskrun", Namespace: "default"}
	r, err := srv.CreateTaskRun(ctx, &pb.CreateTaskRunRequest{TaskRun: &pb.TaskRun{ApiVersion: "1", Metadata: objectmeta}})
	if err != nil {
		t.Fatalf("could not create taskrun: %v", err)
	}

	// Update the created taskrun
	objectmeta = &pb.ObjectMeta{Uid: "123459", Name: "ziweifan", Namespace: "tomorrow"}
	r, err = srv.UpdateTaskRun(ctx, &pb.UpdateTaskRunRequest{TaskRun: &pb.TaskRun{ApiVersion: "v1alpha1", Metadata: objectmeta}})
	if err != nil {
		t.Fatalf("could not update taskrun: %v", err)
	}

	// Validate by checking if we can get the updated taskrun
	rows, err := srv.db.Query("SELECT taskrunlog FROM taskrun WHERE uid = ?", r.GetMetadata().GetUid())
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
		if diff := cmp.Diff(taskrun.String(), r.String()); diff != "" {
			t.Fatalf("Update Function not properly implemented: %v", diff)
		}
	}
}
