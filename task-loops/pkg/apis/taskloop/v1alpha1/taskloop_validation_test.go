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

package v1alpha1_test

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	taskloopv1alpha1 "github.com/tektoncd/experimental/task-loops/pkg/apis/taskloop/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/test/diff"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
)

func TestTaskLoop_Validate_Success(t *testing.T) {
	tests := []struct {
		name string
		tl   *taskloopv1alpha1.TaskLoop
	}{{
		name: "taskRef",
		tl: &taskloopv1alpha1.TaskLoop{
			ObjectMeta: metav1.ObjectMeta{Name: "taskloop"},
			Spec: taskloopv1alpha1.TaskLoopSpec{
				TaskRef: &v1beta1.TaskRef{Name: "mytask"},
			},
		},
	}, {
		name: "taskSpec",
		tl: &taskloopv1alpha1.TaskLoop{
			ObjectMeta: metav1.ObjectMeta{Name: "taskloop"},
			Spec: taskloopv1alpha1.TaskLoopSpec{
				TaskSpec: &v1beta1.TaskSpec{
					Steps: []v1beta1.Step{{
						Container: corev1.Container{Name: "foo", Image: "bar"},
					}},
				},
			},
		},
	}}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.tl.Validate(context.Background())
			if err != nil {
				t.Errorf("Unexpected error for %s: %s", tc.name, err)
			}
		})
	}
}

func TestTaskLoop_Validate_Error(t *testing.T) {
	tests := []struct {
		name          string
		tl            *taskloopv1alpha1.TaskLoop
		expectedError apis.FieldError
	}{{
		name: "no taskRef or taskSpec",
		tl: &taskloopv1alpha1.TaskLoop{
			ObjectMeta: metav1.ObjectMeta{Name: "taskloop"},
			Spec:       taskloopv1alpha1.TaskLoopSpec{},
		},
		expectedError: apis.FieldError{
			Message: "expected exactly one, got neither",
			Paths:   []string{"spec.taskRef", "spec.taskSpec"},
		},
	}, {
		name: "both taskRef and taskSpec",
		tl: &taskloopv1alpha1.TaskLoop{
			ObjectMeta: metav1.ObjectMeta{Name: "taskloop"},
			Spec: taskloopv1alpha1.TaskLoopSpec{
				TaskRef: &v1beta1.TaskRef{Name: "mytask"},
				TaskSpec: &v1beta1.TaskSpec{
					Steps: []v1beta1.Step{{
						Container: corev1.Container{Name: "foo", Image: "bar"},
					}},
				},
			},
		},
		expectedError: apis.FieldError{
			Message: "expected exactly one, got both",
			Paths:   []string{"spec.taskRef", "spec.taskSpec"},
		},
	}, {
		name: "invalid taskRef",
		tl: &taskloopv1alpha1.TaskLoop{
			ObjectMeta: metav1.ObjectMeta{Name: "taskloop"},
			Spec: taskloopv1alpha1.TaskLoopSpec{
				TaskRef: &v1beta1.TaskRef{Name: "_bad"},
			},
		},
		expectedError: apis.FieldError{
			Message: "invalid value: name part must consist of alphanumeric characters, '-', '_' or '.', and must start " +
				"and end with an alphanumeric character (e.g. 'MyName',  or 'my.name',  or '123-abc', regex used for " +
				"validation is '([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]')",
			Paths: []string{"spec.taskRef.name"},
		},
	}, {
		name: "invalid taskSpec",
		tl: &taskloopv1alpha1.TaskLoop{
			ObjectMeta: metav1.ObjectMeta{Name: "taskloop"},
			Spec: taskloopv1alpha1.TaskLoopSpec{
				TaskSpec: &v1beta1.TaskSpec{
					Steps: []v1beta1.Step{{
						Container: corev1.Container{Name: "bad@name!", Image: "bar"},
					}},
				},
			},
		},
		expectedError: apis.FieldError{
			Message: `invalid value "bad@name!"`,
			Details: "Task step name must be a valid DNS Label, For more info refer to https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names",
			Paths:   []string{"spec.taskSpec.taskspec.steps.name"},
		},
	}}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.tl.Validate(context.Background())
			if err == nil {
				t.Errorf("Expected an Error but did not get one for %s", tc.name)
			} else {
				if d := cmp.Diff(tc.expectedError, *err, cmpopts.IgnoreUnexported(apis.FieldError{})); d != "" {
					t.Errorf("Error is different from expected for %s. diff %s", tc.name, diff.PrintWantGot(d))
				}
			}
		})
	}
}
