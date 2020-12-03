package server

import (
	"fmt"

	"github.com/google/cel-go/cel"
	"github.com/google/uuid"
	resultscel "github.com/tektoncd/experimental/results/pkg/api/server/cel"
	pb "github.com/tektoncd/experimental/results/proto/v1alpha2/results_go_proto"
	"gorm.io/gorm"
)

var (
	uid = func() string {
		return uuid.New().String()
	}
)

// Server with implementation of API server
type Server struct {
	pb.UnimplementedResultsServer
	env *cel.Env
	db  *gorm.DB
}

// New set up environment for the api server
func New(db *gorm.DB) (*Server, error) {
	env, err := resultscel.NewEnv()
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL environment: %w", err)
	}
	srv := &Server{
		db:  db,
		env: env,
	}
	return srv, nil
}
