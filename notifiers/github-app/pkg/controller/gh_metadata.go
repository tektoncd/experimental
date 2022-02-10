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
	"errors"
	"net/url"
	"strings"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

// Extract the repo and owner information from the Annotations
func getRepoMetadata(tr *v1beta1.TaskRun) (string, string, error) {
	var owner string
	var repo string
	// The repo-url Annotation takes precedence
	// over separate owner, repo Annotations
	repoUrl := tr.Annotations[key("repo-url")]
	if repoUrl != "" {
		urlParsed, err := url.Parse(repoUrl)
		if err != nil {
			return owner, repo, err
		}
		path := strings.Split(urlParsed.Path, "/")
		owner = path[1]
		// filter if the .git is appended to the url
		repo = strings.Split(path[2], ".")[0]
	} else {
		owner = tr.Annotations[key("owner")]
		repo = tr.Annotations[key("repo")]
	}
	if owner == "" || repo == "" {
		return owner, repo, errors.New(
			"owner or repo is not set. Please set the owner, repo or repo-url annotation",
		)
	}
	return owner, repo, nil
}

// Extract the repo, owner and commit from the taskRun
func getCommitMetadata(tr *v1beta1.TaskRun) (string, error) {
	sha := tr.Annotations[key("commit")]
	if sha == "" {
		return sha, errors.New("commit sha is empty")
	}
	return sha, nil
}

// The full metadata for the status update url
func getStatusMetadata(tr *v1beta1.TaskRun) (map[string]string, error) {
	metadata := make(map[string]string)

	var repoErr error
	metadata["owner"], metadata["repo"], repoErr = getRepoMetadata(tr)
	if repoErr != nil {
		return metadata, errors.New("error returning repo metadata")
	}

	var commitErr error
	metadata["commit"], commitErr = getCommitMetadata(tr)
	if commitErr != nil {
		return metadata, errors.New("error returning commit metadata")
	}

	return metadata, nil
}
