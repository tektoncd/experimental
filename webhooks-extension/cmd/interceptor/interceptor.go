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
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/google/go-github/github"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	envSecret = "GITHUB_SECRET_TOKEN"
)

type Result struct {
	Action     string `json:"action"`
	Repository struct {
		CloneURL string `json:"clone_url"`
	} `json:"repository"`
}

type PushPayload struct {
	github.PushEvent
	WebhookBranch string `json:"webhooks-tekton-git-branch"`
}

type PullRequestPayload struct {
	github.PullRequestEvent
	WebhookBranch string `json:"webhooks-tekton-git-branch"`
}

func main() {
	log.Print("Interceptor started")

	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		foundTriggerName := request.Header.Get("Wext-Trigger-Name")

		config, err := rest.InClusterConfig()
		if err != nil {
			log.Printf("[%s] Error creating in cluster config: %s", foundTriggerName, err.Error())
			http.Error(writer, fmt.Sprint(err), http.StatusInternalServerError)
			return
		}

		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			log.Printf("[%s] Error creating new clientset: %s", foundTriggerName, err.Error())
			http.Error(writer, fmt.Sprint(err), http.StatusInternalServerError)
			return
		}

		foundNamespace := os.Getenv("INSTALLED_NAMESPACE")
		foundSecretName := request.Header.Get("Wext-Secret-Name")

		foundSecret, err := clientset.CoreV1().Secrets(foundNamespace).Get(foundSecretName, metav1.GetOptions{})

		if err != nil {
			log.Printf("[%s] Error getting the secret %s to validate: %s", foundTriggerName, foundSecretName, err.Error())
			http.Error(writer, fmt.Sprint(err), http.StatusBadRequest)
			return
		}

		wantedRepoURL := request.Header.Get("Wext-Repository-Url")

		payload, err := github.ValidatePayload(request, foundSecret.Data["secretToken"])
		if err != nil {
			log.Printf("[%s] Validation FAIL (error %s validating payload)", foundTriggerName, err.Error())
			http.Error(writer, fmt.Sprint(err), http.StatusExpectationFailed)
			return
		}

		var result Result
		err = json.Unmarshal(payload, &result)
		if err != nil {
			log.Printf("[%s] Validation FAIL (error %s marshalling payload as JSON)", foundTriggerName, err.Error())
			http.Error(writer, fmt.Sprint(err), http.StatusInternalServerError)
			return
		}

		cloneURL := result.Repository.CloneURL
		log.Printf("[%s] Clone URL coming in as JSON: %s", foundTriggerName, cloneURL)

		id := github.DeliveryID(request)
		if err != nil {
			log.Printf("[%s] Error handling GitHub Event with delivery ID %s: %s", foundTriggerName, id, err.Error())
			http.Error(writer, fmt.Sprint(err), http.StatusInternalServerError)
			return
		}

		log.Printf("[%s] Handling GitHub Event with delivery ID: %s", foundTriggerName, id)

		validationPassed := false

		if sanitizeGitInput(cloneURL) == sanitizeGitInput(wantedRepoURL) {
			if request.Header.Get("Wext-Incoming-Event") != "" {
				wantedEvent := request.Header.Get("Wext-Incoming-Event")
				foundEvent := request.Header.Get("X-Github-Event")
				if wantedEvent == foundEvent { // Wanted GitHub event type provided AND repository URL matches so all is well
					wantedActions := request.Header["Wext-Incoming-Actions"]
					if len(wantedActions) == 0 {
						validationPassed = true
						log.Printf("[%s] Validation PASS (repository URL, secret payload, event type checked)", foundTriggerName)
					} else {
						actions := strings.Split(wantedActions[0], ",")
						for _, action := range actions {
							if action == result.Action {
								validationPassed = true
								log.Printf("[%s] Validation PASS (repository URL, secret payload, event type, action:%s checked)", foundTriggerName, action)
							}
						}
					}
				} else {
					log.Printf("[%s] Validation FAIL (event type does not match, got %s but wanted %s)", foundTriggerName, foundEvent, wantedEvent)
					http.Error(writer, fmt.Sprint(err), http.StatusExpectationFailed)
					return
				}
			} else { // No wanted GitHub event type provided, but the repository URL matches so all is well
				log.Printf("[%s] Validation PASS (repository URL and secret payload checked)", foundTriggerName)
				validationPassed = true
			}

			if validationPassed {
				returnPayload, err := addBranchToPayload(request.Header.Get("X-Github-Event"), payload)
				if err != nil {
					log.Printf("[%s] Failed to add branch to payload processing Github event ID: %s. Error: %s", foundTriggerName, id, err.Error())
					http.Error(writer, fmt.Sprint(err), http.StatusInternalServerError)
					return
				}

				log.Printf("[%s] Validation PASS so writing response", foundTriggerName)
				_, err = writer.Write(returnPayload)
				if err != nil {
					log.Printf("[%s] Failed to write response for Github event ID: %s. Error: %s", foundTriggerName, id, err.Error())
					http.Error(writer, fmt.Sprint(err), http.StatusInternalServerError)
					return
				}
			} else {
				http.Error(writer, "Validation failed", http.StatusExpectationFailed)
			}
		} else {
			log.Printf("[%s] Validation FAIL (repository URL does not match, got %s but wanted %s): ",
				foundTriggerName,
				sanitizeGitInput(cloneURL),
				sanitizeGitInput(wantedRepoURL))

			http.Error(writer, fmt.Sprint(err), http.StatusExpectationFailed)
			return
		}
	})

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", 8080), nil))
}

func addBranchToPayload(event string, payload []byte) ([]byte, error) {
	if "push" == event {
		var toReturn PushPayload
		var p github.PushEvent
		err := json.Unmarshal(payload, &p)
		if err != nil {
			return nil, err
		}
		toReturn = PushPayload{
			PushEvent:     p,
			WebhookBranch: p.GetRef()[strings.LastIndex(p.GetRef(), "/")+1:],
		}
		return json.Marshal(toReturn)
	} else if "pull_request" == event {
		var toReturn PullRequestPayload
		var pr github.PullRequestEvent
		err := json.Unmarshal(payload, &pr)
		if err != nil {
			return nil, err
		}
		ref := pr.GetPullRequest().GetHead().GetRef()
		toReturn = PullRequestPayload{
			PullRequestEvent: pr,
			WebhookBranch:    ref[strings.LastIndex(ref, "/")+1:],
		}
		return json.Marshal(toReturn)
	} else {
		return payload, nil
	}
}

func sanitizeGitInput(input string) string {
	noGitSuffix := strings.TrimSuffix(input, ".git")
	asLower := strings.ToLower(noGitSuffix)
	noHTTPSPrefix := strings.TrimPrefix(asLower, "https://")
	noHTTPrefix := strings.TrimPrefix(noHTTPSPrefix, "http://")
	return noHTTPrefix
}
