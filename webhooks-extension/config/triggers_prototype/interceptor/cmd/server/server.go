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
	"github.com/google/go-github/github"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"log"
	"net/http"
	"os"
	"strings"
)

const (
	envSecret = "GITHUB_SECRET_TOKEN"
)

type Result struct {
	Repository struct {
		CloneURL string `json:"clone_url"`
	} `json:"repository"`
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

		log.Printf("[%s] Handling GitHub Event with delivery ID: %s", foundTriggerName, id)

		validationPassed := false

		if sanitizeGitInput(cloneURL) == sanitizeGitInput(wantedRepoURL) {
			if request.Header.Get("Wext-Incoming-Event") != "" {
				wantedEvent := request.Header.Get("Wext-Incoming-Event")
				foundEvent := request.Header.Get("X-Github-Event")
				if wantedEvent == foundEvent { // Wanted GitHub event type provided AND repository URL matches so all is well
					validationPassed = true
					log.Printf("[%s] Validation PASS (repository URL, secret payload, event type checked)", foundTriggerName)
				} else {
					log.Printf("[%s] Validation FAIL (event type does not match, got %s but wanted %s)", foundTriggerName, foundEvent, wantedEvent)
					http.Error(writer, fmt.Sprint("event type mismatch"), http.StatusExpectationFailed)
					return
				}
			} else { // No wanted GitHub event type provided, but the repository URL matches so all is well
				log.Printf("[%s] Validation PASS (repository URL and secret payload checked)", foundTriggerName)
				validationPassed = true
			}

			if validationPassed {
				log.Printf("[%s] Validation PASS so writing response", foundTriggerName)
				_, err := writer.Write(payload)
				if err != nil {
					log.Printf("[%s] Failed to write response for Github event ID: %s. Error: %s", foundTriggerName, id, err.Error())
					http.Error(writer, fmt.Sprint(err), http.StatusInternalServerError)
					return
				}
			}
		} else {
			log.Printf("[%s] Validation FAIL (repository URL does not match, got %s but wanted %s): ",
				foundTriggerName,
				sanitizeGitInput(cloneURL),
				sanitizeGitInput(wantedRepoURL))

			http.Error(writer, fmt.Sprint("respository URL mismatch"), http.StatusExpectationFailed)
			return
		}
	})

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", 8080), nil))
}

func sanitizeGitInput(input string) string {
	noGitSuffix := strings.TrimSuffix(input, ".git")
	asLower := strings.ToLower(noGitSuffix)
	noHTTPSPrefix := strings.TrimPrefix(asLower, "https://")
	noHTTPrefix := strings.TrimPrefix(noHTTPSPrefix, "http://")
	return noHTTPrefix
}
