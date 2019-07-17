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
	restful "github.com/emicklei/go-restful"
	logging "github.com/tektoncd/experimental/webhooks-extension/pkg/logging"
	v1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	gh "gopkg.in/go-playground/webhooks.v3/github"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const gitServerLabel = "gitServer"
const gitOrgLabel = "gitOrg"
const gitRepoLabel = "gitRepo"
const gitBranchLabel = "gitBranch"
const githubEventParameter = "Ce-Github-Event"

// BuildInformation - information required to build a particular commit from a Git repository.
type BuildInformation struct {
	BRANCH         string
	REPOURL        string
	SHORTID        string
	COMMITID       string
	REPONAME       string
	TIMESTAMP      string
	SERVICEACCOUNT string
	PULLURL        string
}

// handleWebhook should be called when we hit the / endpoint with webhook data. Todo provide proper responses e.g. 503, server errors, 200 if good
func (r Resource) handleWebhook(request *restful.Request, response *restful.Response) {
	logging.Log.Info("In HandleWebhook for a GitHub event...")
	buildInformation := BuildInformation{}
	logging.Log.Infof("Github event name to look for is: %s.", githubEventParameter)
	gitHubEventType := request.HeaderParameter(githubEventParameter)

	if len(gitHubEventType) < 1 {
		logging.Log.Errorf("error found header (%s) exists but has no value. Request is: %+v.", githubEventParameter, request)
		return
	}

	gitHubEventTypeString := strings.Replace(gitHubEventType, "\"", "", -1)

	logging.Log.Debugf("GitHub event type is: %s.", gitHubEventTypeString)

	timestamp := getDateTimeAsString()

	if gitHubEventTypeString == "ping" {
		response.WriteHeader(http.StatusNoContent)
	} else if gitHubEventTypeString == "push" {
		logging.Log.Info("Handling a push event.")

		webhookData := gh.PushPayload{}

		if err := request.ReadEntity(&webhookData); err != nil {
			logging.Log.Errorf("error decoding webhook data: %s.", err.Error())
			return
		}

		buildInformation.REPOURL = webhookData.Repository.URL
		buildInformation.SHORTID = webhookData.HeadCommit.ID[0:7]
		buildInformation.COMMITID = webhookData.HeadCommit.ID
		buildInformation.REPONAME = webhookData.Repository.Name
		buildInformation.TIMESTAMP = timestamp
		buildInformation.BRANCH = extractBranchFromRef(webhookData.Ref)

		createPipelineRunFromWebhookData(buildInformation, r)
		logging.Log.Debugf("Build information for repository %s:%s: %s.", buildInformation.REPOURL, buildInformation.SHORTID, buildInformation)

	} else if gitHubEventTypeString == "pull_request" {
		logging.Log.Info("Handling a pull request event.")

		webhookData := gh.PullRequestPayload{}

		if err := request.ReadEntity(&webhookData); err != nil {
			logging.Log.Errorf("error decoding webhook data: %s.", err.Error())
			return
		}

		buildInformation.REPOURL = webhookData.Repository.HTMLURL
		buildInformation.SHORTID = webhookData.PullRequest.Head.Sha[0:7]
		buildInformation.COMMITID = webhookData.PullRequest.Head.Sha
		buildInformation.REPONAME = webhookData.Repository.Name
		buildInformation.PULLURL = strings.Replace(webhookData.PullRequest.URL, "api/v3/repos/", "", 1)
		buildInformation.TIMESTAMP = timestamp
		buildInformation.BRANCH = extractBranchFromRef(webhookData.PullRequest.Head.Ref)

		createPipelineRunFromWebhookData(buildInformation, r)
		logging.Log.Debugf("Build information for repository %s:%s: %s.", buildInformation.REPOURL, buildInformation.SHORTID, buildInformation)
		createTaskRunFromWebhookData(buildInformation, r)
		logging.Log.Debugf("created monitoring task for pipelinerun from build information for repository %s sha %s.", buildInformation.REPOURL, 
			buildInformation.SHORTID)
	} else {
		logging.Log.Errorf("error: event wasn't a push, pull, or ping event, no action will be taken. Request is: %+v.", request)
	}
}

