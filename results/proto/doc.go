//go:generate protoc --go_out=proto --go-grpc_out=proto --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative -I$GOPATH/src/github.com/googleapis/googleapis -I. api.proto taskrun.proto

package proto
