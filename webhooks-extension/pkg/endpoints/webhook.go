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

/* Currently Webhooks (GitHubSource must exist in the install namespace only - as must the ConfigMap)
This means that any credentials must also exist in the install namespace. */

// Creates a webhook for a given repository and populates (creating if doesn't yet exist) a ConfigMap storing this information
func (r Resource) createWebhook(request *restful.Request, response *restful.Response) {
	modifyingConfigMapLock.Lock()
	defer modifyingConfigMapLock.Unlock()

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
		err := errors.New("a namespace for creating a webhook is required, but none was given")
		logging.Log.Errorf("error: %s.", err.Error())
		RespondError(response, err, http.StatusBadRequest)
		return
	}

	logging.Log.Infof("Creating webhook: %v.", webhook)
	pieces := strings.Split(webhook.GitRepositoryURL, "/")
	if len(pieces) < 4 {
		logging.Log.Errorf("error creating webhook: GitRepositoryURL format error (%+v).", webhook.GitRepositoryURL)
		RespondError(response, errors.New("GitRepositoryURL format error"), http.StatusBadRequest)
		return
	}

	gitServer := strings.TrimSuffix(webhook.GitRepositoryURL, pieces[len(pieces)-2]+"/"+pieces[len(pieces)-1])
	logging.Log.Debugf("Webhook will create pipelineruns in namespace %s", namespace)
	apiURL := gitServer + "api/v3/"
	ownerRepo := pieces[len(pieces)-2] + "/" + strings.TrimSuffix(pieces[len(pieces)-1], ".git")

	hooks, err := r.getGitHubWebhooks(gitServer + ownerRepo)

	if len(hooks) > 0 {
		for _, hook := range hooks {
			if hook.Name == webhook.Name && hook.Namespace == webhook.Namespace {
				logging.Log.Errorf("error creating webhook: A webhook already exists for GitRepositoryURL %+v with the Name %s and Namespace %s.", webhook.GitRepositoryURL, webhook.Name, webhook.Namespace)
				RespondError(response, errors.New("Webhook already exists for the specified Git repository with the same name, targeting the same namespace"), http.StatusBadRequest)
				return
			}
			if hook.Pipeline == webhook.Pipeline && hook.Namespace == webhook.Namespace {
				logging.Log.Errorf("error creating webhook: A webhook already exists for GitRepositoryURL %+v, running pipeline %s in namespace %s.", webhook.GitRepositoryURL, webhook.Pipeline, webhook.Namespace)
				RespondError(response, errors.New("Webhook already exists for the specified Git repository, running the same pipeline in the same namespace"), http.StatusBadRequest)
				return
			}
			if hook.PullTask != webhook.PullTask {
				msg := fmt.Sprintf("PullTask mismatch. Webhooks on a repository must use the same PullTask existing webhooks use %s not %s.", hook.PullTask, webhook.PullTask)
				logging.Log.Errorf("error creating webhook: " + msg)
				RespondError(response, errors.New(msg), http.StatusBadRequest)
				return
			}
		}
		webhook.GithubSource = hooks[0].GithubSource
	} else {
		logging.Log.Debugf("Creating GitHub source with apiURL: %s and Owner-repo: %s.", apiURL, ownerRepo)

		entry := eventapi.GitHubSource{
			ObjectMeta: metav1.ObjectMeta{GenerateName: "tekton-"},
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
		ghs, err := r.EventSrcClient.SourcesV1alpha1().GitHubSources(installNs).Create(&entry)
		if err != nil {
			logging.Log.Errorf("Error creating GitHub source: %s.", err.Error())
			RespondError(response, err, http.StatusBadRequest)
			return
		}
		webhook.GithubSource = ghs.GetName()
	}

	webhooks, err := r.readGitHubWebhooks()
	if err != nil {
		logging.Log.Errorf("error getting GitHub webhooks: %s.", err.Error())
		RespondError(response, err, http.StatusInternalServerError)
		return
	}

	//store lowercase
	gitServerRepo := strings.ToLower(gitServer + ownerRepo)
	webhooks[gitServerRepo] = append(webhooks[gitServerRepo], webhook)
	logging.Log.Debugf("Writing the GitHubSource webhook ConfigMap in namespace %s", installNs)
	r.writeGitHubWebhooks(webhooks)
	response.WriteHeader(http.StatusCreated)
}

