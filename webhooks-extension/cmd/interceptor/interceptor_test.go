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

package main

import (
	"encoding/json"
	"github.com/google/go-github/github"
	"testing"
)

func TestAddBranchToPushPayload(t *testing.T) {

	ref := "refs/head/master"
	pushPayloadStruct := github.PushEvent{
		Ref: &ref,
	}
	payload, err := json.Marshal(pushPayloadStruct)
	if err != nil {
		t.Errorf("Error in json.Marshal(pushPayloadStruct) %s", err)
	}

	bytes, err := addBranchToPayload("push", payload)
	if err != nil {
		t.Errorf("Error in addBranchToPayload %s", err)
	}

	var p PushPayload
	err = json.Unmarshal(bytes, &p)
	if err != nil {
		t.Errorf("Error in json.Unmarshal %s", err)
	}

	if "master" != p.WebhookBranch {
		t.Errorf("Branch name not added as expected, branch was returned as %s", p.WebhookBranch)
	}

}

func TestAddBranchToPullRequestPayload(t *testing.T) {

	ref := "refs/head/master"
	pullrequestPayloadStruct := github.PullRequestEvent{
		PullRequest: &github.PullRequest{
			Head: &github.PullRequestBranch{
				Ref: &ref,
			},
		},
	}

	payload, err := json.Marshal(pullrequestPayloadStruct)
	if err != nil {
		t.Errorf("Error in json.Marshal(pullrequestPayloadStruct) %s", err)
	}

	bytes, err := addBranchToPayload("pull_request", payload)
	if err != nil {
		t.Errorf("Error in addBranchToPayload %s", err)
	}

	var p PullRequestPayload
	err = json.Unmarshal(bytes, &p)
	if err != nil {
		t.Errorf("Error in json.Unmarshal %s", err)
	}

	if "master" != p.WebhookBranch {
		t.Errorf("Branch name not added as expected, branch was returned as %s", p.WebhookBranch)
	}

}

func TestAddBranchToOtherEventPayload(t *testing.T) {

	eventPayloadStruct := github.PingEvent{}

	payload, err := json.Marshal(eventPayloadStruct)
	if err != nil {
		t.Errorf("Error in json.Marshal(eventPayloadStruct) %s", err)
	}

	bytes, err := addBranchToPayload("ping", payload)
	if err != nil {
		t.Errorf("Error in addBranchToPayload %s", err)
	}

	// Should be unchanged
	var p github.PingEvent
	err = json.Unmarshal(bytes, &p)
	if err != nil {
		t.Errorf("Error in json.Unmarshal - bytes may have been modified in addBranchToPayload, this should only be done for push and pull_request, %s", err)
	}

}
