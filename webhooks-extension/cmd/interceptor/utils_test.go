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

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/google/go-github/github"
	gitlab "github.com/xanzy/go-gitlab"
)

func TestSanitizeGitInput(t *testing.T) {
	// Key = URL to process
	// Value = Expected return
	urls := make(map[string]string)
	urls["https://github.com/foo/bar.git"] = "github.com/foo/bar"
	urls["https://github.com/foo/bar"] = "github.com/foo/bar"
	urls["http://github.com/foo/bar.git"] = "github.com/foo/bar"
	urls["http://github.com/foo/bar"] = "github.com/foo/bar"
	urls["github.com/foo/bar"] = "github.com/foo/bar"
	urls["HTTPS://github.com/foo/bar.GIT"] = "github.com/foo/bar"
	urls["hTtP://GiThUb.CoM/FoO/BaR"] = "github.com/foo/bar"
	urls["http://something.else/foo/bar/wibble.git"] = "something.else/foo/bar/wibble"

	for url := range urls {
		sanitized := sanitizeGitInput(url)
		if urls[url] != sanitized {
			t.Errorf("Error santizeGitInput returned unexpected value processing %s, %s was returned but expected %s", url, sanitized, urls[url])
		}
	}
}

func TestAddBranchAndTagGitHubEvents(t *testing.T) {
	ref1 := "blah/head/foo"
	ref2 := "refs/tags/v1.0"
	id := "12345678901234567890"

	ghPushEventExpectedResults := make(map[string]string)
	ghPushEventExpectedResults[ref1] = "{\"ref\":\"blah/head/foo\",\"head_commit\":{\"id\":\"12345678901234567890\"},\"webhooks-tekton-git-branch\":\"foo\",\"webhooks-tekton-image-tag\":\"1234567\"}"
	ghPushEventExpectedResults[ref2] = "{\"ref\":\"refs/tags/v1.0\",\"head_commit\":{\"id\":\"12345678901234567890\"},\"webhooks-tekton-git-branch\":\"v1.0\",\"webhooks-tekton-image-tag\":\"v1.0\"}"

	ghPullEventExpectedResults := make(map[string]string)
	ghPullEventExpectedResults[ref1] = "{\"pull_request\":{\"head\":{\"ref\":\"blah/head/foo\",\"sha\":\"12345678901234567890\"}},\"webhooks-tekton-git-branch\":\"foo\",\"webhooks-tekton-image-tag\":\"1234567\"}"
	ghPullEventExpectedResults[ref2] = "{\"pull_request\":{\"head\":{\"ref\":\"refs/tags/v1.0\",\"sha\":\"12345678901234567890\"}},\"webhooks-tekton-git-branch\":\"v1.0\",\"webhooks-tekton-image-tag\":\"v1.0\"}"

	// Perform Test
	refs := []string{ref1, ref2}
	for _, ref := range refs {
		// GitHub Push
		ghPushEvent := github.PushEvent{
			Ref: &ref,
			HeadCommit: &github.PushEventCommit{
				ID: &id,
			},
		}
		payload, err := addBranchAndTag(ghPushEvent)
		if err != nil {
			t.Errorf("Error: %s", err.Error())
		}
		if ghPushEventExpectedResults[ref] != string(payload) {
			t.Errorf("GitHub push event result unexpected, received %s, expected %s", string(payload), ghPushEventExpectedResults[ref])
		}

		// GitHub Pull Request
		ghPullEvent := github.PullRequestEvent{
			PullRequest: &github.PullRequest{
				Head: &github.PullRequestBranch{
					Ref: &ref,
					SHA: &id,
				},
			},
		}
		payload, err = addBranchAndTag(ghPullEvent)
		if err != nil {
			t.Errorf("Error: %s", err.Error())
		}
		if ghPullEventExpectedResults[ref] != string(payload) {
			t.Errorf("GitHub pull request event result unexpected, received %s, expected %s", string(payload), ghPullEventExpectedResults[ref])
		}

		// Unsupported Event
		unsupportedEvent := github.StarEvent{
			Action: &ref,
		}
		payload, err = addBranchAndTag(unsupportedEvent)
		if "" != string(payload) {
			t.Errorf("Unsupported event result unexpected, received %s, expected \"\"", string(payload))
		}
		if err.Error() != "Unsupported event type `github.StarEvent` received in addBranchAndTag()" {
			t.Errorf("Unexpected error received: %s", err.Error())
		}
	}
}

