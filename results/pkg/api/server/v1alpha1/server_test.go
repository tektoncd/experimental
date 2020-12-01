package server

import (
	"context"
	"fmt"
	"sort"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/experimental/results/pkg/api/server/test"
	ppb "github.com/tektoncd/experimental/results/proto/pipeline/v1beta1/pipeline_go_proto"
	pb "github.com/tektoncd/experimental/results/proto/v1alpha1/results_go_proto"
	"google.golang.org/genproto/protobuf/field_mask"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/testing/protocmp"
)

// Test functionality of Server code
func TestCreateResult(t *testing.T) {
	// Create a temporay database
	srv, err := New(test.NewDB(t))
	if err != nil {
		t.Fatalf("failed to create temp file for db: %v", err)
	}

	// connect to fake server and do testing
	ctx := context.Background()
	if _, err := srv.CreateResult(ctx, &pb.CreateResultRequest{
		Result: Result(&ppb.TaskRun{
			ApiVersion: "1",
			Metadata: &ppb.ObjectMeta{
				Uid:       "123459",
				Name:      "mytaskrun",
				Namespace: "default",
			}},
		)}); err != nil {
		t.Fatalf("could not create taskrun: %v", err)
	}
}

func TestGetResult(t *testing.T) {
	// Create a temporary database
	srv, err := New(test.NewDB(t))
	if err != nil {
		t.Fatalf("failed to setup db: %v", err)
	}
	ctx := context.Background()
	// Connect to fake server and create a new taskrun
	r, err := srv.CreateResult(ctx, &pb.CreateResultRequest{
		Result: Result(&ppb.TaskRun{
			ApiVersion: "v1beta1",
			Metadata: &ppb.ObjectMeta{
				Uid:       "31415926",
				Name:      "mytaskrun",
				Namespace: "default",
			}},
		),
	})
	if err != nil {
		t.Fatalf("could not create taskrun: %v", err)
	}

	// Test if we can find inserted taskrun
	res, err := srv.GetResult(ctx, &pb.GetResultRequest{Name: r.GetName()})
	if err != nil {
		t.Fatalf("could not get taskrun: %v", err)
	}
	if diff := cmp.Diff(r.String(), res.String()); diff != "" {
		t.Fatalf("could not get the same taskrun: %v", diff)
	}
}

