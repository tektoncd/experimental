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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	logging "github.com/tektoncd/experimental/webhooks-extension/pkg/logging"

	restful "github.com/emicklei/go-restful"
	eventapi "github.com/knative/eventing-sources/pkg/apis/sources/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var modifyingConfigMapLock sync.Mutex

// Creates a webhook for a given repository and populates (creating if doesn't yet exist) a ConfigMap storing this information
func (r Resource) createWebhook(request *restful.Request, response *restful.Response) {
	modifyingConfigMapLock.Lock()
	defer modifyingConfigMapLock.Unlock()

	if request == nil {
		logging.Log.Fatal("nil request in createWebhook, it's all over")
	}

	logging.Log.Infof("Creating webhook with request: %+v.", request)
	installNs := r.Defaults.Namespace
	if installNs == "" {
		installNs = "default"
	}

	webhook := webhook{}
	if err := request.ReadEntity(&webhook); err != nil {
		logging.Log.Errorf("error trying to read request entity as webhook: %s.", err)
		RespondError(response, err, http.StatusBadRequest)
		return
	}

	if webhook.ReleaseName != "" {
		if len(webhook.ReleaseName) > 63 {
			tooLongMessage := fmt.Sprintf("requested release name (%s) must be less than 64 characters", webhook.ReleaseName)
			err := errors.New(tooLongMessage)
			logging.Log.Errorf("error: %s", err.Error())
			RespondError(response, err, http.StatusBadRequest)
			return
		}
	}

	dockerRegDefault := r.Defaults.DockerRegistry
	if webhook.DockerRegistry == "" && dockerRegDefault != "" {
		webhook.DockerRegistry = dockerRegDefault
	}
	logging.Log.Debugf("Docker registry location is: %s", webhook.DockerRegistry)

	namespace := webhook.Namespace
	if namespace == "" {
		err := errors.New("namespace is required, but none was given")
		logging.Log.Errorf("error: %s.", err.Error())
		RespondError(response, err, http.StatusBadRequest)
		return
	}

	logging.Log.Debugf("Webhook to be created in namespace %s", namespace)

	logging.Log.Infof("Creating webhook: %v.", webhook)
	pieces := strings.Split(webhook.GitRepositoryURL, "/")
	if len(pieces) < 4 {
		logging.Log.Errorf("error creating webhook: GitRepositoryURL format error (%+v).", webhook.GitRepositoryURL)
		RespondError(response, errors.New("GitRepositoryURL format error"), http.StatusBadRequest)
		return
	}
	apiURL := strings.TrimSuffix(webhook.GitRepositoryURL, pieces[len(pieces)-2]+"/"+pieces[len(pieces)-1]) + "api/v3/"
	ownerRepo := pieces[len(pieces)-2] + "/" + strings.TrimSuffix(pieces[len(pieces)-1], ".git")

	logging.Log.Debugf("Creating GitHub source with apiURL: %s and Owner-repo: %s.", apiURL, ownerRepo)

	entry := eventapi.GitHubSource{
		ObjectMeta: metav1.ObjectMeta{Name: webhook.Name},
		Spec: eventapi.GitHubSourceSpec{
			OwnerAndRepository: ownerRepo,
			EventTypes:         []string{"push", "pull_request"},
			AccessToken: eventapi.SecretValueFromSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					Key: "accessToken",
					LocalObjectReference: corev1.LocalObjectReference{
						Name: webhook.AccessTokenRef,
					},
				},
			},
			SecretToken: eventapi.SecretValueFromSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					Key: "secretToken",
					LocalObjectReference: corev1.LocalObjectReference{
						Name: webhook.AccessTokenRef,
					},
				},
			},
			Sink: &corev1.ObjectReference{
				APIVersion: "serving.knative.dev/v1alpha1",
				Kind:       "Service",
				Name:       "webhooks-extension-sink",
			},
		},
	}
	if c := strings.Count(apiURL, "."); c == 2 {
		entry.Spec.GitHubAPIURL = apiURL
	} else if c != 1 {
		err := fmt.Errorf("parsing git api url '%s'", apiURL)
		logging.Log.Errorf("Error %s", err.Error())
		RespondError(response, err, http.StatusBadRequest)
		return
	}
	_, err := r.EventSrcClient.SourcesV1alpha1().GitHubSources(installNs).Create(&entry)
	if err != nil {
		logging.Log.Errorf("Error creating GitHub source: %s.", err.Error())
		RespondError(response, err, http.StatusBadRequest)
		return
	}
	webhooks, err := r.readGitHubWebhooks(installNs)
	if err != nil {
		logging.Log.Errorf("error getting GitHub webhooks: %s.", err.Error())
		RespondError(response, err, http.StatusInternalServerError)
		return
	}
	webhooks[webhook.Name] = webhook
	logging.Log.Debugf("Writing the GitHubSource webhook ConfigMap in namespace %s", installNs)
	r.writeGitHubWebhooks(installNs, webhooks)
	response.WriteHeader(http.StatusCreated)
}

