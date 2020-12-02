package server

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"github.com/google/uuid"
	dbmodel "github.com/tektoncd/experimental/results/pkg/api/server/db"
	ppb "github.com/tektoncd/experimental/results/proto/pipeline/v1beta1/pipeline_go_proto"
	pb "github.com/tektoncd/experimental/results/proto/v1alpha1/results_go_proto"
	mask "go.chromium.org/luci/common/proto/mask"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const (
	listResultsDefaultPageSize int32 = 50
	listResultsMaximumPageSize int32 = 10000
)

// Server with implementation of API server
type Server struct {
	pb.UnimplementedResultsServer
	env *cel.Env
	gdb *gorm.DB
	db  *sql.DB
}

// CreateResult receives CreateResultRequest from clients and save it to local Sqlite Server.
func (s *Server) CreateResult(ctx context.Context, req *pb.CreateResultRequest) (*pb.Result, error) {
	r := req.GetResult()
	name := uuid.New().String()
	r.Name = fmt.Sprintf("%s/results/%s", req.GetParent(), name)

	if etag, err := getResultETag(r); err != nil {
		log.Printf("etag generating error: %v", err)
		return nil, fmt.Errorf("failed to generate the etag for a result: %w", err)
	} else {
		r.Etag = etag
	}

	// serialize data and insert it into database.
	b, err := proto.Marshal(r)
	if err != nil {
		log.Printf("result marshaling error: %v", err)
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	// Slightly confusing since this is CreateResult, but this maps better to
	// Records in the v1alpha2 API, so store this as a Record for
	// compatibility.
	record := &dbmodel.Record{
		Parent: req.GetParent(),
		// TODO: Require Records to be nested in Results. Since v1alpha1
		// results ~= records, allow parent-less records for now to allow
		// clients to continue working.
		ResultID: "",
		ID:       name,
		// This should be the parent-less name, but allow for now for compatibility.
		Name: r.Name,
		Data: b,
	}
	if err := s.gdb.WithContext(ctx).Create(record).Error; err != nil {
		return nil, err
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

	// Check if there's any race condition problem.
	if prev.GetEtag() != r.GetEtag() {
		return nil, status.Error(codes.Aborted, "failed on the race condition, the content of this result has been modified.")
	}
	if etag, err := getResultETag(r); err != nil {
		log.Printf("etag generating error: %v", err)
		return nil, fmt.Errorf("failed to generate the etag for a result: %w", err)
	} else {
		r.Etag = etag
	}

	// Write result back to database.
	b, err := proto.Marshal(r)
	if err != nil {
		log.Println("result marshaling error: ", err)
		return nil, fmt.Errorf("result marshaling error: %w", err)
	}
	statement, err := s.db.Prepare("UPDATE records SET data = ? WHERE name = ?")
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
	statement, err := s.db.Prepare("DELETE FROM records WHERE name = ?")
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

	// checks and refines the pageSize
	pageSize := req.GetPageSize()
	if pageSize < 0 {
		return nil, status.Error(codes.InvalidArgument, "PageSize should be greater than 0")
	} else if pageSize == 0 {
		pageSize = listResultsDefaultPageSize
	} else if pageSize > listResultsMaximumPageSize {
		pageSize = listResultsMaximumPageSize
	}

	// retrieve the ListPageIdentifier from PageToken
	var pageIdentifier *pb.ListPageIdentifier
	pageToken := req.GetPageToken()
	if pageToken != "" {
		var err error
		if pageIdentifier, err = decodePageToken(pageToken); err != nil {
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid PageToken: %v", err))
		}
		if req.GetFilter() != pageIdentifier.GetFilter() {
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("use a different CEL `filter` from the last page."))
		}
	}

	var prg cel.Program
	var err error
	// return all results back to users if empty query is given
	if req.GetFilter() != "" {
		// filter from all results
		prg, err = s.env.Program(ast)
		if err != nil {
			log.Printf("program construction error: %s", err)
			return nil, status.Errorf(codes.InvalidArgument, "Error occurred during filter checking step, no Results found for the query string due to invalid field, invalid function to evaluate filter or missing double quotes around field value, please try to enter a query with correct type again: %v", err)
		}
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	// always request one more result to know whether next page exists.
	results, err := getFilteredPaginatedResults(tx, pageSize+1, pageIdentifier, prg)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to commit the query transaction: %v", err))
	}

	if int32(len(results)) > pageSize {
		// there exists next page, generate the nextPageToken, and drop the last one of the results.
		nextResult := results[len(results)-1]
		results := results[:len(results)-1]
		if nextPageToken, err := encodePageResult(&pb.ListPageIdentifier{ResultName: nextResult.GetName(), Filter: req.GetFilter()}); err == nil {
			return &pb.ListResultsResponse{Results: results, NextPageToken: nextPageToken}, nil
		}
	}
	return &pb.ListResultsResponse{Results: results}, nil
}

// Check if the result can be reserved.
func matchCelFilter(r *pb.Result, prg cel.Program) (bool, error) {
	if prg == nil {
		return true, nil
	}
	for _, e := range r.Executions {
		var (
			taskrun     ppb.TaskRun
			pipelinerun ppb.PipelineRun
		)
		if t := e.GetTaskRun(); t != nil {
			taskrun = *(t)
		}
		if p := e.GetPipelineRun(); p != nil {
			pipelinerun = *(p)
		}
		// We can't directly using e.GetTaskRun() and e.GetPipelineRun() here because the CEL doesn't work well with the nil pointer for proto types.
		out, _, err := prg.Eval(map[string]interface{}{
			"taskrun":     taskrun,
			"pipelinerun": pipelinerun,
		})
		if err != nil {
			log.Printf("failed to evaluate the expression: %v", err)
			return false, status.Errorf(codes.InvalidArgument, "Error occurred during filter evaluation step, no Results found for the query string due to invalid field, invalid function to evaluate filter or missing double quotes around field value, please try to enter a query with correct type again: %v", err)
		}
		if out.Value() == true {
			return true, nil
		}
	}
	return false, nil
}

// EncodePageResult encodes a ListPageIdentifier to PageToken
func encodePageResult(pi *pb.ListPageIdentifier) (token string, err error) {
	var tokenByte []byte
	if tokenByte, err = proto.Marshal(pi); err != nil {
		return "", err
	}
	encodedResult := make([]byte, base64.RawURLEncoding.EncodedLen(len(tokenByte)))
	base64.RawURLEncoding.Encode(encodedResult, tokenByte)
	return base64.RawURLEncoding.EncodeToString(encodedResult), nil
}

func decodePageToken(token string) (pi *pb.ListPageIdentifier, err error) {
	var encodedToken []byte
	if encodedToken, err = base64.RawURLEncoding.DecodeString(token); err != nil {
		return nil, err
	}
	tokenByte := make([]byte, base64.RawURLEncoding.DecodedLen(len(encodedToken)))
	if _, err = base64.RawURLEncoding.Decode(tokenByte, encodedToken); err != nil {
		return nil, err
	}
	pi = &pb.ListPageIdentifier{}
	if err = proto.Unmarshal(tokenByte, pi); err != nil {
		return nil, err
	}
	return pi, err
}

// GetFilteredPaginatedResults aims to obtain a fixed size `pageSize` of results from the database, starting
// from the results with the identifier `startPI`, filtered by a compiled CEL program `prg`.
//
// In this function, we query the database multiple times and filter the queried results to
// comprise the final results.
//
// To minimize the query times, we introduce a variable `ratio` to indicate the retention rate
// after filtering a batch of results. The ratio of the queried batch is:
//             ratio = remained_results_size/batch_size.
//
// The batchSize depends on the `ratio` of the previous batch and the `pageSize`:
//                  batchSize = pageSize/last_ratio
// The less the previous ratio is, the bigger the upcoming batch_size is. Then the queried time
// is significantly decreased.
func getFilteredPaginatedResults(tx *sql.Tx, pageSize int32, startPI *pb.ListPageIdentifier, prg cel.Program) (results []*pb.Result, err error) {
	var lastName string
	// for a queried batch, ratio = matchedNum/batchSize, where the `matchedNum` is the number of remained `results` after filtering.
	// we use the ratio of the current batch to dynamically determine the next batchSize, that is, nextBatchSize = pageSize/ratio.
	var ratio float32 = 1
	for int32(len(results)) < pageSize {
		// If didn't get enought results.
		var (
			batchSize    int32 // batchSize = math.Ceil(pageSize/ratio), has the same maximum value as `pageSize`.
			batchGot     int32 // less than batchSize, the size of the actually obtained records from a batch query.
			batchMatched int32 // less than batchGot, the size of results satisfying the condition that `prg` indicates.
		)
		if math.Ceil(float64(pageSize)/float64(ratio)) > float64(listResultsMaximumPageSize) {
			batchSize = listResultsMaximumPageSize
		} else {
			batchSize = int32(math.Ceil(float64(pageSize) / float64(ratio)))
		}
		var rows *sql.Rows
		if lastName == "" {
			if startPI != nil {
				rows, err = tx.Query("SELECT name, data FROM records WHERE name >= ? ORDER BY name LIMIT ? ", startPI.GetResultName(), batchSize)
			} else {
				rows, err = tx.Query("SELECT name, data FROM records ORDER BY name LIMIT ?", batchSize)
			}
		} else {
			rows, err = tx.Query("SELECT name, data FROM records WHERE name > ? ORDER BY name LIMIT ? ", lastName, batchSize)
		}
		if err != nil {
			log.Printf("failed to query on database: %v", err)
			return nil, status.Errorf(codes.Internal, "failed to query results: %v", err)
		}
		for rows.Next() {
			batchGot++
			var b []byte
			if err := rows.Scan(&lastName, &b); err != nil {
				log.Printf("failed to scan a row in query results: %v", err)
				return nil, status.Errorf(codes.Internal, "failed to read result data: %v", err)
			}
			r := &pb.Result{}
			if err := proto.Unmarshal(b, r); err != nil {
				log.Printf("unmarshaling error: %v", err)
				return nil, status.Errorf(codes.Internal, "failed to parse result data: %v", err)
			}
			// filter the results one by one
			if ok, _ := matchCelFilter(r, prg); ok {
				batchMatched++
				results = append(results, r)
				if int32(len(results)) >= pageSize {
					break
				}
			}
		}
		if batchGot < batchSize {
			// No more data in database.
			break
		}
		// update `ratio` to dynamically determine the `batchSize`
		if batchMatched != 0 && batchGot != 0 {
			ratio = float32(batchMatched) / float32(batchGot)
		}
	}
	return results, nil
}

// GetResultByID is the helper function to get a Result by results_id
func (s Server) getResultByID(name string) (*pb.Result, error) {
	rows, err := s.db.Query("SELECT data FROM records WHERE name = ?", name)
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

func getResultETag(result *pb.Result) (etag string, err error) {
	tmpEtag := result.GetEtag()
	result.Etag = ""
	if b, err := proto.Marshal(result); err == nil {
		dst := make([]byte, base64.RawURLEncoding.EncodedLen(len(b)))
		base64.RawURLEncoding.Encode(dst, b)
		etag = base64.RawURLEncoding.EncodeToString(dst)
	}
	result.Etag = tmpEtag
	return
}

// SetupTestDB set up a temporary database for testing
func SetupTestDB(t *testing.T) (*Server, error) {
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

	return New(gdb)
}

// New set up environment for the api server
func New(gdb *gorm.DB) (*Server, error) {
	env, err := cel.NewEnv(
		cel.Types(&pb.Result{}, &ppb.PipelineRun{}, &ppb.TaskRun{}),
		cel.Declarations(decls.NewIdent("taskrun", decls.NewObjectType("tekton.pipeline.v1beta1.TaskRun"), nil)),
		cel.Declarations(decls.NewIdent("pipelinerun", decls.NewObjectType("tekton.pipeline.v1beta1.PipelineRun"), nil)),
	)
	if err != nil {
		log.Fatalf("failed to create environment for filter: %v", err)
	}
	db, err := gdb.DB()
	if err != nil {
		return nil, err
	}
	srv := &Server{
		gdb: gdb,
		db:  db,
		env: env,
	}
	return srv, nil
}
