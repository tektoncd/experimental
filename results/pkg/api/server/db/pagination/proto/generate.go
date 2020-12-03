//go:generate rm -rf internal_go_proto
//go:generate mkdir internal_go_proto
//go:generate protoc --go_out=internal_go_proto --go_opt=paths=source_relative -I. pagination.proto

package internal
