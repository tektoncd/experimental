//go:generate rm -rf results_go_proto
//go:generate mkdir results_go_proto
//go:generate protoc --go_out=results_go_proto --go-grpc_out=results_go_proto --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative -I$GOPATH/src/github.com/googleapis/googleapis -I../pipeline/v1beta1 -I. api.proto

package v1alpha1
