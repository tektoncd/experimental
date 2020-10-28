/*
Copyright 2020 The Tekton Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/tektoncd/experimental/results/pkg/api/server"
	pb "github.com/tektoncd/experimental/results/proto/v1alpha1/results_go_proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	flag.Parse()

	user, pass := os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD")
	if user == "" || pass == "" {
		log.Fatal("Must provide both DB_USER and DB_PASSWORD")
	}
	// Connect to the MySQL database.
	// DSN derived from https://github.com/go-sql-driver/mysql#dsn-data-source-name
	dbURI := fmt.Sprintf("%s:%s@%s(%s)/%s?parseTime=true", user, pass, os.Getenv("DB_PROTOCOL"), os.Getenv("DB_ADDR"), os.Getenv("DB_NAME"))
	db, err := sql.Open("mysql", dbURI)
	if err != nil {
		log.Fatalf("failed to open the results.db: %v", err)
	}
	defer db.Close()

	// Create cel enviroment for filter
	srv, err := server.New(db)
	if err != nil {
		log.Fatalf("failed to create server: %v", err)
	}
	// Listen for gRPC requests.
	port := os.Getenv("PORT")
	if port == "" {
		// Default gRPC server port to this value from tutorials (e.g., https://grpc.io/docs/guides/auth/#go)
		port = "50051"
	}
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterResultsServer(s, srv)
	reflection.Register(s)
	log.Printf("Listening on :%s...", port)
	log.Fatal(s.Serve(lis))
}
