//go:generate protoc --go_out=proto --go-grpc_out=proto --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative -I$GOPATH/src/github.com/googleapis/googleapis -I. common.proto taskrun.proto pipelinerun.proto api.proto

package proto
