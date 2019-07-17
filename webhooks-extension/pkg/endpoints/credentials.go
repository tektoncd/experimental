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

package endpoints

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"

	restful "github.com/emicklei/go-restful"
	logging "github.com/tektoncd/experimental/webhooks-extension/pkg/logging"
	"github.com/tektoncd/experimental/webhooks-extension/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// 'credentials' from the webhooks-extension's point of view, are access tokens. That's the only sort we handle right now.
type credential struct {
	Name        string `json:"name"`
	Namespace   string `json:"namespace,omitempty"`
	AccessToken string `json:"accesstoken"`
	SecretToken string `json:"secrettoken,omitempty"`
}

/*--------------------------------------
This file implements three endpoints from webhooks.go:
	ws.Route(ws.POST("/credentials").To(r.createCredential))
	ws.Route(ws.GET("/credentials").To(r.getAllCredentials))
	ws.Route(ws.DELETE("/credentials/{name}").To(r.deleteCredential))
---------------------------------------*/

func (r Resource) createCredential(request *restful.Request, response *restful.Response) {
	logging.Log.Debug("In createCredential")
	cred := credential{}

	if err := getQueryEntity(&cred, request, response); err != nil {
		logging.Log.Errorf("Error processing query entity: %s", err.Error())
		return
	}

	if !r.verifyCredentialParameters(cred, response) {
		logging.Log.Error("Error verifying credential parameters")
		return
	}

	secret := credentialToSecret(cred, r.Defaults.Namespace, response)

	logging.Log.Debugf("Creating credential %s in namespace %s", cred.Name, r.Defaults.Namespace)

	if _, err := r.K8sClient.CoreV1().Secrets(r.Defaults.Namespace).Create(secret); err != nil {
		errorMessage := fmt.Sprintf("error creating secret in K8sClient: %s", err.Error())
		utils.RespondMessageAndLogError(response, err, errorMessage, http.StatusBadRequest)
		return
	}
	writeResponseLocation(request, response, cred.Name)
}

func (r Resource) deleteCredential(request *restful.Request, response *restful.Response) {
	credName := request.PathParameter("name")
	if !r.verifySecretExists(credName, r.Defaults.Namespace, response) {
		return
	}
	err := r.K8sClient.CoreV1().Secrets(r.Defaults.Namespace).Delete(credName, &metav1.DeleteOptions{})
	if err != nil {
		errorMessage := fmt.Sprintf("error deleting secret from K8sClient: %s.", err.Error())
		utils.RespondMessageAndLogError(response, err, errorMessage, http.StatusInternalServerError)
		return
	}
	response.WriteHeader(204)
}

func (r Resource) getAllCredentials(request *restful.Request, response *restful.Response) {
	// Get secrets from the resource K8sClient
	secrets, err := r.K8sClient.CoreV1().Secrets(r.Defaults.Namespace).List(metav1.ListOptions{})

	if err != nil {
		errorMessage := fmt.Sprintf("error getting secrets from K8sClient: %s.", err.Error())
		response.WriteErrorString(http.StatusInternalServerError, errorMessage)
		logging.Log.Error(errorMessage)
		return
	}

	// Parse K8s secrets to credentials
	creds := []credential{}
	emptyCred := credential{}
	for _, secret := range secrets.Items {
		cred := secretToCredential(&secret, true)
		if cred != emptyCred {
			creds = append(creds, cred)
			logging.Log.Infof("getAllCredentials Found credential %+v\n", cred)
		}
	}

	logging.Log.Infof("getAllCredentials returning +%v", creds)

	// Write the response
	response.AddHeader("Content-Type", "application/json")
	response.WriteEntity(creds)
}

// Sends error message 404 if the secret does not exist in the resource K8sClient
func (r Resource) verifySecretExists(secretName string, namespace string, response *restful.Response) bool {
	_, err := r.K8sClient.CoreV1().Secrets(namespace).Get(secretName, metav1.GetOptions{})
	if err != nil {
		errorMessage := fmt.Sprintf("error getting secret from K8sClient: '%s'.", secretName)
		utils.RespondMessageAndLogError(response, err, errorMessage, http.StatusNotFound)
		return false
	}
	return true
}