// Removes from ConfigMap, removes the actual GitHubSource, removes the webhook
func (r Resource) deleteWebhook(request *restful.Request, response *restful.Response) {
	modifyingConfigMapLock.Lock()
	defer modifyingConfigMapLock.Unlock()
	logging.Log.Debug("In deleteWebhook")
	name := request.PathParameter("name")
	namespace := request.QueryParameter("namespace")
	deletePipelineRuns := request.QueryParameter("deletepipelineruns")

	var toDeletePipelineRuns = false
	var err error

	if deletePipelineRuns != "" {
		toDeletePipelineRuns, err = strconv.ParseBool(deletePipelineRuns)
		if err != nil {
			theError := errors.New("bad request information provided, cannot handle deletepipelineruns query (should be set to true or not provided)")
			logging.Log.Error(theError)
			RespondError(response, err, http.StatusInternalServerError)
			return
		}
	}

	if namespace == "" {
		theError := errors.New("no namespace was provided")
		logging.Log.Error(theError)
		RespondError(response, theError, http.StatusBadRequest)
		return
	}

	logging.Log.Debugf("in deleteWebhook, name: %s, namespace: %s, delete pipeline runs: %s", name, namespace, deletePipelineRuns)

	if name != "" {
		foundRepoURL := r.findRepoURLFromConfigMap(name, namespace)
		// This will remove any PipelineRuns too if specified to, hence the need for the repository URL
		err = r.deleteGitHubWebhookByName(name, namespace, foundRepoURL, toDeletePipelineRuns)
		if err != nil {
			if strings.Contains(err.Error(), "could not find webhook with name") {
				logging.Log.Errorf("webhook (name %s) not found in namespace %s", name, namespace)
				RespondError(response, err, http.StatusNotFound)
				return
			}
			logging.Log.Errorf("error deleting the webhook (name %s), error: %s", name, err)
			RespondError(response, err, http.StatusInternalServerError)
			return

		} else {
			logging.Log.Infof("Deleted the webhook %s OK, deleting from ConfigMap next", name)
		}

		err = r.deleteWebhookFromConfigMapByName(name, namespace)
		if err != nil {
			logging.Log.Errorf("error deleting the webhook information (name %s) from the ConfigMap, error: %s", name, err)
			RespondError(response, err, http.StatusInternalServerError)
			return
		}
	} else {
		logging.Log.Error("no name was provided")
		RespondError(response, err, http.StatusBadRequest)
		return
	}
	response.WriteHeader(204)
}

// Find the repository URL for a webhook. We store webhook in a ConfigMap
func (r Resource) findRepoURLFromConfigMap(webhookName, namespace string) string {
	foundURL := ""
	configMapClient := r.K8sClient.CoreV1().ConfigMaps(namespace)

	configMap, err := configMapClient.Get(ConfigMapName, metav1.GetOptions{})
	if err != nil {
		logging.Log.Errorf("couldn't find ConfigMap %s in namespace %s", ConfigMapName, namespace)
		return ""
	}

	data := configMap.BinaryData["GitHubSource"]

	itsANestedMap := map[string]map[string]string{}
	err = json.Unmarshal(data, &itsANestedMap)
	if err != nil {
		logging.Log.Errorf("invalid data format in the ConfigMap, can't modify it, error: %s", err.Error())
	}

	foundURL = itsANestedMap[webhookName]["gitrepositoryurl"]
	return foundURL
}