func extractBranchFromRef(ref string) string {
	// ref typically resembles "refs/heads/branchhere", so extract "branchhere" from this string
	if ref != "" {
		if strings.Count(ref, "/") == 2 {
			lastIndex := strings.LastIndex(ref, "/")
			toReturn := ref[lastIndex+1:]
			logging.Log.Debugf("Determined branch from ref field: %s", toReturn)
			return toReturn
		}
	}
	logging.Log.Warnf("Couldn't determine the branch from ref field: %s", ref)
	return ""
}

// This is the main flow that handles building and deploying: given everything we need to kick off a build, do so
func createPipelineRunFromWebhookData(buildInformation BuildInformation, r Resource) {
	logging.Log.Debugf("In createPipelineRunFromWebhookData, build information: %s.", buildInformation)

	// TODO: Use the dashboard endpoint to create the PipelineRun
	// Track PR: https://github.com/tektoncd/dashboard/pull/33
	// and issue: https://github.com/tektoncd/dashboard/issues/47

	// Install namespace
	installNs := os.Getenv("INSTALLED_NAMESPACE")
	if installNs == "" {
		installNs = "default"
	}

	logging.Log.Debugf("Looking for the pipeline configmap in the install namespace %s.", installNs)

	// get information from related githubsource instance
	webhook, err := r.getGitHubWebhook(buildInformation.REPOURL, installNs)
	if err != nil {
		logging.Log.Errorf("error getting github webhook: %s.", err.Error())
		return
	}
	dockerRegistry := webhook.DockerRegistry
	helmSecret := webhook.HelmSecret
	pipelineTemplateName := webhook.Pipeline
	pipelineNs := webhook.Namespace
	saName := webhook.ServiceAccount
	requestedReleaseName := webhook.ReleaseName

	if saName == "" {
		saName = "default"
	}

	logging.Log.Debugf("Build information: %+v.", buildInformation)

	// Assumes you've already applied the yml: so the pipeline definition and its tasks must exist upfront.
	startTime := buildInformation.TIMESTAMP
	generatedPipelineRunName := fmt.Sprintf("%s-%s", webhook.Name, startTime)

	// Unique names are required so timestamp them.
	imageResourceName := fmt.Sprintf("%s-docker-image-%s", webhook.Name, startTime)
	gitResourceName := fmt.Sprintf("%s-git-source-%s", webhook.Name, startTime)

	pipeline, err := r.getPipelineImpl(pipelineTemplateName, pipelineNs)
	if err != nil {
		logging.Log.Errorf("could not find the pipeline template %s in namespace %s.", pipelineTemplateName, pipelineNs)
		return
	}
	logging.Log.Debugf("Found the pipeline template %s OK.", pipelineTemplateName)

	logging.Log.Debug("Creating PipelineResources.")

	urlToUse := fmt.Sprintf("%s/%s:%s", dockerRegistry, strings.ToLower(buildInformation.REPONAME), buildInformation.SHORTID)
	logging.Log.Debugf("Constructed image URL is: %s.", urlToUse)

	paramsForImageResource := []v1alpha1.Param{{Name: "url", Value: urlToUse}}
	pipelineImageResource := definePipelineResource(imageResourceName, pipelineNs, paramsForImageResource, nil, "image")
	createdPipelineImageResource, err := r.TektonClient.TektonV1alpha1().PipelineResources(pipelineNs).Create(pipelineImageResource)
	if err != nil {
		logging.Log.Errorf("error creating pipeline image resource to be used in the pipeline: %s.", err.Error())
		return
	}
	logging.Log.Infof("Created pipeline image resource %s successfully.", createdPipelineImageResource.Name)

	paramsForGitResource := []v1alpha1.Param{{Name: "revision", Value: buildInformation.COMMITID}, {Name: "url", Value: buildInformation.REPOURL}}
	pipelineGitResource := definePipelineResource(gitResourceName, pipelineNs, paramsForGitResource, nil, "git")
	createdPipelineGitResource, err := r.TektonClient.TektonV1alpha1().PipelineResources(pipelineNs).Create(pipelineGitResource)

	if err != nil {
		logging.Log.Errorf("error creating pipeline git resource to be used in the pipeline: %s.", err.Error())
		return
	}
	logging.Log.Infof("Created pipeline git resource %s successfully.", createdPipelineGitResource.Name)

	gitResourceRef := v1alpha1.PipelineResourceRef{Name: gitResourceName}
	imageResourceRef := v1alpha1.PipelineResourceRef{Name: imageResourceName}

	resources := []v1alpha1.PipelineResourceBinding{{Name: "docker-image", ResourceRef: imageResourceRef}, {Name: "git-source", ResourceRef: gitResourceRef}}

	imageTag := buildInformation.SHORTID
	imageName := fmt.Sprintf("%s/%s", dockerRegistry, strings.ToLower(buildInformation.REPONAME))

	releaseName := ""

	if requestedReleaseName != "" {
		logging.Log.Infof("Release name based on input: %s", requestedReleaseName)
		releaseName = requestedReleaseName
	} else {
		releaseName = fmt.Sprintf("%s", strings.ToLower(buildInformation.REPONAME))
		logging.Log.Infof("Release name based on repository name: %s", releaseName)
	}

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
	pipelineRunData, err := definePipelineRun(generatedPipelineRunName, pipelineNs, saName, buildInformation.REPOURL, buildInformation.BRANCH,
		pipeline, resources, params)

	logging.Log.Infof("Creating a new PipelineRun named %s in the namespace %s using the service account %s.", generatedPipelineRunName, pipelineNs, saName)

	pipelineRun, err := r.TektonClient.TektonV1alpha1().PipelineRuns(pipelineNs).Create(pipelineRunData)
	if err != nil {
		logging.Log.Errorf("error creating the PipelineRun: %s", err.Error())
		return
	}
	logging.Log.Debugf("PipelineRun created: %+v.", pipelineRun)
}

