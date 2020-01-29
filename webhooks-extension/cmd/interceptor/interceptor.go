/*
 Copyright 2019, 2020 The Tekton Authors
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
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	envSecret = "GITHUB_SECRET_TOKEN"
)

func main() {
	log.Print("Interceptor started")
	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {

		// Copy Headers from the request onto the response
		for k, valueArray := range request.Header {
			for _, v := range valueArray {
				writer.Header().Add(k, v)
			}
		}

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

		url, err := url.Parse(request.Header["Wext-Repository-Url"][0])
		if err != nil {
			log.Printf("[%s] Error parsing repository url %s: %s", foundTriggerName, request.Header["Wext-Repository-Url"][0], err.Error())
			http.Error(writer, fmt.Sprint(err), http.StatusBadRequest)
			return
		}

		var returnPayload []byte
		switch {
		case request.Header["X-Github-Event"] != nil:
			expectingGithub := strings.Contains(url.Host, "github")
			if !expectingGithub {
				msg := fmt.Sprintf("[%s] Validation FAIL (provider mismatch - webhook is from GitHub)", foundTriggerName)
				log.Print(msg)
				http.Error(writer, msg, http.StatusExpectationFailed)
				return
			}
			returnPayload, err = HandleGitHub(request, writer, foundTriggerName, foundSecret)
		case request.Header["X-Gitlab-Event"] != nil:
			expectingGitlab := strings.Contains(url.Host, "gitlab")
			if !expectingGitlab {
				msg := fmt.Sprintf("[%s] Validation FAIL (provider mismatch - webhook is from Gitlab)", foundTriggerName)
				log.Print(msg)
				http.Error(writer, msg, http.StatusExpectationFailed)
				return
			}
			returnPayload, err = HandleGitLab(request, writer, foundTriggerName, foundSecret)
		default:
			log.Print("Webhook did not contain either `X-Github-Event` or `X-Gitlab-Event` headers")
			http.Error(writer, fmt.Sprint(err), http.StatusExpectationFailed)
			return
		}

		if err != nil {
			http.Error(writer, fmt.Sprint(err), http.StatusExpectationFailed)
			return
		}

		_, err = writer.Write(returnPayload)
		if err != nil {
			log.Printf("[%s] Failed to write response. Error: %s", foundTriggerName, err.Error())
			http.Error(writer, fmt.Sprint(err), http.StatusInternalServerError)
		}

	})

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", 8080), nil))
}