// Delete the webhook information from our ConfigMap
func (r Resource) deleteWebhookFromConfigMapByName(webhookName, namespace string) error {
	logging.Log.Debug("Deleting webhook info from ConfigMap")

	configMapClient := r.K8sClient.CoreV1().ConfigMaps(namespace)

	configMap, err := configMapClient.Get(ConfigMapName, metav1.GetOptions{})
	if err != nil {
		logging.Log.Errorf("couldn't find the ConfigMap %s in namespace %s", ConfigMapName, namespace)
		return err
	} else {
		logging.Log.Debugf("Found and deleting webhook info %s from %s", webhookName, configMap.Name)
	}

	/* Contains for example, base64 encoded:
	{
		confiMap.BinaryData[GitHubSource]
		"myhook":
		{
			"name":"myhook",
			"namespace":"default",
			"gitrepositoryurl":"https://myrepourl",
			"accesstoken":"my-secret-for-hooks",
			"pipeline":"simple-helm-pipeline-insecure",
			"dockerregistry":"adamroberts"
		}
	}
	*/

	data := configMap.BinaryData["GitHubSource"]

	itsANestedMap := map[string]map[string]string{}
	err = json.Unmarshal(data, &itsANestedMap)
	if err != nil {
		logging.Log.Errorf("invalid data format in the ConfigMap, can't modify it, error: %s", err.Error())
	}

	delete(itsANestedMap, webhookName)

	itsANestedMapAsBytes, err := json.Marshal(itsANestedMap)
	if err != nil {
		logging.Log.Errorf("error marshalling the nested map, error is: %s", err.Error())
	}

	configMap.BinaryData["GitHubSource"] = itsANestedMapAsBytes

	_, err = configMapClient.Update(configMap)
	if err != nil {
		logging.Log.Errorf("error updating ConfigMap for GitHub webhooks: %s.", err.Error())
	}
	return nil
}

func (r Resource) getAllWebhooks(request *restful.Request, response *restful.Response) {
	installNs := r.Defaults.Namespace
	if installNs == "" {
		installNs = "default"
	}

	logging.Log.Debugf("Get all webhooks in namespace: %s.", installNs)
	sources, err := r.readGitHubWebhooks(installNs)
	if err != nil {
		logging.Log.Errorf("error trying to get webhooks: %s.", err.Error())
		RespondError(response, err, http.StatusInternalServerError)
		return
	}
	sourcesList := []webhook{}
	for _, value := range sources {
		sourcesList = append(sourcesList, value)
	}
	response.WriteEntity(sourcesList)
}

/* Tekton dashboard/extension created PipelineRuns are labelled, e.g. with:
"gitOrg": "myorg"
"gitRepo": "myrepo"
"gitServer": "github.com"
Delete them if the repository URL is what's provided as a parameter. */

func (r Resource) deletePipelineRunsByRepoURL(gitRepoURL, namespace string) error {
	logging.Log.Debugf("Deleting PipelineRuns in namespace %s with repository URL %s", namespace, gitRepoURL)

	allPipelineRuns, err := r.TektonClient.TektonV1alpha1().PipelineRuns(namespace).List(metav1.ListOptions{})

	if err != nil {
		logging.Log.Errorf("Unable to retrieve PipelineRuns in the namespace %s! Error: %s", namespace, err.Error())
		return err
	}

	for _, pipelineRun := range allPipelineRuns.Items {
		labels := pipelineRun.GetLabels()
		serverURL := labels["gitServer"]
		orgName := labels["gitOrg"]
		repoName := labels["gitRepo"]
		foundRepoURL := fmt.Sprintf("https://%s/%s/%s", serverURL, orgName, repoName)

		gitRepoURL = strings.ToLower(gitRepoURL)
		foundRepoURL = strings.ToLower(foundRepoURL)

		if foundRepoURL == gitRepoURL {
			err := r.TektonClient.TektonV1alpha1().PipelineRuns(namespace).Delete(pipelineRun.Name, &metav1.DeleteOptions{})
			if err != nil {
				logging.Log.Errorf("failed to delete %s, error: %s", pipelineRun.Name, err.Error())
				return err
			}
			logging.Log.Infof("Deleted PipelineRun %s", pipelineRun.Name)
		}
	}
	// All is good
	return nil
}