// This creates TaskRun for monitoring the main PipelineRun and reporting the result to the github
func createTaskRunFromWebhookData(buildInformation BuildInformation, r Resource) {
	logging.Log.Debugf("In createTaskRunFromWebhookData, build information: %s.", buildInformation)

	installNs := os.Getenv("INSTALLED_NAMESPACE")
	if installNs == "" {
		installNs = "default"
	}

	logging.Log.Debugf("Looking for the pipeline configmap in the install namespace %s.", installNs)

	// get information from related githubsource instance
	webhook, err := r.getGitHubWebhook(buildInformation.REPOURL, installNs)
	if err != nil {
		logging.Log.Errorf("error getting github webhook: %s.", err.Error())
		return
	}

	taskTemplateName := webhook.PullTask
	taskNs := webhook.Namespace
	saName := webhook.ServiceAccount
	accessTokenRef := webhook.AccessTokenRef
	OnSuccessComment := webhook.OnSuccessComment
	OnFailureComment := webhook.OnFailureComment

	// Assumes you've already applied the yml: so the task definition must exist upfront.
	startTime := buildInformation.TIMESTAMP
	generatedPipelineRunName := fmt.Sprintf("%s-%s", webhook.Name, startTime)
	generatedTaskRunName := generatedPipelineRunName

	if saName == "" {
		saName = "default"
	}

	if taskTemplateName == "" {
		taskTemplateName = "monitor-result-task"
	}

	if OnSuccessComment == "" {
		OnSuccessComment = "OK: " + generatedPipelineRunName
	}

	if OnFailureComment == "" {
		OnFailureComment = "ERROR: " + generatedPipelineRunName
	}

	logging.Log.Debugf("Build information: %+v.", buildInformation)

	// Unique names are required so timestamp them.
	pullRequestResourceName := fmt.Sprintf("%s-pull-request-%s", webhook.Name, startTime)

	task, err := r.getTaskImpl(taskTemplateName, taskNs)
	if err != nil {
		logging.Log.Errorf("could not find the task template %s in namespace %s.", taskTemplateName, taskNs)
		return
	}
	logging.Log.Debugf("Found the task template %s OK.", taskTemplateName)

	logging.Log.Debug("Creating PipelineResources.")

	paramsForPullRequestResource := []v1alpha1.Param{{Name: "url", Value: buildInformation.PULLURL}}
	secretParamsForPullRequestResource := []v1alpha1.SecretParam{{FieldName: "githubToken", SecretKey: "accessToken", SecretName: accessTokenRef}}
	pipelinePullRequestResource := definePipelineResource(pullRequestResourceName, taskNs, paramsForPullRequestResource, secretParamsForPullRequestResource, "pullRequest")
	createdPipelinePullRequestResource, err := r.TektonClient.TektonV1alpha1().PipelineResources(taskNs).Create(pipelinePullRequestResource)
	if err != nil {
		logging.Log.Errorf("error creating pipeline image resource to be used in the pipeline: %s.", err.Error())
		return
	}
	logging.Log.Infof("Created pipeline pull request resource %s successfully.", createdPipelinePullRequestResource.Name)

	pullRequestResourceRef := v1alpha1.PipelineResourceRef{Name: pullRequestResourceName}

	resources := []v1alpha1.TaskResourceBinding{{Name: "pull-request", ResourceRef: pullRequestResourceRef}}

	params := []v1alpha1.Param{{Name: "commentsuccess", Value: OnSuccessComment},
		{Name: "commentfailure", Value: OnFailureComment},
		{Name: "pipelinerun", Value: generatedPipelineRunName }}

	// TaskRun yml defines the references to the above named resources.
	taskRunData, err := defineTaskRun(generatedTaskRunName, taskNs, saName, buildInformation.REPOURL, buildInformation.BRANCH,
		task, resources, params)

	logging.Log.Infof("Creating a new TaskRun named %s in the namespace %s using the service account %s.", generatedPipelineRunName, taskNs, saName)

	taskRun, err := r.TektonClient.TektonV1alpha1().TaskRuns(taskNs).Create(taskRunData)
	if err != nil {
		logging.Log.Errorf("error creating the TaskRun: %s", err.Error())
		return
	}
	logging.Log.Debugf("TaskRun created: %+v.", taskRun)
}

