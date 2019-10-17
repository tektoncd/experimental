/*
Copyright 2019 The Tekton Authors
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

package endpoints

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	v1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
)

func TestStringParam(t *testing.T) {
	want := v1alpha1.Param{
		Name: "foo",
		Value: v1alpha1.ArrayOrString{
			Type:      v1alpha1.ParamTypeString,
			StringVal: "bar",
		},
	}
	got := stringParam("foo", "bar")
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("stringParam(): -want +got: %s", diff)
	}
}
