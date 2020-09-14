package server

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	pb "github.com/tektoncd/experimental/results/proto/proto"
	mask "go.chromium.org/luci/common/proto/mask"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server with implementation of API server
type Server struct {
	pb.UnimplementedResultsServer
	env *cel.Env
	db  *sql.DB
}

// CreateResult receives CreateResultRequest from clients and save it to local Sqlite Server.
func (s *Server) CreateResult(ctx context.Context, req *pb.CreateResultRequest) (*pb.Result, error) {
	r := req.GetResult()
	r.Name = fmt.Sprintf("%s/results/%s", req.GetParent(), uuid.New().String())

	// serialize data and insert it into database.
	b, err := proto.Marshal(r)
	if err != nil {
		log.Printf("result marshaling error: %v", err)
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	statement, err := s.db.Prepare("INSERT INTO results (name, blob) VALUES (?, ?)")
	if err != nil {
		log.Printf("failed to prepare insert: %v", err)
		return nil, fmt.Errorf("failed to insert a new result: %w", err)
	}
	if _, err := statement.Exec(r.GetName(), b); err != nil {
		log.Printf("failed to execute insertion of a new result: %v\n", err)
		return nil, fmt.Errorf("failed to add result: %w", err)
	}
	return r, nil
}

// GetResult received GetResultRequest from users and return Result back to users
func (s *Server) GetResult(ctx context.Context, req *pb.GetResultRequest) (*pb.Result, error) {
	r, err := s.getResultByID(req.GetName())
	if err != nil {
		return nil, fmt.Errorf("failed to find a result: %w", err)
	}
	return r, nil
}

// UpdateResult receives Result and FieldMask from client and uses them to update records in local Sqlite Server.
func (s Server) UpdateResult(ctx context.Context, req *pb.UpdateResultRequest) (*pb.Result, error) {
	// Find corresponding Result in the database according to results_id.
	tx, err := s.db.Begin()
	if err != nil {
		log.Printf("failed to begin a transaction: %v", err)
		return nil, fmt.Errorf("failed to update a result: %w", err)
	}

	prev, err := s.getResultByID(req.GetName())
	if err != nil {
		return nil, fmt.Errorf("failed to find a result: %w", err)
	}

	r := proto.Clone(prev).(*pb.Result)
	// Update entire result if user do not specify paths
	if req.GetUpdateMask() == nil {
		r = req.GetResult()
	} else {
		// Merge Result from client into existing Result based on fieldmask.
		msk, err := mask.FromFieldMask(req.GetUpdateMask(), r, false, true)
		// Return NotFound error to client field is invalid
		if err != nil {
			log.Printf("failed to convert fieldmask to mask: %v", err)
			return nil, status.Errorf(codes.NotFound, "field in fieldmask not found in result")
		}
		if err := msk.Merge(req.GetResult(), r); err != nil {
			log.Printf("failed to merge new result into old result: %v", err)
			return nil, fmt.Errorf("failed to update result: %w", err)
		}
	}

	// Do any most-mask validation to make sure we are not mutating any immutable fields.
	if r.GetName() != prev.GetName() {
		return prev, status.Error(codes.InvalidArgument, "result name cannot be changed")
	}
	if r.GetCreatedTime() != prev.GetCreatedTime() {
		return prev, status.Error(codes.InvalidArgument, "created time cannot be changed")
	}

	// Write result back to database.
	b, err := proto.Marshal(r)
	if err != nil {
		log.Println("result marshaling error: ", err)
		return nil, fmt.Errorf("result marshaling error: %w", err)
	}
	statement, err := s.db.Prepare("UPDATE results SET blob = ? WHERE name = ?")
	if err != nil {
		log.Printf("failed to update a existing result: %v", err)
		return nil, fmt.Errorf("failed to update a exsiting result: %w", err)
	}
	if _, err := statement.Exec(b, r.GetName()); err != nil {
		tx.Rollback()
		log.Printf("failed to execute update of a new result: %v", err)
		return nil, fmt.Errorf("failed to execute update of a new result: %w", err)
	}
	tx.Commit()
	return r, nil
}

// DeleteResult receives DeleteResult request from users and delete Result in local Sqlite Server.
func (s Server) DeleteResult(ctx context.Context, req *pb.DeleteResultRequest) (*empty.Empty, error) {
	statement, err := s.db.Prepare("DELETE FROM results WHERE name = ?")
	if err != nil {
		log.Printf("failed to create delete statement: %v", err)
		return nil, fmt.Errorf("failed to create delete statement: %w", err)
	}
	results, err := statement.Exec(req.GetName())
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
		return nil, status.Errorf(codes.NotFound, "Result not found")
	}
	return nil, nil
}