// Removes from ConfigMap, removes the actual GitHubSource, removes the webhook
func (r Resource) deleteWebhook(request *restful.Request, response *restful.Response) {
	modifyingConfigMapLock.Lock()
	defer modifyingConfigMapLock.Unlock()
	logging.Log.Debug("In deleteWebhook")
	name := request.PathParameter("name")
	repo := request.QueryParameter("repository")
	namespace := request.QueryParameter("namespace")
	deletePipelineRuns := request.QueryParameter("deletepipelineruns")

	var toDeletePipelineRuns = false
	var err error

	if deletePipelineRuns != "" {
		toDeletePipelineRuns, err = strconv.ParseBool(deletePipelineRuns)
		if err != nil {
			theError := errors.New("bad request information provided, cannot handle deletepipelineruns query (should be set to true or not provided)")
			logging.Log.Error(theError)
			RespondError(response, theError, http.StatusInternalServerError)
			return
		}
	}

	if namespace == "" || repo == "" {
		theError := errors.New("bad request information provided, a namespace and a repository must be specified as query parameters")
		logging.Log.Error(theError)
		RespondError(response, theError, http.StatusBadRequest)
		return
	}

	logging.Log.Debugf("in deleteWebhook, name: %s, repo: %s, delete pipeline runs: %s", name, repo, deletePipelineRuns)

	var remaining int
	webhooks, err := r.getGitHubWebhooks(repo)
	if err != nil {
		RespondError(response, err, http.StatusNotFound)
		return
	}

	logging.Log.Debugf("Found %d webhooks/pipelines registered against repo %s", len(webhooks), repo)
	if len(webhooks) < 1 {
		err := fmt.Errorf("no webhook found for repo %s", repo)
		logging.Log.Error(err)
		RespondError(response, err, http.StatusBadRequest)
		return
	}

	found := false
	for _, hook := range webhooks {
		if hook.Name == name && hook.Namespace == namespace {
			found = true
			if len(webhooks) == 1 {
				remaining = 0
				logging.Log.Debugf("No other pipelines triggered by this GitHub webhook, deleting githubsource")
				r.deleteGitHubSource(webhooks[0].GithubSource)
			} else {
				remaining = len(webhooks) - 1
			}
			if toDeletePipelineRuns {
				r.deletePipelineRuns(repo, namespace, hook.Pipeline)
			}
			r.deleteWebhookFromConfigMap(repo, name, namespace, remaining)
			response.WriteHeader(204)
		}
	}

	if !found {
		err := fmt.Errorf("no webhook found for repo %s with name %s associated with namespace %s", repo, name, namespace)
		logging.Log.Error(err)
		RespondError(response, err, http.StatusNotFound)
		return
	}

}

// Delete the webhook information from our ConfigMap
func (r Resource) deleteWebhookFromConfigMap(repository, webhookName, namespace string, remainingCount int) error {
	logging.Log.Debugf("Deleting webhook info named %s on repository %s running in namespace %s from ConfigMap", webhookName, repository, namespace)
	repository = strings.ToLower(strings.TrimSuffix(repository, ".git"))
	allHooks, err := r.readGitHubWebhooks()
	if err != nil {
		return err
	}

	if remainingCount > 0 {
		logging.Log.Debugf("Finding webhook for repository %s", repository)
		for i, hook := range allHooks[repository] {
			if hook.Name == webhookName && hook.Namespace == namespace {
				logging.Log.Debugf("Removing webhook from ConfigMap")
				allHooks[repository][i] = allHooks[repository][len(allHooks[repository])-1]
				allHooks[repository] = allHooks[repository][:len(allHooks[repository])-1]
			}
		}
	} else {
		logging.Log.Debugf("Deleting last webhook for repository %s", repository)
		delete(allHooks, repository)
	}

	err = r.writeGitHubWebhooks(allHooks)
	if err != nil {
		return err
	}
	return nil
}

func (r Resource) getAllWebhooks(request *restful.Request, response *restful.Response) {
	installNs := r.Defaults.Namespace
	if installNs == "" {
		installNs = "default"
	}

	logging.Log.Debugf("Get all webhooks")
	sources, err := r.readGitHubWebhooks()
	if err != nil {
		logging.Log.Errorf("error trying to get webhooks: %s.", err.Error())
		RespondError(response, err, http.StatusInternalServerError)
		return
	}
	sourcesList := []webhook{}
	for _, value := range sources {
		sourcesList = append(sourcesList, value...)
	}
	response.WriteEntity(sourcesList)

}