/* Get all pipelines in a given namespace: the caller needs to handle any errors,
an empty v1alpha1.Pipeline{} is returned if no pipeline is found */
func (r Resource) getPipelineImpl(name, namespace string) (v1alpha1.Pipeline, error) {
	logging.Log.Infof("In getPipelineImpl, name %s, namespace %s.", name, namespace)

	pipelines := r.TektonClient.TektonV1alpha1().Pipelines(namespace)
	pipeline, err := pipelines.Get(name, metav1.GetOptions{})
	if err != nil {
		logging.Log.Errorf("error receiving the pipeline called %s in namespace %s: %s.", name, namespace, err.Error())
		return v1alpha1.Pipeline{}, err
	}
	logging.Log.Info("Found the pipeline definition OK.")
	return *pipeline, nil
}

/* Get all tasks in a given namespace: the caller needs to handle any errors,
an empty v1alpha1.Task{} is returned if no task is found */
func (r Resource) getTaskImpl(name, namespace string) (v1alpha1.Task, error) {
	logging.Log.Infof("In getTaskImpl, name %s, namespace %s.", name, namespace)

	tasks := r.TektonClient.TektonV1alpha1().Tasks(namespace)
	task, err := tasks.Get(name, metav1.GetOptions{})
	if err != nil {
		logging.Log.Errorf("error receiving the task called %s in namespace %s: %s.", name, namespace, err.Error())
		return v1alpha1.Task{}, err
	}
	logging.Log.Info("Found the task definition OK.")
	return *task, nil
}

