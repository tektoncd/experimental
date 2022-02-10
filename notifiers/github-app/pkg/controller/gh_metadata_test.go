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
	"fmt"
	"testing"

	"k8s.io/apimachinery/pkg/util/diff"
)

func TestGetRepoMetadata(t *testing.T) {
	for _, tc := range []struct {
		ownerAnnotation   string
		repoAnnotation    string
		repoUrlAnnotation string
		wantOwner         string
		wantRepo          string
	}{
		{
			ownerAnnotation:   "tektoncd",
			repoAnnotation:    "experimental",
			repoUrlAnnotation: "",
			wantOwner:         "tektoncd",
			wantRepo:          "experimental",
		},
		{
			ownerAnnotation:   "",
			repoAnnotation:    "",
			repoUrlAnnotation: "https://github.com/tektoncd/experimental",
			wantOwner:         "tektoncd",
			wantRepo:          "experimental",
		},
		{
			ownerAnnotation:   "tektoncd1",
			repoAnnotation:    "experimental1",
			repoUrlAnnotation: "https://github.com/tektoncd/experimental.git",
			wantOwner:         "tektoncd",
			wantRepo:          "experimental",
		},
	} {
		t.Run(fmt.Sprintf("%s-%s", tc.wantOwner, tc.wantRepo), func(t *testing.T) {
			tr := taskrun("testdata/taskrun.yaml")
			tr.Annotations[key("owner")] = tc.ownerAnnotation
			tr.Annotations[key("repo")] = tc.repoAnnotation
			tr.Annotations[key("repo-url")] = tc.repoUrlAnnotation

			owner, repo, err := getRepoMetadata(tr)

			if tc.wantOwner != owner {
				t.Errorf("-want,+got:\n%s", diff.StringDiff(tc.wantOwner, owner))
			}

			if tc.wantRepo != repo {
				t.Errorf("-want,+got:\n%s", diff.StringDiff(tc.wantRepo, repo))
			}

			if err != nil {
				t.Fatalf("getRepoMetadata: %v", err)
			}
		})
	}
}

func TestGetCommitMetadata(t *testing.T) {
	for _, tc := range []struct {
		commitAnnotation string
	}{
		{
			commitAnnotation: "1234",
		},
	} {
		t.Run(fmt.Sprintf(tc.commitAnnotation), func(t *testing.T) {
			tr := taskrun("testdata/taskrun.yaml")
			tr.Annotations[key("commit")] = tc.commitAnnotation

			//metadata := make(map[string]string)
			//var err error
			commit, err := getCommitMetadata(tr)

			if tc.commitAnnotation != commit {
				t.Errorf("-want,+got:\n%s", diff.StringDiff(tc.commitAnnotation, commit))
			}

			if err != nil {
				t.Fatalf("getCommitMetadata: %v", err)
			}
		})
	}
}

func TestGetStatusMetadata(t *testing.T) {
	for _, tc := range []struct {
		ownerAnnotation   string
		repoAnnotation    string
		repoUrlAnnotation string
		commitAnnotation  string
		wantOwner         string
		wantRepo          string
	}{
		{
			ownerAnnotation:   "tektoncd",
			repoAnnotation:    "experimental",
			repoUrlAnnotation: "",
			commitAnnotation:  "1234",
			wantOwner:         "tektoncd",
			wantRepo:          "experimental",
		},
		{
			ownerAnnotation:   "",
			repoAnnotation:    "",
			repoUrlAnnotation: "https://github.com/tektoncd/experimental",
			commitAnnotation:  "1234",
			wantOwner:         "tektoncd",
			wantRepo:          "experimental",
		},
		{
			ownerAnnotation:   "tektoncd1",
			repoAnnotation:    "experimental1",
			repoUrlAnnotation: "https://github.com/tektoncd/experimental",
			commitAnnotation:  "1234",
			wantOwner:         "tektoncd",
			wantRepo:          "experimental",
		},
	} {
		t.Run(fmt.Sprintf("%s-%s", tc.wantOwner, tc.wantRepo), func(t *testing.T) {
			tr := taskrun("testdata/taskrun.yaml")
			tr.Annotations[key("owner")] = tc.ownerAnnotation
			tr.Annotations[key("repo")] = tc.repoAnnotation
			tr.Annotations[key("repo-url")] = tc.repoUrlAnnotation
			tr.Annotations[key("commit")] = tc.commitAnnotation

			//metadata := make(map[string]string)
			//var err error
			metadata, err := getStatusMetadata(tr)

			if tc.wantOwner != metadata["owner"] {
				t.Errorf("-want,+got:\n%s", diff.StringDiff(tc.wantOwner, metadata["owner"]))
			}

			if tc.wantRepo != metadata["repo"] {
				t.Errorf("-want,+got:\n%s", diff.StringDiff(tc.wantRepo, metadata["repo"]))
			}

			if tc.commitAnnotation != metadata["commit"] {
				t.Errorf("-want,+got:\n%s", diff.StringDiff(tc.commitAnnotation, metadata["commit"]))
			}

			if err != nil {
				t.Fatalf("getStatusMetadata: %v", err)
			}
		})
	}
}