func (r Resource) deletePipelineRuns(gitRepoURL, namespace, pipeline string) error {
	logging.Log.Debugf("Looking for PipelineRuns in namespace %s with repository URL %s for pipeline %s", namespace, gitRepoURL, pipeline)

	allPipelineRuns, err := r.TektonClient.TektonV1alpha1().PipelineRuns(namespace).List(metav1.ListOptions{})

	if err != nil {
		logging.Log.Errorf("Unable to retrieve PipelineRuns in the namespace %s! Error: %s", namespace, err.Error())
		return err
	}

	found := false
	for _, pipelineRun := range allPipelineRuns.Items {
		if pipelineRun.Spec.PipelineRef.Name == pipeline {
			labels := pipelineRun.GetLabels()
			serverURL := labels["gitServer"]
			orgName := labels["gitOrg"]
			repoName := labels["gitRepo"]
			foundRepoURL := fmt.Sprintf("https://%s/%s/%s", serverURL, orgName, repoName)

			gitRepoURL = strings.ToLower(strings.TrimSuffix(gitRepoURL, ".git"))
			foundRepoURL = strings.ToLower(strings.TrimSuffix(foundRepoURL, ".git"))

			if foundRepoURL == gitRepoURL {
				found = true
				err := r.TektonClient.TektonV1alpha1().PipelineRuns(namespace).Delete(pipelineRun.Name, &metav1.DeleteOptions{})
				if err != nil {
					logging.Log.Errorf("failed to delete %s, error: %s", pipelineRun.Name, err.Error())
					return err
				}
				logging.Log.Infof("Deleted PipelineRun %s", pipelineRun.Name)
			}
		}
	}
	if !found {
		logging.Log.Infof("No matching PipelineRuns found")
	}
	return nil
}

func (r Resource) deleteGitHubSource(name string) error {
	logging.Log.Debugf("Deleting GitHub webhook with name %s", name)

	_, err := r.EventSrcClient.SourcesV1alpha1().GitHubSources(r.Defaults.Namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		logging.Log.Errorf("error fetching webhook: %s", err.Error())
		return fmt.Errorf("could not find webhook with name: %s", name)
	}

	err = r.EventSrcClient.SourcesV1alpha1().GitHubSources(r.Defaults.Namespace).Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		logging.Log.Errorf("error deleting webhook: %s", err.Error())
		return fmt.Errorf("could not delete webhook with name: %s", name)
	}

	return nil
}

// Retrieve registry secret, helm secret and pipeline name for the GitHub URL
func (r Resource) getGitHubWebhooks(gitRepoURL string) ([]webhook, error) {
	logging.Log.Debugf("Getting GitHub webhooks for repository URL %s", gitRepoURL)

	sources, err := r.readGitHubWebhooks()
	if err != nil {
		return []webhook{}, err
	}
	gitRepoURL = strings.ToLower(strings.TrimSuffix(gitRepoURL, ".git"))
	if sources[gitRepoURL] != nil {
		return sources[gitRepoURL], nil
	}

	return []webhook{}, fmt.Errorf("could not find webhook with GitRepositoryURL: %s", gitRepoURL)

}

func (r Resource) readGitHubWebhooks() (map[string][]webhook, error) {
	logging.Log.Debugf("Reading GitHub webhooks.")
	configMapClient := r.K8sClient.CoreV1().ConfigMaps(r.Defaults.Namespace)
	configMap, err := configMapClient.Get(ConfigMapName, metav1.GetOptions{})
	if err != nil {
		logging.Log.Debugf("Creating a new ConfigMap as an error occurred retrieving an existing one: %s.", err.Error())
		configMap = &corev1.ConfigMap{}
		configMap.BinaryData = make(map[string][]byte)
	}
	raw, ok := configMap.BinaryData["GitHubSource"]
	var result map[string][]webhook
	if ok {
		err = json.Unmarshal(raw, &result)
		if err != nil {
			logging.Log.Errorf("error unmarshalling in readGitHubSource: %s", err.Error())
			return map[string][]webhook{}, err
		}
	} else {
		result = make(map[string][]webhook)
	}
	return result, nil
}

func (r Resource) writeGitHubWebhooks(sources map[string][]webhook) error {
	logging.Log.Debugf("In writeGitHubWebhooks")
	configMapClient := r.K8sClient.CoreV1().ConfigMaps(r.Defaults.Namespace)
	configMap, err := configMapClient.Get(ConfigMapName, metav1.GetOptions{})
	var create = false
	if err != nil {
		configMap = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ConfigMapName,
				Namespace: r.Defaults.Namespace,
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
