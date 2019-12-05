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
	"github.com/google/go-github/github"
	gitlab "github.com/xanzy/go-gitlab"
	"log"
	"net/http"
	"reflect"
	"strings"
)

const (
	RequiredRepositoryHeader = "Wext-Repository-Url"
	RequiredEventHeader      = "Wext-Incoming-Event"
	RequiredActionsHeader    = "Wext-Incoming-Actions"
)

type ghPushPayload struct {
	github.PushEvent
	WebhookBranch            string `json:"webhooks-tekton-git-branch"`
	WebhookSuggestedImageTag string `json:"webhooks-tekton-image-tag"`
}

type ghPullRequestPayload struct {
	github.PullRequestEvent
	WebhookBranch            string `json:"webhooks-tekton-git-branch"`
	WebhookSuggestedImageTag string `json:"webhooks-tekton-image-tag"`
}

type glPushPayload struct {
	gitlab.PushEvent
	WebhookBranch            string `json:"webhooks-tekton-git-branch"`
	WebhookSuggestedImageTag string `json:"webhooks-tekton-image-tag"`
}

type glPullRequestPayload struct {
	gitlab.MergeEvent
	WebhookBranch            string `json:"webhooks-tekton-git-branch"`
	WebhookSuggestedImageTag string `json:"webhooks-tekton-image-tag"`
}

type glTagPayload struct {
	gitlab.TagEvent
	WebhookBranch            string `json:"webhooks-tekton-git-branch"`
	WebhookSuggestedImageTag string `json:"webhooks-tekton-image-tag"`
}

func Validate(request *http.Request, httpsCloneURL, eventHeader, pullRequestAction, foundTriggerName string) (bool, error) {

	wantedRepoURL := request.Header.Get(RequiredRepositoryHeader)
	wantedActions := request.Header[RequiredActionsHeader]
	wantedEvents := request.Header[RequiredEventHeader]

	if sanitizeGitInput(httpsCloneURL) == sanitizeGitInput(wantedRepoURL) {
		if request.Header.Get(RequiredEventHeader) != "" {
			foundEvent := request.Header.Get(eventHeader)
			events := strings.Split(wantedEvents[0], ",")
			eventMatch := false
			for _, event := range events {
				if strings.TrimSpace(event) == foundEvent {
					eventMatch = true
					if len(wantedActions) == 0 {
						log.Printf("[%s] Validation PASS (repository URL, secret payload, event type checked)", foundTriggerName)
						return true, nil
					} else {
						actions := strings.Split(wantedActions[0], ",")
						for _, action := range actions {
							if strings.TrimSpace(action) == pullRequestAction {
								log.Printf("[%s] Validation PASS (repository URL, secret payload, event type, action:%s checked)", foundTriggerName, action)
								return true, nil
							}
						}
					}
				}
			}
			if !eventMatch {
				log.Printf("[%s] Validation FAIL (event type does not match, got %s but wanted one of %s)", foundTriggerName, foundEvent, wantedEvents)
				return false, errors.New("Validator failed as event type does not not match")
			}
			if len(wantedActions) > 0 {
				log.Printf("[%s] Validation FAIL (action type does not match, got %s but wanted one of %s)", foundTriggerName, pullRequestAction, wantedActions)
				return false, errors.New("Validator failed as action does not not match")
			}
			log.Printf("[%s] Validation FAIL (unable to match attributes)", foundTriggerName)
			return false, errors.New("Validator failed")
		}
		// Repository URL matches and no event type restrictions active
		log.Printf("[%s] Validation PASS (repository URL and secret payload checked)", foundTriggerName)
		return true, nil
	}

	log.Printf("[%s] Validation FAIL (repository URLs do not match, got %s but wanted %s)", foundTriggerName, sanitizeGitInput(httpsCloneURL), sanitizeGitInput(wantedRepoURL))
	return false, errors.New("Validator failed as repository URLs do not match")

}

func addBranchAndTag(webhookEvent interface{}) ([]byte, error) {
	switch event := webhookEvent.(type) {
	case github.PushEvent:
		toReturn := ghPushPayload{
			PushEvent:                event,
			WebhookBranch:            event.GetRef()[strings.LastIndex(event.GetRef(), "/")+1:],
			WebhookSuggestedImageTag: getSuggestedTag(event.GetRef(), *event.HeadCommit.ID),
		}
		return json.Marshal(toReturn)
	case github.PullRequestEvent:
		ref := event.GetPullRequest().GetHead().GetRef()
		toReturn := ghPullRequestPayload{
			PullRequestEvent:         event,
			WebhookBranch:            ref[strings.LastIndex(ref, "/")+1:],
			WebhookSuggestedImageTag: getSuggestedTag(ref, *event.PullRequest.Head.SHA),
		}
		return json.Marshal(toReturn)
	case *gitlab.PushEvent:
		ref := event.Ref
		toReturn := glPushPayload{
			PushEvent:                *event,
			WebhookBranch:            ref[strings.LastIndex(ref, "/")+1:],
			WebhookSuggestedImageTag: getSuggestedTag(ref, event.CheckoutSHA),
		}
		return json.Marshal(toReturn)
	case *gitlab.MergeEvent:
		ref := event.ObjectAttributes.TargetBranch
		toReturn := glPullRequestPayload{
			MergeEvent:               *event,
			WebhookBranch:            ref,
			WebhookSuggestedImageTag: getSuggestedTag(ref, event.ObjectAttributes.LastCommit.ID),
		}
		return json.Marshal(toReturn)
	case *gitlab.TagEvent:
		ref := event.Ref
		toReturn := glTagPayload{
			TagEvent:                 *event,
			WebhookBranch:            ref[strings.LastIndex(ref, "/")+1:],
			WebhookSuggestedImageTag: getSuggestedTag(ref, event.CheckoutSHA),
		}
		return json.Marshal(toReturn)
	default:
		msg := fmt.Sprintf("Unsupported event type `%s` received in addBranchAndTag()", reflect.TypeOf(webhookEvent))
		return []byte{}, errors.New(msg)
	}
}

func getSuggestedTag(ref, commit string) string {
	var suggestedImageTag string
	if strings.HasPrefix(ref, "refs/tags/") {
		suggestedImageTag = ref[strings.LastIndex(ref, "/")+1:]
	} else {
		suggestedImageTag = commit[0:7]
	}
	return suggestedImageTag
}

func sanitizeGitInput(input string) string {
	asLower := strings.ToLower(input)
	noGitSuffix := strings.TrimSuffix(asLower, ".git")
	noHTTPSPrefix := strings.TrimPrefix(noGitSuffix, "https://")
	noHTTPrefix := strings.TrimPrefix(noHTTPSPrefix, "http://")
	return noHTTPrefix
}
