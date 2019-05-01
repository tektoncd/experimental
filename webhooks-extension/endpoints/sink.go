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
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	restful "github.com/emicklei/go-restful"
	v1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	gh "gopkg.in/go-playground/webhooks.v3/github"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const gitServerLabel = "gitServer"
const gitOrgLabel = "gitOrg"
const gitRepoLabel = "gitRepo"
const githubEventParameter = "Ce-Github-Event"

// BuildInformation - information required to build a particular commit from a Git repository.
type BuildInformation struct {
	REPOURL        string
	SHORTID        string
	COMMITID       string
	REPONAME       string
	TIMESTAMP      string
	SERVICEACCOUNT string
}

// handleWebhook should be called when we hit the / endpoint with webhook data. Todo provide proper responses e.g. 503, server errors, 200 if good
func (r Resource) handleWebhook(request *restful.Request, response *restful.Response) {
	log.Print("In HandleWebhook for a GitHub event.")
	buildInformation := BuildInformation{}
	log.Printf("Github event name to look for is: %s.", githubEventParameter)
	gitHubEventType := request.HeaderParameter(githubEventParameter)

	if len(gitHubEventType) < 1 {
		log.Printf("Error found header (%s) exists but has no value. Request is: %+v.", githubEventParameter, request)
		return
	}

	gitHubEventTypeString := strings.Replace(gitHubEventType, "\"", "", -1)

	log.Printf("GitHub event type is: %s.", gitHubEventTypeString)

	timestamp := getDateTimeAsString()

	if gitHubEventTypeString == "ping" {
		response.WriteHeader(http.StatusNoContent)
	} else if gitHubEventTypeString == "push" {
		log.Print("Handling a push event.")

		webhookData := gh.PushPayload{}

		if err := request.ReadEntity(&webhookData); err != nil {
			log.Printf("Error decoding webhook data: %s.", err.Error())
			return
		}

		buildInformation.REPOURL = webhookData.Repository.URL
		buildInformation.SHORTID = webhookData.HeadCommit.ID[0:7]
		buildInformation.COMMITID = webhookData.HeadCommit.ID
		buildInformation.REPONAME = webhookData.Repository.Name
		buildInformation.TIMESTAMP = timestamp

		createPipelineRunFromWebhookData(buildInformation, r)
		log.Printf("Build information for repository %s:%s: %s.", buildInformation.REPOURL, buildInformation.SHORTID, buildInformation)

	} else if gitHubEventTypeString == "pull_request" {
		log.Print("Handling a pull request event.")

		webhookData := gh.PullRequestPayload{}

		if err := request.ReadEntity(&webhookData); err != nil {
			log.Printf("Error decoding webhook data: %s.", err.Error())
			return
		}

		buildInformation.REPOURL = webhookData.Repository.HTMLURL
		buildInformation.SHORTID = webhookData.PullRequest.Head.Sha[0:7]
		buildInformation.COMMITID = webhookData.PullRequest.Head.Sha
		buildInformation.REPONAME = webhookData.Repository.Name
		buildInformation.TIMESTAMP = timestamp

		createPipelineRunFromWebhookData(buildInformation, r)
		log.Printf("Build information for repository %s:%s: %s.", buildInformation.REPOURL, buildInformation.SHORTID, buildInformation)

	} else {
		log.Printf("Error: event wasn't a push, pull, or ping event, no action will be taken. Request is: %+v.", request)
	}
}

