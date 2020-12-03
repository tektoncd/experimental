package server

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/experimental/results/pkg/api/server/db/pagination"
	"github.com/tektoncd/experimental/results/pkg/api/server/test"
	pb "github.com/tektoncd/experimental/results/proto/v1alpha2/results_go_proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
)

var (
	// Used for deterministically increasing UUID generation.
	lastID = uint32(0)
)

func TestMain(m *testing.M) {
	uid = func() string {
		return fmt.Sprint(atomic.AddUint32(&lastID, 1))
	}
	os.Exit(m.Run())
}

func TestCreateResult(t *testing.T) {
	srv, err := New(test.NewDB(t))
	if err != nil {
		t.Fatalf("failed to create temp file for db: %v", err)
	}

	ctx := context.Background()
	req := &pb.CreateResultRequest{
		Parent: "foo",
		Result: &pb.Result{
			Name: "foo/results/bar",
		},
	}
	t.Run("success", func(t *testing.T) {
		got, err := srv.CreateResult(ctx, req)
		if err != nil {
			t.Fatalf("could not create result: %v", err)
		}
		want := proto.Clone(req.GetResult()).(*pb.Result)
		want.Id = fmt.Sprint(lastID)
		if diff := cmp.Diff(got, want, protocmp.Transform()); diff != "" {
			t.Errorf("-want, +got: %s", diff)
		}
	})

	// Errors
	for _, tc := range []struct {
		name string
		req  *pb.CreateResultRequest
		want codes.Code
	}{
		{
			name: "mismatched parent",
			req: &pb.CreateResultRequest{
				Parent: "foo",
				Result: &pb.Result{
					Name: "baz/results/bar",
				},
			},
			want: codes.InvalidArgument,
		},
		{
			name: "missing name",
			req: &pb.CreateResultRequest{
				Parent: "foo",
				Result: &pb.Result{},
			},
			want: codes.InvalidArgument,
		},
		{
			name: "already exists",
			req:  req,
			want: codes.AlreadyExists,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := srv.CreateResult(ctx, tc.req); status.Code(err) != tc.want {
				t.Fatalf("want: %v, got: %v - %+v", tc.want, status.Code(err), err)
			}
		})
	}
}

func TestGetResult(t *testing.T) {
	srv, err := New(test.NewDB(t))
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	ctx := context.Background()
	create, err := srv.CreateResult(ctx, &pb.CreateResultRequest{
		Parent: "foo",
		Result: &pb.Result{
			Name: "foo/results/bar",
		},
	})
	if err != nil {
		t.Fatalf("could not create result: %v", err)
	}

	get, err := srv.GetResult(ctx, &pb.GetResultRequest{Name: create.GetName()})
	if err != nil {
		t.Fatalf("could not get result: %v", err)
	}
	if diff := cmp.Diff(create, get, protocmp.Transform()); diff != "" {
		t.Errorf("-want, +got: %s", diff)
	}

	// Errors
	for _, tc := range []struct {
		name string
		req  *pb.GetResultRequest
		want codes.Code
	}{
		{
			name: "no name",
			req:  &pb.GetResultRequest{},
			want: codes.InvalidArgument,
		},
		{
			name: "not found",
			req:  &pb.GetResultRequest{Name: "a/results/doesnotexist"},
			want: codes.NotFound,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := srv.GetResult(ctx, tc.req); status.Code(err) != tc.want {
				t.Fatalf("want: %v, got: %v - %+v", tc.want, status.Code(err), err)
			}
		})
	}
}

func TestDeleteResult(t *testing.T) {
	srv, err := New(test.NewDB(t))
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	ctx := context.Background()
	r, err := srv.CreateResult(ctx, &pb.CreateResultRequest{
		Parent: "foo",
		Result: &pb.Result{
			Name: "foo/results/bar",
		},
	})
	if err != nil {
		t.Fatalf("could not create result: %v", err)
	}

	t.Run("success", func(t *testing.T) {
		// Delete inserted taskrun
		if _, err := srv.DeleteResult(ctx, &pb.DeleteResultRequest{Name: r.GetName()}); err != nil {
			t.Fatalf("could not delete taskrun: %v", err)
		}

		// Check if the taskrun is deleted
		if r, err := srv.GetResult(ctx, &pb.GetResultRequest{Name: r.GetName()}); err == nil {
			t.Fatalf("expected result to be deleted, got: %+v", r)
		}
	})

	t.Run("already deleted", func(t *testing.T) {
		// Check if a deleted taskrun can be deleted again
		if _, err := srv.DeleteResult(ctx, &pb.DeleteResultRequest{Name: r.GetName()}); status.Code(err) != codes.NotFound {
			t.Fatalf("expected NOT_FOUND, got: %v", err)
		}
	})
}

