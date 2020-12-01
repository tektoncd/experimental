package server

import (
	"github.com/google/uuid"
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
	db *gorm.DB
}

// New set up environment for the api server
func New(db *gorm.DB) (*Server, error) {
	srv := &Server{
		db: db,
	}
	return srv, nil
}
