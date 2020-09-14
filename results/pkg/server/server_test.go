package server

import (
	"context"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	pb "github.com/tektoncd/experimental/results/proto/proto"
	"google.golang.org/genproto/protobuf/field_mask"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/testing/protocmp"
)

const (
	address = "localhost:0"
)

// Test functionality of Server code
func TestCreateResult(t *testing.T) {
	// Create a temporay database
	srv, err := SetupTestDB(t)
	if err != nil {
		t.Fatalf("failed to create temp file for db: %v", err)
	}

	// connect to fake server and do testing
	ctx := context.Background()
	if _, err := srv.CreateResult(ctx, &pb.CreateResultRequest{
		Result: TaskRunResult(&pb.TaskRun{
			ApiVersion: "1",
			Metadata: &pb.ObjectMeta{
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
	srv, err := SetupTestDB(t)
	if err != nil {
		t.Fatalf("failed to setup db: %v", err)
	}
	ctx := context.Background()
	// Connect to fake server and create a new taskrun
	r, err := srv.CreateResult(ctx, &pb.CreateResultRequest{
		Result: TaskRunResult(&pb.TaskRun{
			ApiVersion: "v1beta1",
			Metadata: &pb.ObjectMeta{
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
	srv, err := SetupTestDB(t)
	if err != nil {
		t.Fatalf("failed to create temp file for db: %v", err)
	}
	ctx := context.Background()

	// Validate by checking if ouput equlas expected Result
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
			update: TaskRunResult(&pb.TaskRun{
				ApiVersion: "v1beta1",
				Metadata: &pb.ObjectMeta{
					Uid:       "123456",
					Name:      "newtaskrun",
					Namespace: "tekton",
				},
			}),
			// update entire taskrun
			expect: TaskRunResult(&pb.TaskRun{
				ApiVersion: "v1beta1",
				Metadata: &pb.ObjectMeta{
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
			in: TaskRunResult(&pb.TaskRun{
				ApiVersion: "v1alpha1",
				Metadata: &pb.ObjectMeta{
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
			in: TaskRunResult(&pb.TaskRun{
				Metadata: &pb.ObjectMeta{
					Name: "foo",
				},
			}),
			name:      "Test Mask updating repeated fields",
			fieldmask: &field_mask.FieldMask{Paths: []string{"executions"}},
			// update entire repeated field(all elements in array) - standard update
			update: TaskRunResult(&pb.TaskRun{
				Metadata: &pb.ObjectMeta{
					Name: "bar",
				},
			}),
			expect: TaskRunResult(&pb.TaskRun{
				Metadata: &pb.ObjectMeta{
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
	srv, err := SetupTestDB(t)
	if err != nil {
		t.Fatalf("failed to create temp file for db: %v", err)
	}
	// Connect to fake server and insert a new taskrun
	ctx := context.Background()
	r, err := srv.CreateResult(ctx, &pb.CreateResultRequest{
		Result: TaskRunResult(&pb.TaskRun{
			ApiVersion: "1",
			Metadata: &pb.ObjectMeta{
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
	srv, err := SetupTestDB(t)
	if err != nil {
		t.Fatalf("failed to setup db: %v", err)
	}
	ctx := context.Background()

	// Create a bunch of taskruns for testing
	t1 := &pb.TaskRun{
		ApiVersion: "v1beta1",
		Metadata: &pb.ObjectMeta{
			Uid:       "31415926",
			Name:      "taskrun",
			Namespace: "default",
		},
	}
	t2 := &pb.TaskRun{
		ApiVersion: "v1alpha1",
		Metadata: &pb.ObjectMeta{
			Uid:       "43245243",
			Name:      "task",
			Namespace: "default",
		},
	}
	t3 := &pb.TaskRun{
		ApiVersion: "v1beta1",
		Metadata: &pb.ObjectMeta{
			Uid:       "1234556",
			Name:      "mytaskrun",
			Namespace: "demo",
		},
	}
	t4 := &pb.TaskRun{
		ApiVersion: "v1beta1",
		Metadata: &pb.ObjectMeta{
			Uid:       "543535",
			Name:      "newtaskrun",
			Namespace: "demo",
		},
	}
	t5 := &pb.TaskRun{
		ApiVersion: "v1alpha1",
		Metadata: &pb.ObjectMeta{
			Uid:       "543535",
			Name:      "newtaskrun",
			Namespace: "official",
		},
	}
	results := []*pb.Result{TaskRunResult(t1), TaskRunResult(t2), TaskRunResult(t3), TaskRunResult(t4), TaskRunResult(t5)}
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
	tt := []struct {
		name         string
		filter       string
		expect       []*pb.Result
		expectStatus codes.Code
	}{
		{
			name:         "test simple query",
			filter:       `taskrun.api_version=="v1beta1"`,
			expect:       []*pb.Result{gotResults[0], gotResults[2], gotResults[3]},
			expectStatus: codes.OK,
		},
		{
			name:         "test query with simple function",
			filter:       `taskrun.metadata.name.endsWith("run")`,
			expect:       []*pb.Result{gotResults[0], gotResults[2], gotResults[3], gotResults[4]},
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
	}
	// Test if we can find inserted taskruns
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			res, err := srv.ListResultsResult(ctx, &pb.ListResultsRequest{
				Filter: tc.filter,
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
		})
	}
}

func TaskRunResult(tr *pb.TaskRun) *pb.Result {
	return &pb.Result{
		Executions: []*pb.Execution{{
			Execution: &pb.Execution_TaskRun{tr},
		}},
	}
}