// Convert credential struct into K8s secret struct
func credentialToSecret(cred credential, namespace string, response *restful.Response) *corev1.Secret {
	// Create new secret struct
	secret := corev1.Secret{}
	secret.Type = corev1.SecretTypeOpaque
	secret.SetNamespace(namespace)
	secret.SetName(cred.Name)
	secret.Data = make(map[string][]byte)
	secret.Data["accessToken"] = []byte(cred.AccessToken)
	if cred.SecretToken != "" {
		secret.Data["secretToken"] = []byte(cred.SecretToken)
	} else {
		secret.Data["secretToken"] = getRandomSecretToken()
	}
	return &secret
}

var (
	src = rand.NewSource(time.Now().UnixNano())
)

const (
	tokenBytes   = "123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	tokenIdxBits = 6                   // 6 bits = 2^6 = 64 characters in tokenBytes
	tokenIdxMask = 1<<tokenIdxBits - 1 // All 1-bits, as many as tokenIdxBits
)

// Generate a random 20-character string, returned as []byte.
// With thanks to https://medium.com/@kpbird/golang-generate-fixed-size-random-string-dd6dbd5e63c0
func getRandomSecretToken() []byte {
	b := make([]byte, 20)
	for i := 0; i < 20; {
		idx := int(src.Int63() & tokenIdxMask)
		if idx < len(tokenBytes) {
			b[i] = tokenBytes[idx]
			i++
		}
	}
	return b
}

// Convert K8s secret struct into credential struct
func secretToCredential(secret *corev1.Secret, mask bool) credential {
	var cred credential
	if secret.Data["accessToken"] != nil {
		cred = credential{
			Name:        secret.GetName(),
			Namespace:   secret.Namespace,
			AccessToken: string(secret.Data["accessToken"]),
			SecretToken: string(secret.Data["secretToken"]),
		}
		if mask {
			cred.AccessToken = "********"
		}
	}
	return cred
}

// Checks the Accept header and reads the content into the entityPointer.
func getQueryEntity(entityPointer interface{}, request *restful.Request, response *restful.Response) (err error) {
	if err := request.ReadEntity(entityPointer); err != nil {
		errorMessage := "error parsing request body."
		utils.RespondMessageAndLogError(response, err, errorMessage, http.StatusBadRequest)
		return err
	}
	return nil
}

func (r Resource) verifyCredentialParameters(cred credential, response *restful.Response) bool {
	errorMessage := ""
	if cred.Name == "" {
		errorMessage = fmt.Sprintf("error: Name must be specified")
	} else if cred.AccessToken == "" {
		errorMessage = fmt.Sprintf("error: AccessToken must be specified")
	}
	if errorMessage != "" {
		utils.RespondErrorMessage(response, errorMessage, http.StatusBadRequest)
		return false
	}
	return true
}

// Write Content-Location header within POST methods and set StatusCode to 201
// Headers MUST be set before writing to body (if any) to succeed
func writeResponseLocation(request *restful.Request, response *restful.Response, identifier string) {
	location := request.Request.URL.Path
	if request.Request.Method == http.MethodPost {
		location = location + "/" + identifier
	}
	response.AddHeader("Content-Location", location)
	response.WriteHeader(201)
}

// Returns true if namespace exists. Returns false and logs an HTTP error 400 if it does not.
func (r Resource) namespaceExists(namespace string, response *restful.Response) bool {
	_, err := r.K8sClient.CoreV1().Namespaces().Get(namespace, metav1.GetOptions{})
	if err != nil {
		errorMessage := fmt.Sprintf("error: namespace does not exist: '%s'.", namespace)
		utils.RespondMessageAndLogError(response, err, errorMessage, http.StatusBadRequest)
		return false
	}
	return true
}
