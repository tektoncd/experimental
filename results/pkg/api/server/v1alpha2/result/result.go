// Package result provides utilities for manipulating and validating Results.
package result

import (
	"fmt"
	"regexp"

	"github.com/tektoncd/experimental/results/pkg/api/server/db"
	pb "github.com/tektoncd/experimental/results/proto/v1alpha2/results_go_proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	// NameRegex matches valid name specs for a Result.
	NameRegex = regexp.MustCompile("(^[a-z0-9_-]{1,63})/results/([a-z0-9_-]{1,63}$)")
)

// ParseName splits a full Result name into its individual (parent, name)
// components.
func ParseName(raw string) (parent, name string, err error) {
	s := NameRegex.FindStringSubmatch(raw)
	if len(s) != 3 {
		return "", "", status.Errorf(codes.InvalidArgument, "name must match %s", NameRegex.String())
	}
	return s[1], s[2], nil
}

// ToStorage converts an API Result into its corresponding database storage
// equivalent.
// parent,name should be the name parts (e.g. not containing "/results/").
func ToStorage(parent, name string, r *pb.Result) (*db.Result, error) {
	result := &db.Result{
		Parent: parent,
		ID:     r.GetId(),
		Name:   name,
	}
	return result, nil
}

// ToAPI converts a database storage Result into its corresponding API
// equivalent.
func ToAPI(r *db.Result) *pb.Result {
	return &pb.Result{
		Name: fmt.Sprintf("%s/results/%s", r.Parent, r.Name),
		Id:   r.ID,
	}
}
