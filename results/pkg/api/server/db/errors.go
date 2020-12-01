package db

import (
	"errors"

	"github.com/mattn/go-sqlite3"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

// WrapError converts database error codes into their corresponding gRPC status
// codes.
func WrapError(err error) error {
	if err == nil {
		return err
	}

	// Check for gorm provided errors first - these are more likely to be
	// supported across drivers.
	if code, ok := gormCode(err); ok {
		return status.Error(code, err.Error())
	}

	// Fallback to implementation specific codes.
	switch e := err.(type) {
	case sqlite3.Error:
		return status.Error(sqlite(e), err.Error())
	}

	return err
}

// gormCode returns gRPC status codes corresponding to gorm errors. This is not
// an exhaustive list.
// See https://pkg.go.dev/gorm.io/gorm@v1.20.7#pkg-variables for list of
// errors.
func gormCode(err error) (codes.Code, bool) {
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return codes.NotFound, true
	}
	return codes.Unknown, false
}

// sqlite converts sqlite3 error codes to gRPC status codes. This is not an
// exhaustive list.
// See https://pkg.go.dev/github.com/mattn/go-sqlite3#pkg-variables for list of
// error codes.
func sqlite(err sqlite3.Error) codes.Code {
	switch err.Code {
	case sqlite3.ErrConstraint:
		switch err.ExtendedCode {
		case sqlite3.ErrConstraintUnique:
			return codes.AlreadyExists
		}
		return codes.InvalidArgument
	case sqlite3.ErrNotFound:
		return codes.NotFound
	}
	return status.Code(err)
}

// TODO: MySQL codes