/* Delete a GitHubSource based on its name and namespace, returns no error (nil) if all is OK.
   Optionally deletes PipelineRuns too. This method does not delete from the ConfigMap, that's done as an additional step. */

func (r Resource) deleteGitHubWebhookByName(name, namespace, foundRepoURL string, deletePipelineRuns bool) error {
	logging.Log.Debugf("Deleting GitHub webhook in namespace %s with name %s", namespace, name)

	foundRealGitHubSource, err := r.EventSrcClient.SourcesV1alpha1().GitHubSources(namespace).Get(name, metav1.GetOptions{})

	if err != nil {
		return fmt.Errorf("could not find webhook with name: %s", name)
	}

	err = r.EventSrcClient.SourcesV1alpha1().GitHubSources(namespace).Delete(foundRealGitHubSource.Name, &metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("could not delete webhook with name: %s", name)
	}

	if deletePipelineRuns {
		err := r.deletePipelineRunsByRepoURL(foundRepoURL, namespace)
		if err != nil {
			return fmt.Errorf("could not delete the pipelineruns associated with repo URL %s in namespace %s", foundRepoURL, namespace)
		}
		logging.Log.Infof("Deleted PipelineRuns OK")

	} else {
		logging.Log.Info("Preserving PipelineRuns")
	}
	// All good
	return nil
}

// Retrieve registry secret, helm secret and pipeline name for the GitHub URL
func (r Resource) getGitHubWebhook(gitRepoURL string, namespace string) (webhook, error) {
	logging.Log.Debugf("Getting GitHub webhook in namespace %s with repository URL %s", namespace, gitRepoURL)

	sources, err := r.readGitHubWebhooks(namespace)
	if err != nil {
		return webhook{}, err
	}
	for _, source := range sources {
		if strings.TrimSuffix(strings.ToLower(source.GitRepositoryURL), ".git") == strings.TrimSuffix(strings.ToLower(gitRepoURL), ".git") {
			return source, nil
		}
	}
	return webhook{}, fmt.Errorf("could not find webhook with GitRepositoryURL: %s", gitRepoURL)
}

func (r Resource) readGitHubWebhooks(namespace string) (map[string]webhook, error) {
	logging.Log.Debugf("Reading GitHub webhooks in namespace %s.", namespace)
	configMapClient := r.K8sClient.CoreV1().ConfigMaps(namespace)
	configMap, err := configMapClient.Get(ConfigMapName, metav1.GetOptions{})
	if err != nil {
		logging.Log.Debugf("Creating a new ConfigMap as an error occurred retrieving an existing one: %s.", err.Error())
		configMap = &corev1.ConfigMap{}
		configMap.BinaryData = make(map[string][]byte)
	}
	raw, ok := configMap.BinaryData["GitHubSource"]
	var result map[string]webhook
	if ok {
		err = json.Unmarshal(raw, &result)
		if err != nil {
			logging.Log.Errorf("error unmarshalling in readGitHubSource: %s", err.Error())
			return map[string]webhook{}, err
		}
	} else {
		result = make(map[string]webhook)
	}
	return result, nil
}

