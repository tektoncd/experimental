//go:generate rm -rf pipeline_go_proto
//go:generate mkdir pipeline_go_proto
//go:generate protoc --go_out=pipeline_go_proto --go-grpc_out=pipeline_go_proto --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative -I$GOPATH/src/github.com/googleapis/googleapis -I. common.proto taskrun.proto pipelinerun.proto

package v1beta1