func TestAddBranchAndTagGitLabEvents(t *testing.T) {

	// GitLab Push
	glPushEvent := gitlab.PushEvent{
		Ref:         "blah/head/foo",
		CheckoutSHA: "12345678901234567890",
	}
	glPushEventExpectedResult := "{\"object_kind\":\"\",\"before\":\"\",\"after\":\"\",\"ref\":\"blah/head/foo\",\"checkout_sha\":\"12345678901234567890\",\"user_id\":0,\"user_name\":\"\",\"user_username\":\"\",\"user_email\":\"\",\"user_avatar\":\"\",\"project_id\":0,\"project\":{\"name\":\"\",\"description\":\"\",\"avatar_url\":\"\",\"git_ssh_url\":\"\",\"git_http_url\":\"\",\"namespace\":\"\",\"path_with_namespace\":\"\",\"default_branch\":\"\",\"homepage\":\"\",\"url\":\"\",\"ssh_url\":\"\",\"http_url\":\"\",\"web_url\":\"\",\"visibility\":\"\"},\"repository\":null,\"commits\":null,\"total_commits_count\":0,\"webhooks-tekton-git-branch\":\"foo\",\"webhooks-tekton-image-tag\":\"1234567\"}"
	payload, err := addBranchAndTag(&glPushEvent)
	if err != nil {
		t.Errorf("Error: %s", err.Error())
	}

	if glPushEventExpectedResult != string(payload) {
		t.Errorf("GitLab push event result unexpected, received %s, expected %s", string(payload), glPushEventExpectedResult)
	}

	// GitLab Tag Push
	glTagEvent := gitlab.TagEvent{
		Ref:         "refs/tags/v1.0",
		CheckoutSHA: "12345678901234567890",
	}
	glTagEventExpectedResult := "{\"object_kind\":\"\",\"before\":\"\",\"after\":\"\",\"ref\":\"refs/tags/v1.0\",\"checkout_sha\":\"12345678901234567890\",\"user_id\":0,\"user_name\":\"\",\"user_avatar\":\"\",\"project_id\":0,\"message\":\"\",\"project\":{\"name\":\"\",\"description\":\"\",\"avatar_url\":\"\",\"git_ssh_url\":\"\",\"git_http_url\":\"\",\"namespace\":\"\",\"path_with_namespace\":\"\",\"default_branch\":\"\",\"homepage\":\"\",\"url\":\"\",\"ssh_url\":\"\",\"http_url\":\"\",\"web_url\":\"\",\"visibility\":\"\"},\"repository\":null,\"commits\":null,\"total_commits_count\":0,\"webhooks-tekton-git-branch\":\"v1.0\",\"webhooks-tekton-image-tag\":\"v1.0\"}"
	payload, err = addBranchAndTag(&glTagEvent)
	if err != nil {
		t.Errorf("Error: %s", err.Error())
	}
	if glTagEventExpectedResult != string(payload) {
		t.Errorf("GitLab tag event result unexpected, received %s, expected %s", string(payload), glTagEventExpectedResult)
	}

	hookJSON := getGitlabMergeRequest()
	parsedEvent, err := gitlab.ParseWebhook("Merge Request Hook", []byte(hookJSON))
	if err != nil {
		t.Errorf("Error parsing merge request hook: %s", err)
	}

	event, ok := parsedEvent.(*gitlab.MergeEvent)
	if !ok {
		t.Errorf("Expected MergeEvent, but parsing produced %T", parsedEvent)
	}

	payload, err = addBranchAndTag(event)
	if err != nil {
		fmt.Println(err.Error())
	}

	var glMergeResult glPullRequestPayload
	err = json.Unmarshal(payload, &glMergeResult)
	if err != nil {
		t.Errorf("Error during unmarshall of payload for gitlab merge request in TestaddBranchAndTagGitLabMergeRequest test")
	}

	if glMergeResult.ObjectAttributes.TargetBranch != "foo" {
		t.Errorf("Error - TargetBranch appears to have changed to %s, the Event should be unaltered", glMergeResult.ObjectAttributes.TargetBranch)
	}
	if glMergeResult.WebhookBranch != "foo" {
		t.Errorf("Error - Inccorect branch name set, expected foo, received %s", glMergeResult.WebhookBranch)
	}
	if glMergeResult.WebhookSuggestedImageTag != "1234567" {
		t.Errorf("Error - Inccorect tag name set, expected 1234567, received %s", glMergeResult.WebhookSuggestedImageTag)
	}
}

