// Copyright 2020 The Tekton Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controller

import (
	"io/ioutil"
	"testing"

	"k8s.io/apimachinery/pkg/util/diff"
)

func TestTemplateRender(t *testing.T) {
	tr := taskrun()
	got, err := render(tr)
	if err != nil {
		t.Fatalf("error rendering template: %v", err)
	}

	want, err := ioutil.ReadFile("testdata/taskrun.md.golden")
	if err != nil {
		t.Fatalf("error reading desired result markdown: %v", err)
	}

	if string(want) != got {
		t.Errorf("-want,+got: %s", diff.StringDiff(string(want), got))
	}
}
