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
	"github.com/google/go-github/github"
	"log"
	"net/http"

	corev1 "k8s.io/api/core/v1"
)

func HandleGitHub(request *http.Request, writer http.ResponseWriter, foundTriggerName string, secret *corev1.Secret) ([]byte, error) {

	payload, err := github.ValidatePayload(request, secret.Data["secretToken"])
	if err != nil {
		log.Printf("[%s] Validation FAIL (error %s validating payload)", foundTriggerName, err.Error())
		return nil, err
	}

	event := request.Header.Get("X-Github-Event")
	if event != "" {
		switch {
		case event == "push":
			return handlePush(request, writer, foundTriggerName, payload)
		case event == "pull_request":
			return handlePull(request, writer, foundTriggerName, payload)
		}
	}

	return nil, errors.New("Unsupported Github event received")
}

func handlePush(request *http.Request, writer http.ResponseWriter, foundTriggerName string, payload []byte) ([]byte, error) {
	var hookPayload github.PushEvent
	err := json.Unmarshal(payload, &hookPayload)
	if err != nil {
		log.Printf("[%s] Validation FAIL (error %s marshalling payload as JSON)", foundTriggerName, err.Error())
		return nil, err
	}

	cloneURL := hookPayload.Repo.GetCloneURL()
	log.Printf("[%s] Clone URL coming in as JSON: %s", foundTriggerName, cloneURL)

	id := github.DeliveryID(request)
	log.Printf("[%s] Handling GitHub Event with delivery ID: %s", foundTriggerName, id)

	validationPassed, err := Validate(request, cloneURL, "X-Github-Event", "", foundTriggerName)
	if err != nil {
		if !validationPassed {
			return nil, err
		}
	}

	if validationPassed {
		returnPayload, err := addBranchAndTag(hookPayload)
		if err != nil {
			log.Printf("[%s] Failed to add branch to payload processing Github event ID: %s. Error: %s", foundTriggerName, id, err.Error())
			return nil, err
		}
		log.Printf("[%s] Validation PASS so writing response", foundTriggerName)
		return returnPayload, nil
	} else {
		// In theory we will have returned if there was an error so just state validation failed
		return nil, errors.New("Validation Failed")
	}
}

func handlePull(request *http.Request, writer http.ResponseWriter, foundTriggerName string, payload []byte) ([]byte, error) {
	var hookPayload github.PullRequestEvent
	err := json.Unmarshal(payload, &hookPayload)
	if err != nil {
		log.Printf("[%s] Validation FAIL (error %s marshalling payload as JSON)", foundTriggerName, err.Error())
		return nil, err
	}

	cloneURL := hookPayload.Repo.GetCloneURL()
	log.Printf("[%s] Clone URL coming in as JSON: %s", foundTriggerName, cloneURL)

	id := github.DeliveryID(request)
	log.Printf("[%s] Handling GitHub Event with delivery ID: %s", foundTriggerName, id)

	validationPassed, err := Validate(request, cloneURL, "X-Github-Event", *hookPayload.Action, foundTriggerName)
	if err != nil {
		if !validationPassed {
			return nil, err
		}
	}

	if validationPassed {
		returnPayload, err := addBranchAndTag(hookPayload)
		if err != nil {
			log.Printf("[%s] Failed to add branch to payload processing Github event ID: %s. Error: %s", foundTriggerName, id, err.Error())
			return nil, err
		}
		log.Printf("[%s] Validation PASS so writing response", foundTriggerName)
		return returnPayload, nil
	} else {
		// In theory we will have returned if there was an error so just state validation failed
		return nil, errors.New("Validation Failed")
	}
}