func TestListResults(t *testing.T) {
	// Reset so IDs match names
	lastID = 0

	// Create a temporary database
	srv, err := New(test.NewDB(t))
	if err != nil {
		t.Fatalf("failed to setup db: %v", err)
	}
	ctx := context.Background()

	parent := "foo"
	results := make([]*pb.Result, 0, 5)
	for i := 1; i <= cap(results); i++ {
		res, err := srv.CreateResult(ctx, &pb.CreateResultRequest{
			Parent: "foo",
			Result: &pb.Result{
				Name: fmt.Sprintf("%s/results/%d", parent, i),
			},
		})
		if err != nil {
			t.Fatalf("could not create result: %v", err)
		}
		t.Logf("Created name: %s, id: %s", res.GetName(), res.GetId())
		results = append(results, res)
	}

	tt := []struct {
		name   string
		req    *pb.ListResultsRequest
		want   *pb.ListResultsResponse
		status codes.Code
	}{
		{
			name: "list all",
			req: &pb.ListResultsRequest{
				Parent: parent,
			},
			want: &pb.ListResultsResponse{
				Results: results,
			},
			status: codes.OK,
		},
		{
			name: "list all w/ pagination token",
			req: &pb.ListResultsRequest{
				Parent:   parent,
				PageSize: int32(len(results)),
			},
			want: &pb.ListResultsResponse{
				Results: results,
			},
			status: codes.OK,
		},
		{
			name: "no results",
			req: &pb.ListResultsRequest{
				Parent: fmt.Sprintf("%s-doesnotexist", parent),
			},
			want:   &pb.ListResultsResponse{},
			status: codes.OK,
		},
		{
			name:   "missing parent",
			req:    &pb.ListResultsRequest{},
			status: codes.InvalidArgument,
		},
		{
			name: "simple query",
			req: &pb.ListResultsRequest{
				Parent: parent,
				Filter: `result.id == "1"`,
			},
			want: &pb.ListResultsResponse{
				Results: results[:1],
			},
		},
		{
			name: "simple query - function",
			req: &pb.ListResultsRequest{
				Parent: parent,
				Filter: `result.id.endsWith("1")`,
			},
			want: &pb.ListResultsResponse{
				Results: results[:1],
			},
		},
		{
			name: "complex query",
			req: &pb.ListResultsRequest{
				Parent: parent,
				Filter: `result.id == "1" || result.id == "2"`,
			},
			want: &pb.ListResultsResponse{
				Results: results[:2],
			},
		},
		{
			name: "filter all",
			req: &pb.ListResultsRequest{
				Parent: parent,
				Filter: `result.id == "doesnotexist"`,
			},
			want: &pb.ListResultsResponse{},
		},
		{
			name: "non-boolean expression",
			req: &pb.ListResultsRequest{
				Parent: parent,
				Filter: `result.id`,
			},
			status: codes.InvalidArgument,
		},
		{
			name: "wrong resource type",
			req: &pb.ListResultsRequest{
				Parent: parent,
				Filter: `taskrun.api_version != ""`,
			},
			status: codes.InvalidArgument,
		},
		{
			name: "partial response",
			req: &pb.ListResultsRequest{
				Parent:   parent,
				PageSize: 1,
			},
			want: &pb.ListResultsResponse{
				Results:       results[:1],
				NextPageToken: pagetoken(t, results[1].GetId(), ""),
			},
		},
		{
			name: "partial response with filter",
			req: &pb.ListResultsRequest{
				Parent:   parent,
				PageSize: 1,
				Filter:   `result.id > "1"`,
			},
			want: &pb.ListResultsResponse{
				Results:       results[1:2],
				NextPageToken: pagetoken(t, results[2].GetId(), `result.id > "1"`),
			},
		},
		{
			name: "with page token",
			req: &pb.ListResultsRequest{
				Parent:    parent,
				PageToken: pagetoken(t, results[1].GetId(), ""),
			},
			want: &pb.ListResultsResponse{
				Results: results[1:],
			},
		},
		{
			name: "with page token and filter and page size",
			req: &pb.ListResultsRequest{
				Parent:    parent,
				PageToken: pagetoken(t, results[1].GetId(), `result.id > "1"`),
				Filter:    `result.id > "1"`,
				PageSize:  1,
			},
			want: &pb.ListResultsResponse{
				Results:       results[1:2],
				NextPageToken: pagetoken(t, results[2].GetId(), `result.id > "1"`),
			},
		},
		{
			name: "invalid page size",
			req: &pb.ListResultsRequest{
				Parent:   parent,
				PageSize: -1,
			},
			status: codes.InvalidArgument,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			got, err := srv.ListResults(ctx, tc.req)
			if status.Code(err) != tc.status {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tc.want, got, protocmp.Transform()); diff != "" {
				t.Errorf("-want,+got: %s", diff)
			}
		})
	}
}

func pagetoken(t *testing.T, name, filter string) string {
	if token, err := pagination.EncodeToken(name, filter); err != nil {
		t.Fatalf("Failed to get encoded token: %v", err)
		return ""
	} else {
		return token
	}
}