// This is the main flow that handles building and deploying: given everything we need to kick off a build, do so
func createPipelineRunFromWebhookData(buildInformation BuildInformation, r Resource) {
	log.Printf("In createPipelineRunFromWebhookData, build information: %s.", buildInformation)

	// TODO: Use the dashboard endpoint to create the PipelineRun
	// Track PR: https://github.com/tektoncd/dashboard/pull/33
	// and issue: https://github.com/tektoncd/dashboard/issues/47

	// Install namespace
	installNs := os.Getenv("INSTALLED_NAMESPACE")
	if installNs == "" {
		installNs = "default"
	}

	log.Printf("Looking for the pipeline configmap in the install namespace %s.", installNs)

	// get information from related githubsource instance
	webhook, err := r.getGitHubWebhook(buildInformation.REPOURL, installNs)
	if err != nil {
		log.Printf("Error getting github webhook: %s.", err.Error())
		return
	}
	dockerRegistry := webhook.DockerRegistry
	helmSecret := webhook.HelmSecret
	pipelineTemplateName := webhook.Pipeline
	pipelineNs := webhook.Namespace
	saName := webhook.ServiceAccount
	if saName == "" {
		saName = "default"
	}

	log.Printf("Build information: %+v.", buildInformation)

	// Assumes you've already applied the yml: so the pipeline definition and its tasks must exist upfront.
	startTime := getDateTimeAsString()
	generatedPipelineRunName := fmt.Sprintf("%s-%s", webhook.Name, startTime)

	// Unique names are required so timestamp them.
	imageResourceName := fmt.Sprintf("%s-docker-image-%s", webhook.Name, startTime)
	gitResourceName := fmt.Sprintf("%s-git-source-%s", webhook.Name, startTime)

	pipeline, err := r.getPipelineImpl(pipelineTemplateName, pipelineNs)
	if err != nil {
		log.Printf("Could not find the pipeline template %s in namespace %s.", pipelineTemplateName, pipelineNs)
		return
	}
	log.Printf("Found the pipeline template %s OK.", pipelineTemplateName)

	log.Print("Creating PipelineResources.")

	urlToUse := fmt.Sprintf("%s/%s:%s", dockerRegistry, strings.ToLower(buildInformation.REPONAME), buildInformation.SHORTID)
	log.Printf("Image URL is: %s.", urlToUse)

	paramsForImageResource := []v1alpha1.Param{{Name: "url", Value: urlToUse}}
	pipelineImageResource := definePipelineResource(imageResourceName, pipelineNs, paramsForImageResource, "image")
	createdPipelineImageResource, err := r.TektonClient.TektonV1alpha1().PipelineResources(pipelineNs).Create(pipelineImageResource)
	if err != nil {
		log.Printf("Error creating pipeline image resource to be used in the pipeline: %s.", err.Error())
		return
	}
	log.Printf("Created pipeline image resource %s successfully.", createdPipelineImageResource.Name)

	paramsForGitResource := []v1alpha1.Param{{Name: "revision", Value: buildInformation.COMMITID}, {Name: "url", Value: buildInformation.REPOURL}}
	pipelineGitResource := definePipelineResource(gitResourceName, pipelineNs, paramsForGitResource, "git")
	createdPipelineGitResource, err := r.TektonClient.TektonV1alpha1().PipelineResources(pipelineNs).Create(pipelineGitResource)

	if err != nil {
		log.Printf("Error creating pipeline git resource to be used in the pipeline: %s.", err.Error())
		return
	}
	log.Printf("Created pipeline git resource %s successfully.", createdPipelineGitResource.Name)

	gitResourceRef := v1alpha1.PipelineResourceRef{Name: gitResourceName}
	imageResourceRef := v1alpha1.PipelineResourceRef{Name: imageResourceName}

	resources := []v1alpha1.PipelineResourceBinding{{Name: "docker-image", ResourceRef: imageResourceRef}, {Name: "git-source", ResourceRef: gitResourceRef}}

	imageTag := buildInformation.SHORTID
	imageName := fmt.Sprintf("%s/%s", dockerRegistry, strings.ToLower(buildInformation.REPONAME))
	releaseName := fmt.Sprintf("%s-%s", strings.ToLower(buildInformation.REPONAME), buildInformation.SHORTID)
	repositoryName := strings.ToLower(buildInformation.REPONAME)
	params := []v1alpha1.Param{{Name: "image-tag", Value: imageTag},
		{Name: "image-name", Value: imageName},
		{Name: "release-name", Value: releaseName},
		{Name: "repository-name", Value: repositoryName},
		{Name: "target-namespace", Value: pipelineNs}}

	if dockerRegistry != "" {
		params = append(params, v1alpha1.Param{Name: "docker-registry", Value: dockerRegistry})
	}
	if helmSecret != "" {
		params = append(params, v1alpha1.Param{Name: "helm-secret", Value: helmSecret})
	}

	// PipelineRun yml defines the references to the above named resources.
	pipelineRunData, err := definePipelineRun(generatedPipelineRunName, pipelineNs, saName, buildInformation.REPOURL,
		pipeline, v1alpha1.PipelineTriggerTypeManual, resources, params)

	log.Printf("Creating a new PipelineRun named %s in the namespace %s using the service account %s.", generatedPipelineRunName, pipelineNs, saName)

	pipelineRun, err := r.TektonClient.TektonV1alpha1().PipelineRuns(pipelineNs).Create(pipelineRunData)
	if err != nil {
		log.Printf("Error creating the PipelineRun: %s", err.Error())
		return
	}
	log.Printf("PipelineRun created: %+v.", pipelineRun)
}