func TestValidate(t *testing.T) {

	type test_configuration struct {
		requiredRepo       string
		requiredEvent      string
		requiredAction     string
		webhookURL         string
		webhookEventHeader string
		webhookEvent       string
		webhookPRAction    string
		triggerName        string
		expectation        bool
		expectedErr        error
	}

	configs := make(map[string]test_configuration)
	configs["push-valid"] = test_configuration{
		requiredRepo:       "http://github.com/foo/bar",
		requiredEvent:      "push, Push Hook, Tag Push Hook",
		requiredAction:     "",
		webhookURL:         "http://github.com/foo/bar",
		webhookEventHeader: "X-Github-Event",
		webhookEvent:       "push",
		webhookPRAction:    "",
		triggerName:        "github-push-valid",
		expectation:        true,
		expectedErr:        nil,
	}
	configs["push-valid-two"] = test_configuration{
		requiredRepo:       "http://gitlab.com/foo/bar",
		requiredEvent:      "push, Push Hook, Tag Push Hook",
		requiredAction:     "",
		webhookURL:         "http://gitlab.com/foo/bar",
		webhookEventHeader: "X-Gitlab-Event",
		webhookEvent:       "Tag Push Hook",
		webhookPRAction:    "",
		triggerName:        "push-valid-two",
		expectation:        true,
		expectedErr:        nil,
	}
	configs["push-valid-three-protocol-and-caps"] = test_configuration{
		requiredRepo:       "https://GITLAB.com/foo/BAR",
		requiredEvent:      "push, Push Hook, Tag Push Hook",
		requiredAction:     "",
		webhookURL:         "http://gitlab.com/foo/bar",
		webhookEventHeader: "X-Gitlab-Event",
		webhookEvent:       "Tag Push Hook",
		webhookPRAction:    "",
		triggerName:        "push-valid-three-protocol-and-caps",
		expectation:        true,
		expectedErr:        nil,
	}
	configs["push-repo-mismatch"] = test_configuration{
		requiredRepo:       "http://github.com/foo/bar",
		requiredEvent:      "push, Push Hook, Tag Push Hook",
		requiredAction:     "",
		webhookURL:         "http://github.com/foo/wrongrepo",
		webhookEventHeader: "X-Github-Event",
		webhookEvent:       "push",
		webhookPRAction:    "",
		triggerName:        "push-repo-mismatch",
		expectation:        false,
		expectedErr:        errors.New("Validator failed as repository URLs do not match"),
	}
	configs["push-event-mismatch"] = test_configuration{
		requiredRepo:       "http://github.com/foo/bar",
		requiredEvent:      "push, Push Hook, Tag Push Hook",
		requiredAction:     "",
		webhookURL:         "http://github.com/foo/bar",
		webhookEventHeader: "X-Github-Event",
		webhookEvent:       "pull_request",
		webhookPRAction:    "",
		triggerName:        "push-event-mismatch",
		expectation:        false,
		expectedErr:        errors.New("Validator failed as event type does not not match"),
	}
	configs["pull-request-valid"] = test_configuration{
		requiredRepo:       "http://github.com/foo/bar",
		requiredEvent:      "pull_request, Merge Request Hook",
		requiredAction:     "opened, reopened, synchronize",
		webhookURL:         "http://github.com/foo/bar",
		webhookEventHeader: "X-Github-Event",
		webhookEvent:       "pull_request",
		webhookPRAction:    "reopened",
		triggerName:        "pull-request-valid",
		expectation:        true,
		expectedErr:        nil,
	}
	configs["pull-request-action-mismatch"] = test_configuration{
		requiredRepo:       "http://github.com/foo/bar",
		requiredEvent:      "pull_request, Merge Request Hook",
		requiredAction:     "opened, reopened, synchronize",
		webhookURL:         "http://github.com/foo/bar",
		webhookEventHeader: "X-Github-Event",
		webhookEvent:       "pull_request",
		webhookPRAction:    "labelled",
		triggerName:        "pull-request-action-mismatch",
		expectation:        false,
		expectedErr:        errors.New("Validator failed as action does not not match"),
	}

	request, _ := http.NewRequest("POST", "", strings.NewReader("foo"))
	for _, tt := range configs {
		request.Header["Wext-Repository-Url"] = []string{tt.requiredRepo}
		request.Header["Wext-Incoming-Event"] = []string{tt.requiredEvent}
		request.Header["Wext-Incoming-Actions"] = []string{tt.requiredAction}
		request.Header[tt.webhookEventHeader] = []string{tt.webhookEvent}
		result, err := Validate(request, tt.webhookURL, tt.webhookEventHeader, tt.webhookPRAction, tt.triggerName)
		if tt.expectation != result {
			t.Errorf("Failure validating trigger: %+s, expected %+v but received %+v", tt.triggerName, tt.expectation, result)
		}
		if tt.expectedErr != nil {
			if tt.expectedErr.Error() != err.Error() {
				t.Errorf("Failure validating trigger: %+s, expected error: `%+v` but received: `%+v`", tt.triggerName, tt.expectedErr.Error(), err.Error())
			}
		} else {
			if err != nil {
				t.Errorf("Failure validating trigger: %+s, expected no error but received: `%+v`", tt.triggerName, err.Error())
			}
		}
	}
}