func TestUpdateResult(t *testing.T) {
	// Create a temporary database
	srv, err := New(test.NewDB(t))
	if err != nil {
		t.Fatalf("failed to create temp file for db: %v", err)
	}
	ctx := context.Background()

	// Validate by checking if output equals expected Result
	tt := []struct {
		name      string
		in        *pb.Result
		fieldmask *field_mask.FieldMask
		update    *pb.Result
		expect    *pb.Result
		err       bool
	}{
		{
			in:   &pb.Result{},
			name: "Test no Mask",
			update: Result(&ppb.TaskRun{
				ApiVersion: "v1beta1",
				Metadata: &ppb.ObjectMeta{
					Uid:       "123456",
					Name:      "newtaskrun",
					Namespace: "tekton",
				},
			}),
			// update entire taskrun
			expect: Result(&ppb.TaskRun{
				ApiVersion: "v1beta1",
				Metadata: &ppb.ObjectMeta{
					Uid:       "123456",
					Name:      "newtaskrun",
					Namespace: "tekton",
				},
			}),
		},
		{
			in:        &pb.Result{},
			name:      "Test partial Mask",
			fieldmask: &field_mask.FieldMask{Paths: []string{"annotations"}},
			update:    &pb.Result{Annotations: map[string]string{"foo": "bar"}},
			// update fields in fieldmask only
			expect: &pb.Result{Annotations: map[string]string{"foo": "bar"}},
		},
		{
			in: Result(&ppb.TaskRun{
				ApiVersion: "v1alpha1",
				Metadata: &ppb.ObjectMeta{
					Uid:       "31415926",
					Name:      "mytaskrun",
					Namespace: "default",
				},
			}),
			name:      "Test Mask with excess field",
			fieldmask: &field_mask.FieldMask{Paths: []string{"annotations", "executions"}},
			// unset field value to default value in fieldmask
			update: &pb.Result{Annotations: map[string]string{"foo": "bar"}},
			expect: &pb.Result{Annotations: map[string]string{"foo": "bar"}},
		},
		{
			in:        &pb.Result{},
			name:      "Test Mask with empty field",
			fieldmask: &field_mask.FieldMask{Paths: []string{}},
			// do not update
			update: &pb.Result{Annotations: map[string]string{"foo": "bar"}},
			expect: &pb.Result{},
		},
		{
			in: Result(&ppb.TaskRun{
				Metadata: &ppb.ObjectMeta{
					Name: "foo",
				},
			}),
			name:      "Test Mask updating repeated fields",
			fieldmask: &field_mask.FieldMask{Paths: []string{"executions"}},
			// update entire repeated field(all elements in array) - standard update
			update: Result(&ppb.TaskRun{
				Metadata: &ppb.ObjectMeta{
					Name: "bar",
				},
			}),
			expect: Result(&ppb.TaskRun{
				Metadata: &ppb.ObjectMeta{
					Name: "bar",
				},
			}),
		},
		{
			in:        &pb.Result{},
			name:      "Test Mask with nil Paths field",
			fieldmask: &field_mask.FieldMask{},
			// do not update
			update: &pb.Result{Annotations: map[string]string{"foo": "bar"}},
			expect: &pb.Result{},
		},

		// Errors
		{
			in:        &pb.Result{},
			name:      "ERR Test Mask with invalid field",
			fieldmask: &field_mask.FieldMask{Paths: []string{"annotations", "invalid_field"}},
			// do not update
			update: &pb.Result{Annotations: map[string]string{"foo": "bar"}},
			err:    true,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			r, err := srv.CreateResult(ctx, &pb.CreateResultRequest{Result: tc.in})
			if err != nil {
				t.Fatalf("could not create taskrun: %v", err)
			}

			// If we're doing a full update, pass through immutable fields to
			// the update result. Since these are created dynamicly,
			// we can't prepopulate these.
			if tc.fieldmask == nil {
				tc.update.Name = r.GetName()
				tc.update.CreatedTime = r.GetCreatedTime()
			}

			// Update the created taskrun
			r, err = srv.UpdateResult(ctx, &pb.UpdateResultRequest{Result: tc.update, Name: r.GetName(), UpdateMask: tc.fieldmask})
			if err != nil {
				if tc.err {
					return
				}
				t.Fatalf("could not update taskrun: %v, %v", err, status.Code(err))
			}

			// Expected results should always match the created result.
			tc.expect.Name = r.GetName()
			tc.expect.CreatedTime = r.GetCreatedTime()
			got, err := srv.GetResult(ctx, &pb.GetResultRequest{Name: r.GetName()})
			if err != nil {
				t.Fatalf("GetResult: %v", err)
			}
			if diff := cmp.Diff(tc.expect, got, protocmp.Transform()); diff != "" {
				t.Fatalf("-want, +got: %s", diff)
			}
		})
	}
}

func TestDeleteResult(t *testing.T) {
	// Create a temporay database
	srv, err := New(test.NewDB(t))
	if err != nil {
		t.Fatalf("failed to create temp file for db: %v", err)
	}
	// Connect to fake server and insert a new taskrun
	ctx := context.Background()
	r, err := srv.CreateResult(ctx, &pb.CreateResultRequest{
		Result: Result(&ppb.TaskRun{
			ApiVersion: "1",
			Metadata: &ppb.ObjectMeta{
				Uid:       "123459",
				Name:      "mytaskrun",
				Namespace: "default",
			},
		}),
	})
	if err != nil {
		t.Fatalf("could not create taskrun: %v", err)
	}

	// Delete inserted taskrun
	if _, err := srv.DeleteResult(ctx, &pb.DeleteResultRequest{Name: r.GetName()}); err != nil {
		t.Fatalf("could not delete taskrun: %v", err)
	}

	// Check if the taskrun is deleted
	if r, err := srv.GetResult(ctx, &pb.GetResultRequest{Name: r.GetName()}); err == nil {
		t.Fatalf("expected result to be deleted, got: %+v", r)
	}

	// Check if a deleted taskrun can be deleted again
	if _, err := srv.DeleteResult(ctx, &pb.DeleteResultRequest{Name: r.GetName()}); status.Code(err) != codes.NotFound {
		t.Fatalf("same taskrun not supposed to be deleted again: %v", r.String())
	}
}