// ListResultsResult receives a ListResultRequest from users and return to users a list of Results according to the query
func (s *Server) ListResultsResult(ctx context.Context, req *pb.ListResultsRequest) (*pb.ListResultsResponse, error) {
	// Set up environment for cel and check if filter is empty string
	ast, issues := s.env.Compile(req.GetFilter())
	if issues != nil && issues.Err() != nil && req.GetFilter() != "" {
		log.Printf("type-check error: %s", issues.Err())
		return nil, status.Errorf(codes.InvalidArgument, "Error occurred during filter parse step, no Results found for the query string due to invalid field, invalid function to evaluate filter or missing double quotes around field value, please try to enter a query with correct type again: %v", issues.Err())
	}
	// get all results from database
	rows, err := s.db.Query("SELECT blob FROM results")
	if err != nil {
		log.Printf("failed to query on database: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to query results: %v", err)
	}
	var results []*pb.Result
	for rows.Next() {
		var b []byte
		if err := rows.Scan(&b); err != nil {
			log.Printf("failed to scan a row in query results: %v", err)
			return nil, status.Errorf(codes.Internal, "failed to read result data: %v", err)
		}
		r := &pb.Result{}
		if err := proto.Unmarshal(b, r); err != nil {
			log.Printf("unmarshaling error: %v", err)
			return nil, status.Errorf(codes.Internal, "failed to parse result data: %v", err)
		}
		results = append(results, r)
	}

	// return all results back to users if empty query is given
	if req.GetFilter() == "" {
		return &pb.ListResultsResponse{Results: results}, nil
	}

	// filter from all results
	prg, err := s.env.Program(ast)
	if err != nil {
		log.Printf("program construction error: %s", err)
		return nil, status.Errorf(codes.InvalidArgument, "Error occurred during filter checking step, no Results found for the query string due to invalid field, invalid function to evaluate filter or missing double quotes around field value, please try to enter a query with correct type again: %v", err)
	}
	var resp []*pb.Result
	for _, r := range results {
		for _, e := range r.Executions {
			out, _, err := prg.Eval(map[string]interface{}{
				"taskrun": e.GetTaskRun(),
			})
			if err != nil {
				log.Printf("failed to evaluate the expression: %v", err)
				return nil, status.Errorf(codes.InvalidArgument, "Error occurred during filter evaluation step, no Results found for the query string due to invalid field, invalid function to evaluate filter or missing double quotes around field value, please try to enter a query with correct type again: %v", err)
			}
			if out.Value() == true {
				resp = append(resp, r)
			}
		}
	}
	return &pb.ListResultsResponse{Results: resp}, nil
}

// GetResultByID is the helper function to get a Result by results_id
func (s Server) getResultByID(name string) (*pb.Result, error) {

	rows, err := s.db.Query("SELECT blob FROM results WHERE name = ?", name)
	if err != nil {
		log.Printf("failed to query on database: %v", err)
		return nil, fmt.Errorf("failed to query on a result: %w", err)
	}
	result := &pb.Result{}
	rowNum := 0
	for rows.Next() {
		var b []byte
		rowNum++
		if rowNum >= 2 {
			log.Println("Warning: multiple rows found")
			break
		}
		rows.Scan(&b)
		if err := proto.Unmarshal(b, result); err != nil {
			log.Printf("unmarshaling error: %v", err)
			return nil, fmt.Errorf("failed to unmarshal result: %w", err)
		}
	}
	if rowNum == 0 {
		return nil, status.Error(codes.NotFound, "result not found")
	}
	return result, nil
}

// SetupTestDB set up a temporary database for testing
func SetupTestDB(t *testing.T) (*Server, error) {
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
	if err != nil {
		t.Fatalf("failed to open the results.db: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
	})
	schema, err := ioutil.ReadFile("../../cmd/api/results.sql")
	if err != nil {
		t.Fatalf("failed to read schema file: %v", err)
	}
	// Create result table
	statement, err := db.Prepare(string(schema))
	if err != nil {
		t.Fatalf("failed to create result table: %v", err)
	}
	if _, err := statement.Exec(); err != nil {
		t.Fatalf("failed to execute the result table creation statement statement: %v", err)
	}
	return New(db)
}

// New set up environment for the api server
func New(db *sql.DB) (*Server, error) {
	env, err := cel.NewEnv(
		cel.Types(&pb.Result{}),
		cel.Declarations(decls.NewIdent("taskrun", decls.NewObjectType("tekton.pipeline.v1.TaskRun"), nil)),
	)
	if err != nil {
		log.Fatalf("failed to create environment for filter: %v", err)
	}
	srv := &Server{db: db, env: env}
	return srv, nil
}
