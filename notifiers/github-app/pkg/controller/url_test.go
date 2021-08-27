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
	"testing"

	"k8s.io/apimachinery/pkg/util/diff"
)

func TestDashboardURL(t *testing.T) {
	for _, tc := range []struct {
		detailsURLAnnotation string
		wantDetailsURL       string
	}{
		{
			detailsURLAnnotation: "https://tekton.dev",
			wantDetailsURL:       "https://tekton.dev",
		},
		{
			detailsURLAnnotation: "https://dashboard.dogfooding.tekton.dev/#/namespaces/{{ .Namespace }}/taskruns/{{ .Name }}",
			wantDetailsURL:       "https://dashboard.dogfooding.tekton.dev/#/namespaces/default/taskruns/echo-6b4fn-echo-xrxq4",
		},
	} {
		t.Run(tc.detailsURLAnnotation, func(t *testing.T) {
			tr := taskrun("testdata/taskrun.yaml")
			tr.Annotations[key("url")] = tc.detailsURLAnnotation

			url, err := dashboardURL(tr)

			if tc.wantDetailsURL != url {
				t.Errorf("-want,+got:\n%s", diff.StringDiff(tc.wantDetailsURL, url))
			}

			if err != nil {
				t.Fatalf("DetailsURL: %v", err)
			}
		})
	}
}
