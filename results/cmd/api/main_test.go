package main

import (
	"context"
	"database/sql"
	"io/ioutil"
	"os"
	"testing"
	"time"

	pb "github.com/tektoncd/experimental/results/proto/proto"
)

const (
	address = "localhost:50051"
)

// Test functionality of Server code
func TestCreateTaskRun(t *testing.T) {
	// Create a temporay database
	testDB, err := setupTestDB("testdb", t)
	if err != nil {
		t.Fatalf("failed to create temp file for db: %v", err)
	}

	// connect to fake server and do testing
	srv := &server{db: testDB}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	objectmeta := &pb.ObjectMeta{Uid: "123459", Name: "mytaskrun", Namespace: "default"}
	r, err := srv.CreateTaskRun(ctx, &pb.CreateTaskRunRequest{TaskRun: &pb.TaskRun{ApiVersion: "1", Metadata: objectmeta}})
	if err != nil {
		t.Fatalf("could not create taskrun: %v", err)
	}
	t.Logf("Created taskrun: %s", r.String())
}

// setupTestDB set up a temporary database for testing
func setupTestDB(dbName string, t *testing.T) (*sql.DB, error) {
	// Create a temporary file
	tmpfile, err := ioutil.TempFile("", "testdb")
	if err != nil {
		t.Fatalf("failed to create temp file for db: %v", err)
	}

	// Connect to sqlite DB.
	db, err := sql.Open("sqlite3", tmpfile.Name())
	if err != nil {
		t.Fatalf("failed to open the results.db: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
		os.Remove(tmpfile.Name())
	})

	// Create taskrun table
	statement, err := db.Prepare("CREATE TABLE IF NOT EXISTS taskrun (logid binary(16) PRIMARY KEY, taskrunlog BLOB, uid INTEGER, name TEXT, namespace TEXT)")
	if err != nil {
		t.Fatalf("failed to create taskrun table: %v", err)
	}
	if _, err := statement.Exec(); err != nil {
		t.Fatalf("failed to execute the taskrun table creation statement statement: %v", err)
	}
	return db, nil
}
