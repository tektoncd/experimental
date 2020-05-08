//go:generate protoc -I . --go_out=plugins=grpc:. api.proto taskrun.proto

package proto
