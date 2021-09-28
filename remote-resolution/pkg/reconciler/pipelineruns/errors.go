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

package pipelineruns

import "fmt"

type ErrorInvalidResourceKey struct {
	key      string
	original error
}

var _ error = &ErrorInvalidResourceKey{}

func (e *ErrorInvalidResourceKey) Error() string {
	return fmt.Sprintf("invalid resource key %q: %v", e.key, e.original)
}

func (e *ErrorInvalidResourceKey) Unwrap() error {
	return e.original
}

type ErrorGettingResource struct {
	kind     string
	key      string
	original error
}

var _ error = &ErrorGettingResource{}

func (e *ErrorGettingResource) Error() string {
	return fmt.Sprintf("error getting %s %q: %v", e.kind, e.key, e.original)
}

func (e *ErrorGettingResource) Unwrap() error {
	return e.original
}