/* Create a new PipelineResource: this should be of type git or image */
func definePipelineResource(name, namespace string, params []v1alpha1.Param, secrets []v1alpha1.SecretParam, resourceType v1alpha1.PipelineResourceType) *v1alpha1.PipelineResource {
	pipelineResource := v1alpha1.PipelineResource{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: v1alpha1.PipelineResourceSpec{
			Type:   resourceType,
			Params: params,
		},
	}
	if secrets != nil {
		pipelineResource.Spec.SecretParams = secrets
	}
	resourcePointer := &pipelineResource
	return resourcePointer
}
/* Create a new PipelineRun - repoUrl, resourceBinding and params can be nill depending on the Pipeline
each PipelineRun has a 1 hour timeout: */
func definePipelineRun(pipelineRunName, namespace, saName, repoURL, branch string,
	pipeline v1alpha1.Pipeline,
	resourceBinding []v1alpha1.PipelineResourceBinding,
	params []v1alpha1.Param) (*v1alpha1.PipelineRun, error) {

	gitServer, gitOrg, gitRepo := "", "", ""
	err := errors.New("")
	if repoURL != "" {
		gitServer, gitOrg, gitRepo, err = getGitValues(repoURL)
		if err != nil {
			logging.Log.Errorf("error getting the Git values: %s.", err)
			return &v1alpha1.PipelineRun{}, err
		}
	}

	pipelineRunData := v1alpha1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pipelineRunName,
			Namespace: namespace,
			Labels: map[string]string{
				"app":          "tekton-webhook-handler",
				gitServerLabel: gitServer,
				gitOrgLabel:    gitOrg,
				gitRepoLabel:   gitRepo,
				gitBranchLabel: branch,
			},
		},

		Spec: v1alpha1.PipelineRunSpec{
			PipelineRef: v1alpha1.PipelineRef{Name: pipeline.Name},
			ServiceAccount: saName,
			Timeout:        &metav1.Duration{Duration: 1 * time.Hour},
			Resources:      resourceBinding,
			Params:         params,
		},
	}

	pipelineRunPointer := &pipelineRunData
	return pipelineRunPointer, nil
}

/* Create a new TaskRun - repoUrl, resourceBinding and params can be nill depending on the Pipeline
each TaskRun has a 1 hour timeout: */
func defineTaskRun(taskRunName, namespace, saName, repoURL string, branch string,
	task v1alpha1.Task,
	resourceBinding []v1alpha1.TaskResourceBinding,
	params []v1alpha1.Param) (*v1alpha1.TaskRun, error) {

	gitServer, gitOrg, gitRepo := "", "", ""
	err := errors.New("")
	if repoURL != "" {
		gitServer, gitOrg, gitRepo, err = getGitValues(repoURL)
		if err != nil {
			logging.Log.Errorf("error getting the Git values: %s.", err)
			return &v1alpha1.TaskRun{}, err
		}
	}

	taskRunData := v1alpha1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      taskRunName,
			Namespace: namespace,
			Labels: map[string]string{
				"app":          "tekton-webhook-handler",
				gitServerLabel: gitServer,
				gitOrgLabel:    gitOrg,
				gitRepoLabel:   gitRepo,
				gitBranchLabel: branch,
			},
		},

		Spec: v1alpha1.TaskRunSpec{
			TaskRef:        &v1alpha1.TaskRef{Name: task.Name},
			Timeout:        &metav1.Duration{Duration: 1 * time.Hour},
			Inputs:  v1alpha1.TaskRunInputs{
				Resources:      resourceBinding,
				Params:         params,
			},
			Outputs:  v1alpha1.TaskRunOutputs{
				Resources:      resourceBinding,
			},
		},
	}
	taskRunPointer := &taskRunData
	return taskRunPointer, nil
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
		return "", "", "", errors.New("URL didn't match the requirement (at least two slashes)")
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
func (r Resource) SinkWebService(container *restful.Container) {
	ws := new(restful.WebService)
	ws.Path("/")
	ws.Route(ws.POST("").To(r.handleWebhook))

	container.Add(ws)
}