func TestListResults(t *testing.T) {
	// Create a temporary database
	srv, err := New(test.NewDB(t))
	if err != nil {
		t.Fatalf("failed to setup db: %v", err)
	}
	ctx := context.Background()

	// Create a bunch of taskruns for testing
	t1 := &ppb.TaskRun{
		ApiVersion: "v1beta1",
		Metadata: &ppb.ObjectMeta{
			Uid:       "00000001",
			Name:      "taskrun",
			Namespace: "default",
		},
	}
	t2 := &ppb.TaskRun{
		ApiVersion: "v1alpha1",
		Metadata: &ppb.ObjectMeta{
			Uid:       "00000002",
			Name:      "task",
			Namespace: "default",
		},
	}
	t3 := &ppb.TaskRun{
		ApiVersion: "v1beta1",
		Metadata: &ppb.ObjectMeta{
			Uid:       "00000003",
			Name:      "mytaskrun",
			Namespace: "demo",
		},
	}
	t4 := &ppb.TaskRun{
		ApiVersion: "v1beta1",
		Metadata: &ppb.ObjectMeta{
			Uid:       "00000004",
			Name:      "newtaskrun",
			Namespace: "demo",
		},
	}
	t5 := &ppb.TaskRun{
		ApiVersion: "v1alpha1",
		Metadata: &ppb.ObjectMeta{
			Uid:       "00000005",
			Name:      "newtaskrun",
			Namespace: "official",
		},
	}
	t6 := &ppb.PipelineRun{
		ApiVersion: "v1beta1",
		Metadata: &ppb.ObjectMeta{
			Uid:       "00000006",
			Name:      "pipelinerun",
			Namespace: "default",
		},
	}
	t7 := &ppb.PipelineRun{
		ApiVersion: "v1alpha1",
		Metadata: &ppb.ObjectMeta{
			Uid:       "00000007",
			Name:      "pipeline",
			Namespace: "demo",
		},
	}

	results := []*pb.Result{Result(t1), Result(t2), Result(t3), Result(t4), Result(t5), Result(t6), Result(t7), Result(t4, t5, t6, t7)}
	gotResults := []*pb.Result{}
	for _, r := range results {
		res, err := srv.CreateResult(ctx, &pb.CreateResultRequest{
			Result: r,
		})
		if err != nil {
			t.Fatalf("could not create result: %v", err)
		}
		gotResults = append(gotResults, res)
	}
	sortResults := func(results ...[]*pb.Result) {
		for _, rs := range results {
			sort.Slice(rs, func(i, j int) bool { return rs[i].Name < rs[j].Name })
		}
	}
	tkrV1beta1Results := []*pb.Result{gotResults[0], gotResults[2], gotResults[3], gotResults[7]}
	plrV1beta1Results := []*pb.Result{gotResults[5], gotResults[7]}
	taskRunResults := []*pb.Result{gotResults[0], gotResults[2], gotResults[3], gotResults[4], gotResults[7]}
	pipelineRunResults := []*pb.Result{gotResults[5], gotResults[6], gotResults[7]}
	mixedQueryResults := []*pb.Result{gotResults[0], gotResults[2], gotResults[3], gotResults[5], gotResults[7]}

	sortResults(tkrV1beta1Results, plrV1beta1Results, taskRunResults, pipelineRunResults, mixedQueryResults)
	tt := []struct {
		name          string
		filter        string
		pageSize      int32
		pageToken     string
		nextPageToken string
		nextPageName  string
		expect        []*pb.Result
		expectStatus  codes.Code
	}{
		{
			name:         "test query taskrun",
			filter:       `taskrun.api_version=="v1beta1"`,
			expect:       tkrV1beta1Results,
			expectStatus: codes.OK,
		},
		{
			name:         "test query pipelinerun",
			filter:       `pipelinerun.api_version=="v1beta1"`,
			expect:       plrV1beta1Results,
			expectStatus: codes.OK,
		},
		{
			name:         "test query taskrun with simple function",
			filter:       `taskrun.metadata.name.endsWith("run")`,
			expect:       taskRunResults,
			expectStatus: codes.OK,
		},
		{
			name:         "test query pipelinerun with simple function",
			filter:       `pipelinerun.metadata.name.startsWith("pipeline")`,
			expect:       pipelineRunResults,
			expectStatus: codes.OK,
		},
		{
			name:         "test query in a mixed way",
			filter:       `pipelinerun.api_version=="v1beta1" || taskrun.api_version=="v1beta1"`,
			expect:       mixedQueryResults,
			expectStatus: codes.OK,
		},
		{
			name:         "test empty filter",
			filter:       "",
			expect:       gotResults,
			expectStatus: codes.OK,
		},
		{
			name:         "test invalid field",
			filter:       `task.name=="newtaskrun"`,
			expectStatus: codes.InvalidArgument,
		},
		{
			name:         "test invalid value type with no double quotes around value",
			filter:       `taskrun.name==newtaskrun`,
			expectStatus: codes.InvalidArgument,
		},
		{
			name:         "test value not existing in the server record",
			filter:       `taskrun.metadata.name=="notaskrun"`,
			expect:       nil,
			expectStatus: codes.OK,
		},
		{
			name:         "test invalid field outside of our defined top level field",
			filter:       `taskrun.unexistfield=="notaskrun"`,
			expectStatus: codes.InvalidArgument,
		},
		{
			name:         "test valid field but not boolean experssion",
			filter:       `taskrun.api_version"`,
			expectStatus: codes.InvalidArgument,
		},
		{
			name:         "test random word input",
			filter:       `tekton`,
			expectStatus: codes.InvalidArgument,
		},
		{
			name:         "test if field is case sensitive",
			filter:       `taskrun.MetaData.name=="notaskrun"`,
			expectStatus: codes.InvalidArgument,
		},
		{
			name:         "test query with invalid pagesize",
			filter:       `taskrun.api_version=="v1beta1"`,
			expectStatus: codes.InvalidArgument,
			pageSize:     -2,
		},
		{
			name:         "test query with invalid pagetoken, can't be decoded",
			filter:       `taskrun.api_version=="v1beta1"`,
			expectStatus: codes.InvalidArgument,
			pageToken:    "invalid_token",
		},
		{
			name:         "test query with invalid pagetoken, decoded filter mismatched",
			filter:       `taskrun.api_version=="v1beta1"`,
			expectStatus: codes.InvalidArgument,
			pageToken:    getEncodedPageToken(t, &pb.ListPageIdentifier{ResultName: tkrV1beta1Results[2].GetName(), Filter: `taskrun.api_version=="v1"`}),
		},
		{
			name:          "test query with pagesize",
			filter:        `taskrun.api_version=="v1beta1"`,
			expect:        []*pb.Result{tkrV1beta1Results[0]},
			expectStatus:  codes.OK,
			pageSize:      1,
			nextPageToken: getEncodedPageToken(t, &pb.ListPageIdentifier{ResultName: tkrV1beta1Results[1].GetName(), Filter: `taskrun.api_version=="v1beta1"`}),
			nextPageName:  tkrV1beta1Results[1].GetName(),
		},
		{
			name:          "test query with pagesize and pagetoken",
			filter:        `taskrun.api_version=="v1beta1"`,
			expect:        []*pb.Result{tkrV1beta1Results[1]},
			expectStatus:  codes.OK,
			pageSize:      1,
			pageToken:     getEncodedPageToken(t, &pb.ListPageIdentifier{ResultName: tkrV1beta1Results[1].GetName(), Filter: `taskrun.api_version=="v1beta1"`}),
			nextPageToken: getEncodedPageToken(t, &pb.ListPageIdentifier{ResultName: tkrV1beta1Results[2].GetName(), Filter: `taskrun.api_version=="v1beta1"`}),
			nextPageName:  tkrV1beta1Results[2].GetName(),
		},
		{
			name:          "test query with pagesize and pagetoken, no more next page",
			filter:        `taskrun.api_version=="v1beta1"`,
			expect:        []*pb.Result{tkrV1beta1Results[1], tkrV1beta1Results[2], tkrV1beta1Results[3]},
			expectStatus:  codes.OK,
			pageSize:      3,
			pageToken:     getEncodedPageToken(t, &pb.ListPageIdentifier{ResultName: tkrV1beta1Results[1].GetName(), Filter: `taskrun.api_version=="v1beta1"`}),
			nextPageToken: "",
		},
		{
			name:          "test query with pagesize and pagetoken, the last page, got insufficient pages",
			filter:        `taskrun.api_version=="v1beta1"`,
			expect:        []*pb.Result{tkrV1beta1Results[2], tkrV1beta1Results[3]},
			expectStatus:  codes.OK,
			pageSize:      3,
			pageToken:     getEncodedPageToken(t, &pb.ListPageIdentifier{ResultName: tkrV1beta1Results[2].GetName(), Filter: `taskrun.api_version=="v1beta1"`}),
			nextPageToken: "",
		},
	}
	// Test if we can find inserted taskruns
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			res, err := srv.ListResultsResult(ctx, &pb.ListResultsRequest{
				Filter:    tc.filter,
				PageSize:  tc.pageSize,
				PageToken: tc.pageToken,
			})
			if tc.expectStatus != status.Code(err) {
				t.Fatalf("failed on test %v: %v", tc.name, err)
			}
			gotList := res.GetResults()
			sort.SliceStable(gotList, func(i, j int) bool {
				return gotList[i].String() < gotList[j].String()
			})
			sort.SliceStable(tc.expect, func(i, j int) bool {
				return tc.expect[i].String() < tc.expect[j].String()
			})
			if diff := cmp.Diff(gotList, tc.expect, protocmp.Transform()); diff != "" {
				t.Fatalf("could not get the same taskrun: %v", diff)
			}
			nextPageToken := res.GetNextPageToken()
			if tc.nextPageToken != nextPageToken {
				t.Fatalf("NextPageToken mismatched, expected: %v, got: %v", tc.nextPageToken, nextPageToken)
			}
			if tc.nextPageName != "" {
				if nextPageIdentifier, err := decodePageToken(nextPageToken); err != nil {
					t.Fatalf("Error decoding nextPageToken: %v", err)
				} else if nextPageIdentifier.ResultName != tc.nextPageName {
					t.Fatalf("NextPageName mismatched, expected: %v, got: %v", tc.nextPageName, nextPageIdentifier.ResultName)
				}
			}
		})
	}
}

func Result(in ...proto.Message) *pb.Result {
	executions := make([]*pb.Execution, 0, len(in))
	for _, m := range in {
		switch x := m.(type) {
		case *ppb.TaskRun:
			executions = append(executions, &pb.Execution{Execution: &pb.Execution_TaskRun{x}})
		case *ppb.PipelineRun:
			executions = append(executions, &pb.Execution{Execution: &pb.Execution_PipelineRun{x}})
		default:
			panic(fmt.Sprintf("unknown message: %v", m))
		}
	}
	return &pb.Result{Executions: executions}
}

func getEncodedPageToken(t *testing.T, pi *pb.ListPageIdentifier) string {
	if token, err := encodePageResult(pi); err != nil {
		t.Fatalf("Failed to get encoded token: %v", err)
		return ""
	} else {
		return token
	}
}