/* Get all pipelines in a given namespace: the caller needs to handle any errors,
an empty v1alpha1.Pipeline{} is returned if no pipeline is found */
func (r Resource) getPipelineImpl(name, namespace string) (v1alpha1.Pipeline, error) {
	log.Printf("In getPipelineImpl, name %s, namespace %s.", name, namespace)

	pipelines := r.TektonClient.TektonV1alpha1().Pipelines(namespace)
	pipeline, err := pipelines.Get(name, metav1.GetOptions{})
	if err != nil {
		log.Printf("Error revreiving the pipeline called %s in namespace %s: %s.", name, namespace, err.Error())
		return v1alpha1.Pipeline{}, err
	}
	log.Print("Found the pipeline definition OK.")
	return *pipeline, nil
}

/* Create a new PipelineResource: this should be of type git or image */
func definePipelineResource(name, namespace string, params []v1alpha1.Param, resourceType v1alpha1.PipelineResourceType) *v1alpha1.PipelineResource {
	pipelineResource := v1alpha1.PipelineResource{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: v1alpha1.PipelineResourceSpec{
			Type:   resourceType,
			Params: params,
		},
	}
	resourcePointer := &pipelineResource
	return resourcePointer
}

/* Create a new PipelineRun - repoUrl, resourceBinding and params can be nill depending on the Pipeline
each PipelineRun has a 1 hour timeout: */
func definePipelineRun(pipelineRunName, namespace, saName, repoURL string,
	pipeline v1alpha1.Pipeline,
	triggerType v1alpha1.PipelineTriggerType,
	resourceBinding []v1alpha1.PipelineResourceBinding,
	params []v1alpha1.Param) (*v1alpha1.PipelineRun, error) {

	gitServer, gitOrg, gitRepo := "", "", ""
	err := errors.New("")
	if repoURL != "" {
		gitServer, gitOrg, gitRepo, err = getGitValues(repoURL)
		if err != nil {
			log.Printf("Error getting the Git values: %s.", err)
			return &v1alpha1.PipelineRun{}, err
		}
	}

	pipelineRunData := v1alpha1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pipelineRunName,
			Namespace: namespace,
			Labels: map[string]string{
				"app":          "devops-knative",
				gitServerLabel: gitServer,
				gitOrgLabel:    gitOrg,
				gitRepoLabel:   gitRepo,
			},
		},

		Spec: v1alpha1.PipelineRunSpec{
			PipelineRef: v1alpha1.PipelineRef{Name: pipeline.Name},
			// E.g. v1alpha1.PipelineTriggerTypeManual
			Trigger:        v1alpha1.PipelineTrigger{Type: triggerType},
			ServiceAccount: saName,
			Timeout:        &metav1.Duration{Duration: 1 * time.Hour},
			Resources:      resourceBinding,
			Params:         params,
		},
	}
	pipelineRunPointer := &pipelineRunData
	return pipelineRunPointer, nil
}

// Returns the git server excluding transport, org and repo
func getGitValues(url string) (gitServer, gitOrg, gitRepo string, err error) {
	repoURL := ""
	if url != "" {
		url = strings.ToLower(url)
		if strings.Contains(url, "https://") {
			repoURL = strings.TrimPrefix(url, "https://")
		} else {
			repoURL = strings.TrimPrefix(url, "http://")
		}
	}

	// example at this point: github.com/tektoncd/pipeline
	numSlashes := strings.Count(repoURL, "/")
	if numSlashes < 2 {
		return "", "", "", errors.New("Url didn't match the requirements (at least two slashes)")
	}
	repoURL = strings.TrimSuffix(repoURL, "/")

	gitServer = repoURL[0:strings.Index(repoURL, "/")]
	gitOrg = repoURL[strings.Index(repoURL, "/")+1 : strings.LastIndex(repoURL, "/")]
	gitRepo = repoURL[strings.LastIndex(repoURL, "/")+1:]

	return gitServer, gitOrg, gitRepo, nil
}

func getDateTimeAsString() string {
	return strconv.FormatInt(time.Now().Unix(), 10)
}

// SinkWebService returns the liveness web service
func SinkWebService(r Resource) *restful.WebService {
	ws := new(restful.WebService)
	ws.
		Path("/")
	ws.Route(ws.POST("").To(r.handleWebhook))

	return ws
}
