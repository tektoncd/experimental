package test

import (
	"database/sql"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// NewDB set up a temporary database for testing
func NewDB(t *testing.T) *gorm.DB {
	t.Helper()

	// Create a temporary file
	tmpfile, err := ioutil.TempFile("", "testdb")
	if err != nil {
		t.Fatalf("failed to create temp file for db: %v", err)
	}
	t.Log("test database: ", tmpfile.Name())
	t.Cleanup(func() {
		tmpfile.Close()
		os.Remove(tmpfile.Name())
	})

	// Connect to sqlite DB manually to load in schema.
	db, err := sql.Open("sqlite3", tmpfile.Name())
	if err != nil {
		t.Fatalf("failed to open the results.db: %v", err)
	}
	defer db.Close()

	_, b, _, _ := runtime.Caller(0)
	basepath := filepath.Dir(b)
	schema, err := ioutil.ReadFile(path.Join(basepath, "../../../../schema/results.sql"))
	if err != nil {
		t.Fatalf("failed to read schema file: %v", err)
	}
	// Create result table using the checked in scheme to ensure compatibility.
	if _, err := db.Exec(string(schema)); err != nil {
		t.Fatalf("failed to execute the result table creation statement statement: %v", err)
	}

	// Reopen DB using gorm to use all the nice gorm tools.
	gdb, err := gorm.Open(sqlite.Open(tmpfile.Name()), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open the results.db: %v", err)
	}

	return gdb
}
