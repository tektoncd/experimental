package server

import (
	"context"

	"github.com/tektoncd/experimental/results/pkg/api/server/db"
	"github.com/tektoncd/experimental/results/pkg/api/server/v1alpha2/result"
	pb "github.com/tektoncd/experimental/results/proto/v1alpha2/results_go_proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CreateResult creates a new result in the database.
func (s *Server) CreateResult(ctx context.Context, req *pb.CreateResultRequest) (*pb.Result, error) {
	r := req.GetResult()

	// Validate the incoming request
	parent, name, err := result.ParseName(r.GetName())
	if err != nil {
		return nil, err
	}
	if req.GetParent() != parent {
		return nil, status.Error(codes.InvalidArgument, "requested parent does not match resource name")
	}

	// Populate Result with server provided fields.
	r.Id = uid()

	store, err := result.ToStorage(parent, name, req.GetResult())
	if err != nil {
		return nil, err
	}
	if err := db.WrapError(s.db.WithContext(ctx).Create(store).Error); err != nil {
		return nil, err
	}

	return result.ToAPI(store), nil
}

// GetResult returns a single Result.
func (s *Server) GetResult(ctx context.Context, req *pb.GetResultRequest) (*pb.Result, error) {
	parent, name, err := result.ParseName(req.GetName())
	if err != nil {
		return nil, err
	}
	store := &db.Result{}
	q := s.db.WithContext(ctx).
		Where(&db.Result{Parent: parent, Name: name}).
		First(store)
	if err := db.WrapError(q.Error); err != nil {
		return nil, err
	}
	return result.ToAPI(store), nil
}