func getGitlabMergeRequest() string {

	//Example API payload
	raw := `{
		"object_kind": "merge_request",
		"event_type": "merge_request",
		"user": {
			"name": "Administrator",
			"username": "root",
			"avatar_url": "https://secure.gravatar.com/avatar/e64c7d89f26bd1972efa854d13d7dd61?s=80&d=identicon"
		},
		"project": {
			"id": 2,
			"name": "project2",
			"description": "",
			"web_url": "https://gitlab.apps.domain.com/root/project2",
			"avatar_url": null,
			"git_ssh_url": "git@gitlab.apps.domain.com:root/project2.git",
			"git_http_url": "https://gitlab.apps.domain.com/root/project2.git",
			"namespace": "Administrator",
			"visibility_level": 20,
			"path_with_namespace": "root/project2",
			"default_branch": "master",
			"ci_config_path": null,
			"homepage": "https://gitlab.apps.domain.com/root/project2",
			"url": "git@gitlab.apps.domain.com:root/project2.git",
			"ssh_url": "git@gitlab.apps.domain.com:root/project2.git",
			"http_url": "https://gitlab.apps.domain.com/root/project2.git"
		},
		"object_attributes": {
			"assignee_id": null,
			"author_id": 1,
			"created_at": "2020-01-20 10:09:58 UTC",
			"description": "",
			"head_pipeline_id": 28,
			"id": 7,
			"iid": 4,
			"last_edited_at": null,
			"last_edited_by_id": null,
			"merge_commit_sha": null,
			"merge_error": null,
			"merge_params": {
				"force_remove_source_branch": "1"
			},
			"merge_status": "cannot_be_merged",
			"merge_user_id": null,
			"merge_when_pipeline_succeeds": false,
			"milestone_id": null,
			"source_branch": "test",
			"source_project_id": 2,
			"state": "opened",
			"target_branch": "foo",
			"target_project_id": 2,
			"time_estimate": 0,
			"title": "Test",
			"updated_at": "2020-01-29 16:32:31 UTC",
			"updated_by_id": null,
			"url": "https://gitlab.apps.domain.com/root/project2/merge_requests/4",
			"source": {
				"id": 2,
				"name": "project2",
				"description": "",
				"web_url": "https://gitlab.apps.domain.com/root/project2",
				"avatar_url": null,
				"git_ssh_url": "git@gitlab.apps.domain.com:root/project2.git",
				"git_http_url": "https://gitlab.apps.domain.com/root/project2.git",
				"namespace": "Administrator",
				"visibility_level": 20,
				"path_with_namespace": "root/project2",
				"default_branch": "master",
				"ci_config_path": null,
				"homepage": "https://gitlab.apps.domain.com/root/project2",
				"url": "git@gitlab.apps.domain.com:root/project2.git",
				"ssh_url": "git@gitlab.apps.domain.com:root/project2.git",
				"http_url": "https://gitlab.apps.domain.com/root/project2.git"
			},
			"target": {
				"id": 2,
				"name": "project2",
				"description": "",
				"web_url": "https://gitlab.apps.domain.com/root/project2",
				"avatar_url": null,
				"git_ssh_url": "git@gitlab.apps.domain.com:root/project2.git",
				"git_http_url": "https://gitlab.apps.domain.com/root/project2.git",
				"namespace": "Administrator",
				"visibility_level": 20,
				"path_with_namespace": "root/project2",
				"default_branch": "master",
				"ci_config_path": null,
				"homepage": "https://gitlab.apps.domain.com/root/project2",
				"url": "git@gitlab.apps.domain.com:root/project2.git",
				"ssh_url": "git@gitlab.apps.domain.com:root/project2.git",
				"http_url": "https://gitlab.apps.domain.com/root/project2.git"
			},
			"last_commit": {
				"id": "123456747baf002968e375b6d11d4de412af5550",
				"message": "Update README.md",
				"timestamp": "2020-01-21T16:18:01Z",
				"url": "https://gitlab.apps.domain.com/root/project2/commit/123456747baf002968e375b6d11d4de412af5550",
				"author": {
					"name": "Administrator",
					"email": "admin@example.com"
				}
			},
			"work_in_progress": false,
			"total_time_spent": 0,
			"human_total_time_spent": null,
			"human_time_estimate": null,
			"assignee_ids": [
	
			],
			"action": "reopen"
		},
		"labels": [
	
		],
		"changes": {
			"state": {
				"previous": "closed",
				"current": "opened"
			},
			"updated_at": {
				"previous": "2020-01-29 16:32:25 UTC",
				"current": "2020-01-29 16:32:31 UTC"
			},
			"total_time_spent": {
				"previous": null,
				"current": 0
			}
		},
		"repository": {
			"name": "project2",
			"url": "git@gitlab.apps.domain.com:root/project2.git",
			"description": "",
			"homepage": "https://gitlab.apps.domain.com/root/project2"
		}}`
	return raw
}