func (r Resource) writeGitHubWebhooks(namespace string, sources map[string]webhook) error {
	logging.Log.Debugf("In writeGitHubWebhooks, namespace: %s", namespace)
	configMapClient := r.K8sClient.CoreV1().ConfigMaps(namespace)
	configMap, err := configMapClient.Get(ConfigMapName, metav1.GetOptions{})
	var create = false
	if err != nil {
		configMap = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ConfigMapName,
				Namespace: namespace,
			},
		}
		configMap.BinaryData = make(map[string][]byte)
		create = true
	}
	buf, err := json.Marshal(sources)
	if err != nil {
		logging.Log.Errorf("error marshalling GitHub webhooks: %s.", err.Error())
		return err
	}
	configMap.BinaryData["GitHubSource"] = buf
	if create {
		_, err = configMapClient.Create(configMap)
		if err != nil {
			logging.Log.Errorf("error creating configmap for GitHub webhooks: %s.", err.Error())
			return err
		}
	} else {
		_, err = configMapClient.Update(configMap)
		if err != nil {
			logging.Log.Errorf("error updating configmap for GitHub webhooks: %s.", err.Error())
		}
	}
	return nil
}

func (r Resource) getDefaults(request *restful.Request, response *restful.Response) {
	logging.Log.Debugf("getDefaults returning: %v", r.Defaults)
	response.WriteEntity(r.Defaults)
}

// RespondError ...
func RespondError(response *restful.Response, err error, statusCode int) {
	logging.Log.Errorf("Error for RespondError: %s.", err.Error())
	logging.Log.Errorf("Response is %v.", *response)
	response.AddHeader("Content-Type", "text/plain")
	response.WriteError(statusCode, err)
}

// RespondErrorMessage ...
func RespondErrorMessage(response *restful.Response, message string, statusCode int) {
	logging.Log.Errorf("Message for RespondErrorMessage: %s.", message)
	response.AddHeader("Content-Type", "text/plain")
	response.WriteErrorString(statusCode, message)
}

// RespondErrorAndMessage ...
func RespondErrorAndMessage(response *restful.Response, err error, message string, statusCode int) {
	logging.Log.Errorf("Error for RespondErrorAndMessage: %s.", err.Error())
	logging.Log.Errorf("Message for RespondErrorAndMesage: %s.", message)
	response.AddHeader("Content-Type", "text/plain")
	response.WriteErrorString(statusCode, message)
}

// RegisterExtensionWebService registers the webhook webservice
func (r Resource) RegisterExtensionWebService(container *restful.Container) {
	ws := new(restful.WebService)
	ws.
		Path("/webhooks").
		Consumes(restful.MIME_JSON, restful.MIME_JSON).
		Produces(restful.MIME_JSON, restful.MIME_JSON)

	ws.Route(ws.POST("/").To(r.createWebhook))
	ws.Route(ws.GET("/").To(r.getAllWebhooks))
	ws.Route(ws.GET("/defaults").To(r.getDefaults))
	ws.Route(ws.DELETE("/{name}").To(r.deleteWebhook))

	ws.Route(ws.POST("/credentials").To(r.createCredential))
	ws.Route(ws.GET("/credentials").To(r.getAllCredentials))
	ws.Route(ws.DELETE("/credentials/{name}").To(r.deleteCredential))

	container.Add(ws)
}

// RegisterWeb registers extension web bundle on the container
func (r Resource) RegisterWeb(container *restful.Container) {
	var handler http.Handler
	webResourcesDir := os.Getenv("WEB_RESOURCES_DIR")
	koDataPath := os.Getenv("KO_DATA_PATH")
	_, err := os.Stat(webResourcesDir)
	if err != nil {
		if os.IsNotExist(err) {
			if koDataPath != "" {
				logging.Log.Warnf("WEB_RESOURCES_DIR %s not found, serving static content from KO_DATA_PATH instead.", webResourcesDir)
				handler = http.FileServer(http.Dir(koDataPath))
			} else {
				logging.Log.Errorf("WEB_RESOURCES_DIR %s not found and KO_DATA_PATH not found, static resource (UI) problems to be expected.", webResourcesDir)
			}
		} else {
			logging.Log.Errorf("error returned while checking for WEB_RESOURCES_DIR %s", webResourcesDir)
		}
	} else {
		logging.Log.Infof("Serving static files from WEB_RESOURCES_DIR: %s", webResourcesDir)
		handler = http.FileServer(http.Dir(webResourcesDir))
	}
	container.Handle("/web/", http.StripPrefix("/web/", handler))
}
