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
	"errors"
	"fmt"
	gitlab "github.com/xanzy/go-gitlab"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	"log"
	"net/http"
	"strconv"
)

func HandleGitLab(request *http.Request, writer http.ResponseWriter, foundTriggerName string, secret *corev1.Secret) ([]byte, error) {

	var payload []byte
	if request.Header["X-Gitlab-Token"][0] != string(secret.Data["secretToken"]) {
		errorString := fmt.Sprintf("X-Gitlab-Token did not match the token stored in the secret: %s", secret.Name)
		return nil, errors.New(errorString)
	}

	var err error
	payload, err = ioutil.ReadAll(request.Body)
	if err != nil {
		log.Printf("[%s] Validation FAIL (error %s reading request body)", foundTriggerName, err.Error())
		return nil, err
	}

	event, err := gitlab.ParseWebhook(gitlab.WebhookEventType(request), payload)
	if err != nil {
		log.Printf("[%s] Validation FAIL (error %s parsing webhook)", foundTriggerName, err.Error())
		return nil, err
	}

	var projectURL, id, action string
	switch event := event.(type) {
	case *gitlab.PushEvent:
		projectURL = event.Repository.GitHTTPURL
		id = event.CheckoutSHA //cannot obtain webhook event id so will log commit
		action = ""
	case *gitlab.MergeEvent:
		projectURL = event.ObjectAttributes.Target.GitHTTPURL
		id = strconv.Itoa(event.ObjectAttributes.ID) //cannot obtain webhook event id so will log commit
		action = event.ObjectAttributes.State
	case *gitlab.TagEvent:
		projectURL = event.Repository.GitHTTPURL
		id = event.CheckoutSHA //cannot obtain webhook event id so will log commit
		action = ""
	default:
		log.Printf("[%s] Validation FAIL (unsupported gitlab event)", foundTriggerName)
		errorString := fmt.Sprintf("X-Gitlab-Event did not match any of the supported events")
		return nil, errors.New(errorString)
	}

	validationPassed, err := validateGitlab(request, foundTriggerName, projectURL, id, action)

	if validationPassed {
		returnPayload, err := addBranchAndTag(event)
		if err != nil {
			log.Printf("[%s] Failed to add branch to payload processing Gitlab event for commit ID: %s. Error: %s", foundTriggerName, id, err.Error())
			return nil, err
		}
		log.Printf("[%s] Validation PASS so writing response", foundTriggerName)
		return returnPayload, nil
	}
	return nil, errors.New("Validation Failed")
}

func validateGitlab(request *http.Request, foundTriggerName, projectURL, id, action string) (bool, error) {

	log.Printf("[%s] Project URL coming in as JSON: %s", foundTriggerName, projectURL)
	log.Printf("[%s] Handling GitLab Event for commit ID: %s", foundTriggerName, id)

	validationPassed, err := Validate(request, projectURL, "X-Gitlab-Event", action, foundTriggerName)
	if err != nil {
		if !validationPassed {
			return false, err
		}
	}
	return validationPassed, nil
}
