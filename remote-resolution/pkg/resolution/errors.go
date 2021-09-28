/*
Copyright 2021 The Tekton Authors

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

package resolution

import (
	"errors"
	"fmt"
)

// resolution.Error embeds both a short machine-readable string reason for resolution
// problems alongside the original error generated during the resolution flow.
type Error struct {
	Reason   string
	Original error
}

var _ error = &Error{}

// Error returns the original error's message. This is intended to meet the error.Error interface.
func (e *Error) Error() string {
	return e.Original.Error()
}

// Unwrap returns the original error without the Reason annotation. This is
// intended to support usage of errors.Is and errors.As with resolution.Errors.
func (e *Error) Unwrap() error {
	return e.Original
}

// NewError returns a resolution.Error with the given reason and underlying
// original error.
func NewError(reason string, err error) *Error {
	return &Error{
		Reason:   reason,
		Original: err,
	}
}

var (
	// ErrorTaskRunAlreadyResolved is a sentinel value that consumers of the resolution package can use to determine if a taskrun
	// was already resolved and, if so, customize their fallback behaviour.
	ErrorTaskRunAlreadyResolved = NewError("TaskRunAlreadyResolved", errors.New("TaskRun is already resolved"))

	// ErrorPipelineRunAlreadyResolved is a sentinel value that consumers of the resolution package can use to determine if a pipelinerun
	// was already resolved and, if so, customize their fallback behaviour.
	ErrorPipelineRunAlreadyResolved = NewError("PipelineRunAlreadyResolved", errors.New("PipelineRun is already resolved"))

	// ErrorResourceNotResolved is a sentinel value to indicate that a TaskRun or PipelineRun
	// has not been resolved yet.
	ErrorResourceNotResolved = NewError("ResourceNotResolved", errors.New("Resource has not been resolved"))
)

type ErrorInvalidResourceKey struct {
	Key      string
	Original error
}

var _ error = &ErrorInvalidResourceKey{}

func (e *ErrorInvalidResourceKey) Error() string {
	return fmt.Sprintf("invalid resource key %q: %v", e.Key, e.Original)
}

func (e *ErrorInvalidResourceKey) Unwrap() error {
	return e.Original
}

// ErrorInvalidRequest is an error received when a
// ResourceRequest is badly formed for some reason: either the
// parameters don't match the resolver's expectations or there is some
// other structural issue with the submitted ResourceRequest object.
type ErrorInvalidRequest struct {
	ResourceRequestKey string
	Message            string
}

var _ error = &ErrorInvalidRequest{}

func (e *ErrorInvalidRequest) Error() string {
	return fmt.Sprintf("invalid resource request %q: %s", e.ResourceRequestKey, e.Message)
}

// ErrorGettingResource is an error received during what should
// otherwise have been a successful resource request.
type ErrorGettingResource struct {
	Kind     string
	Key      string
	Original error
}

var _ error = &ErrorGettingResource{}

func (e *ErrorGettingResource) Error() string {
	return fmt.Sprintf("error getting %q %q: %v", e.Kind, e.Key, e.Original)
}

func (e *ErrorGettingResource) Unwrap() error {
	return e.Original
}

// ErrorUpdatingRequest is an error during any part of the update
// process for a ResourceRequest, e.g. when attempting to patch the
// ResourceRequest with resolved data.
type ErrorUpdatingRequest struct {
	ResourceRequestKey string
	Original           error
}

var _ error = &ErrorUpdatingRequest{}

func (e *ErrorUpdatingRequest) Error() string {
	return fmt.Sprintf("error updating resource request %q with data: %w", e.ResourceRequestKey, e.Original)
}

func (e *ErrorUpdatingRequest) Unwrap() error {
	return e.Original
}

// ReasonError extracts the reason and underlying error
// embedded in a given error or returns some sane defaults
// if the error isn't a resolution.Error.
func ReasonError(err error) (string, error) {
	reason := ReasonResolutionFailed
	resolutionError := err

	if e, ok := err.(*Error); ok {
		reason = e.Reason
		resolutionError = e.Unwrap()
	}

	return reason, resolutionError
}
