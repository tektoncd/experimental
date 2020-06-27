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
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/golang/protobuf/proto"
	pb "github.com/tektoncd/experimental/results/proto/proto"
	"google.golang.org/grpc"
)

const (
	dbname     = "tekton-results"
	dbprotocol = "tcp"
)

var dbaddr = flag.String("db_addr", "mysql.tekton-pipelines.svc.cluster.local", "Address of MySQL database to use.")

func main() {
	flag.Parse()

	user, pass := os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD")
	if user == "" || pass == "" {
		log.Fatal("Must provide both DB_USER and DB_PASSWORD")
	}

	if *dbaddr == "" {
		log.Fatal("Must provide --db_addr")
	}

	port := os.Getenv("PORT")
	if port == "" {
		// Default gRPC server port to this value from tutorials (e.g., https://grpc.io/docs/guides/auth/#go)
		port = ":50051"
	}

	// Connect to the MySQL database.
	// DSN derived from https://github.com/go-sql-driver/mysql#dsn-data-source-name
	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@%s(%s)/", user, pass, dbprotocol, *dbaddr))
	if err != nil {
		log.Fatalf("failed to open the results.db: %v", err)
	}
	defer db.Close()

	// Sanity check that we can connect to MySQL.
	if _, err := db.Exec("SHOW databases;"); err != nil {
		log.Fatalf("show databases: %v", err)
	}

	// Initialize the database if it doesn't exist.
	if _, err := db.Exec("CREATE DATABASE IF NOT EXISTS `tekton-results`;"); err != nil {
		log.Fatalf("create database: %v", err)
	}

	// Listen for gRPC requests.
	srv := &server{db: db}
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterResultsServer(s, srv)
	log.Printf("Listening on %s...", port)
	log.Fatal(s.Serve(lis))
}

type server struct {
	pb.UnimplementedResultsServer

	db *sql.DB
}

// CreateTaskRun receives TaskRun from Watcher and save it to local Sqlite Server.
func (s *server) CreateTaskRun(ctx context.Context, req *pb.CreateTaskRunRequest) (*pb.TaskRun, error) {
	database := s.db
	statement, err := database.Prepare("INSERT INTO taskrun (taskrunlog, uid, name, namespace) VALUES (?, ?, ?, ?)")
	if err != nil {
		log.Printf("failed to insert a new taskrun: %v\n", err)
		return nil, fmt.Errorf("failed to insert a new taskrun: %v", err)
	}

	// serialize data and insert it into database.
	taskrunFromClient := req.GetTaskRun()
	blobData, err := proto.Marshal(taskrunFromClient)
	if err != nil {
		log.Println("taskrun marshaling error: ", err)
		return nil, fmt.Errorf("failed to marshal taskrun: %v", err)
	}
	taskrunMeta := taskrunFromClient.GetMetadata()
	if _, err := statement.Exec(blobData, taskrunMeta.GetUid(), taskrunMeta.GetName(), taskrunMeta.GetNamespace()); err != nil {
		log.Printf("failed to execute insertion of a new taskrun: %v\n", err)
		return nil, fmt.Errorf("failed to excute insertion a new taskrun: %v", err)

	}
	return taskrunFromClient, nil
}
