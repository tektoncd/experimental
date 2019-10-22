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
	json "encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	restful "github.com/emicklei/go-restful"
	logging "github.com/tektoncd/experimental/webhooks-extension/pkg/logging"
	v1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	gh "gopkg.in/go-playground/webhooks.v3/github"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	gitServerLabel       = "gitServer"
	gitOrgLabel          = "gitOrg"
	gitRepoLabel         = "gitRepo"
	gitBranchLabel       = "gitBranch"
	githubEventParameter = "Ce-Github-Event"
)

// BuildInformation is information required to build a particular commit from a git repository.
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

	requestBodyBytes, err := ioutil.ReadAll(request.Request.Body)
	if err != nil {
		logging.Log.Errorf("Error reading request body: %s.", err.Error())
		return
	}
	requestHeaderBytes, err := json.Marshal(request.Request.Header)
	if err != nil {
		logging.Log.Errorf("Error reading request headers: %s.", err.Error())
		return
	}

	if gitHubEventTypeString == "ping" {
		response.WriteHeader(http.StatusNoContent)
	} else if gitHubEventTypeString == "push" {
		logging.Log.Info("Handling a push event.")

		webhookData := gh.PushPayload{}

		if err := json.Unmarshal(requestBodyBytes, &webhookData); err != nil {
			logging.Log.Errorf("Error decoding webhook data: %s.", err.Error())
			return
		}

		buildInformation.REPOURL = webhookData.Repository.URL
		buildInformation.SHORTID = webhookData.HeadCommit.ID[0:7]
		buildInformation.COMMITID = webhookData.HeadCommit.ID
		buildInformation.REPONAME = webhookData.Repository.Name
		buildInformation.TIMESTAMP = timestamp
		buildInformation.BRANCH = extractBranchFromPushEventRef(webhookData.Ref)

		r.createPipelineRunsFromWebhookData(buildInformation, requestBodyBytes, requestHeaderBytes)
		logging.Log.Debugf("Build information for repository %s:%s: %s.", buildInformation.REPOURL, buildInformation.SHORTID, buildInformation)

	} else if gitHubEventTypeString == "pull_request" {
		logging.Log.Info("Handling a pull request event.")

		webhookData := gh.PullRequestPayload{}

		if err := json.Unmarshal(requestBodyBytes, &webhookData); err != nil {
			logging.Log.Errorf("Error decoding webhook data: %s.", err.Error())
			return
		}

		buildInformation.REPOURL = webhookData.Repository.HTMLURL
		buildInformation.SHORTID = webhookData.PullRequest.Head.Sha[0:7]
		buildInformation.COMMITID = webhookData.PullRequest.Head.Sha
		buildInformation.REPONAME = webhookData.Repository.Name
		buildInformation.PULLURL = webhookData.PullRequest.HTMLURL
		buildInformation.TIMESTAMP = timestamp
		buildInformation.BRANCH = webhookData.PullRequest.Head.Ref

		pipelineruns := r.createPipelineRunsFromWebhookData(buildInformation, requestBodyBytes, requestHeaderBytes)
		logging.Log.Debugf("Build information for repository %s:%s: %s.", buildInformation.REPOURL, buildInformation.SHORTID, buildInformation)
		createTaskRunFromWebhookData(buildInformation, r, pipelineruns)
		logging.Log.Debugf("created monitoring task for pipelinerun from build information for repository %s commitId %s.", buildInformation.REPOURL,
			buildInformation.SHORTID)
	} else {
		logging.Log.Errorf("error: event wasn't a push, pull, or ping event, no action will be taken. Request is: %+v.", request)
	}
}

func extractBranchFromPushEventRef(ref string) string {
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
func (r Resource) createPipelineRunsFromWebhookData(buildInformation BuildInformation, eventPayload, eventHeaders []byte) (result []*v1alpha1.PipelineRun) {
	logging.Log.Debugf("In createPipelineRunFromWebhookData, build information: %s.", buildInformation)

	// TODO: Use the dashboard endpoint to create the PipelineRun
	// Track PR: https://github.com/tektoncd/dashboard/pull/33
	// and issue: https://github.com/tektoncd/dashboard/issues/47

	// Install namespace
	installNs := os.Getenv("INSTALLED_NAMESPACE")
	if installNs == "" {
		installNs = "tekton-pipelines"
	}

	logging.Log.Debugf("Looking for the pipeline configmap in the install namespace %s.", installNs)

	// get information from related githubsource instance
	webhooks, err := r.getGitHubWebhooks(buildInformation.REPOURL)
	if err != nil {
		logging.Log.Errorf("error getting github webhook: %s.", err.Error())
		return nil
	}

	var pipelineRuns []*v1alpha1.PipelineRun
	for _, hook := range webhooks {
		dockerRegistry := hook.DockerRegistry
		helmSecret := hook.HelmSecret
		pipelineTemplateName := hook.Pipeline
		pipelineNs := hook.Namespace
		saName := hook.ServiceAccount
		requestedReleaseName := hook.ReleaseName

		// Assumes you've already applied the yml: so the pipeline definition and its tasks must exist upfront.
		startTime := buildInformation.TIMESTAMP
		pipelineRunNamePrefix := fmt.Sprintf("%s-%s", hook.Name, startTime)
		if saName == "" {
			saName = "default"
		}

		logging.Log.Debugf("Build information: %+v.", buildInformation)

		// Unique names are required so timestamp them.
		imageResourceName := fmt.Sprintf("%s-docker-image-", hook.Name)
		gitResourceName := fmt.Sprintf("%s-git-source-", hook.Name)

		pipeline, err := r.getPipelineImpl(pipelineTemplateName, pipelineNs)
		if err != nil {
			logging.Log.Errorf("could not find the pipeline template %s in namespace %s.", pipelineTemplateName, pipelineNs)
			return nil
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
			return nil
		}
		logging.Log.Infof("Created pipeline image resource %s successfully.", createdPipelineImageResource.Name)

		paramsForGitResource := []v1alpha1.Param{{Name: "revision", Value: buildInformation.COMMITID}, {Name: "url", Value: buildInformation.REPOURL}}
		pipelineGitResource := definePipelineResource(gitResourceName, pipelineNs, paramsForGitResource, nil, "git")
		createdPipelineGitResource, err := r.TektonClient.TektonV1alpha1().PipelineResources(pipelineNs).Create(pipelineGitResource)

		if err != nil {
			logging.Log.Errorf("error creating pipeline git resource to be used in the pipeline: %s.", err.Error())
			return nil
		}
		logging.Log.Infof("Created pipeline git resource %s successfully.", createdPipelineGitResource.Name)

		gitResourceRef := v1alpha1.PipelineResourceRef{Name: createdPipelineGitResource.Name}
		imageResourceRef := v1alpha1.PipelineResourceRef{Name: createdPipelineImageResource.Name}

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

		params := []v1alpha1.Param{
			{Name: "image-tag", Value: imageTag},
			{Name: "image-name", Value: imageName},
			{Name: "release-name", Value: releaseName},
			{Name: "repository-name", Value: repositoryName},
			{Name: "target-namespace", Value: pipelineNs},
			{Name: "event-payload", Value: string(eventPayload)},
			{Name: "event-headers", Value: string(eventHeaders)},
			{Name: "branch", Value: buildInformation.BRANCH},
		}

		if dockerRegistry != "" {
			params = append(params, v1alpha1.Param{Name: "docker-registry", Value: dockerRegistry})
		}

		if helmSecret != "" {
			params = append(params, v1alpha1.Param{Name: "helm-secret", Value: helmSecret})
		}

		// PipelineRun yml defines the references to the above named resources.
		pipelineRunData, err := definePipelineRun(pipelineRunNamePrefix, pipelineNs, saName, buildInformation.REPOURL, buildInformation.BRANCH,
			pipeline, resources, params)

		logging.Log.Infof("Creating a new PipelineRun named %s in the namespace %s using the service account %s.", pipelineRunNamePrefix, pipelineNs, saName)

		pipelineRun, err := r.TektonClient.TektonV1alpha1().PipelineRuns(pipelineNs).Create(pipelineRunData)
		if err != nil {
			logging.Log.Errorf("error creating the PipelineRun: %s", err.Error())
			return nil
		}
		logging.Log.Debugf("PipelineRun created: %+v.", pipelineRun)
		pipelineRuns = append(pipelineRuns, pipelineRun)
	}
	return pipelineRuns
}

func getDashboardURL(r Resource, installNs string) string {
	type element struct {
		Type string `json:"type"`
		URL  string `json:"url"`
	}

	toReturn := "http://localhost:9097/"

	labelLookup := "app=tekton-dashboard"
	if "openshift" == os.Getenv("PLATFORM") {
		labelLookup = "app=tekton-dashboard-internal"
	}

	services, err := r.K8sClient.CoreV1().Services(installNs).List(metav1.ListOptions{LabelSelector: labelLookup})
	if err != nil {
		logging.Log.Errorf("could not find the dashboard's service: %s", err.Error())
		return toReturn
	}
	name := services.Items[0].GetName()
	proto := services.Items[0].Spec.Ports[0].Name
	port := services.Items[0].Spec.Ports[0].Port
	url := fmt.Sprintf("%s://%s:%d/v1/namespaces/%s/endpoints", proto, name, port, installNs)
	logging.Log.Debugf("using url: %s", url)
	resp, err := http.DefaultClient.Get(url)
	if err != nil {
		logging.Log.Errorf("error occurred when hitting the endpoints REST endpoint: %s", err.Error())
		return toReturn
	}
	if resp.StatusCode != 200 {
		logging.Log.Errorf("return code was not 200 when hitting the endpoints REST endpoint, code returned was: %d", resp.StatusCode)
		return toReturn
	}

	bodyJSON := []element{}
	json.NewDecoder(resp.Body).Decode(&bodyJSON)
	return bodyJSON[0].URL
}

// This creates TaskRun for monitoring the main PipelineRun and reporting the result to the github
func createTaskRunFromWebhookData(buildInformation BuildInformation, r Resource, pipelineruns []*v1alpha1.PipelineRun) {
	logging.Log.Debugf("In createTaskRunFromWebhookData, build information: %s.", buildInformation)

	installNs := os.Getenv("INSTALLED_NAMESPACE")
	if installNs == "" {
		installNs = "tekton-pipelines"
	}

	logging.Log.Debugf("Looking for the pipeline configmap in the install namespace %s.", installNs)

	// get information from related githubsource instance
	webhooksForRepo, err := r.getGitHubWebhooks(buildInformation.REPOURL)
	if err != nil {
		logging.Log.Errorf("error getting github webhook: %s.", err.Error())
		return
	}

	// all webhooks for a repo must use the same pull request update task so just look it props on the first one
	taskTemplateName := webhooksForRepo[0].PullTask
	onSuccessComment := webhooksForRepo[0].OnSuccessComment
	onFailureComment := webhooksForRepo[0].OnFailureComment
	accessTokenRef := webhooksForRepo[0].AccessTokenRef

	// Assumes you've already applied the yml: so the task definition must exist upfront.
	startTime := buildInformation.TIMESTAMP
	generatedTaskRunNamePrefix := fmt.Sprintf("%s-%s", "pr-monitor", startTime)

	var pipelineRunsDetailsArray []string
	for _, pr := range pipelineruns {
		pipelineRunsDetailsArray = append(pipelineRunsDetailsArray, pr.GetName()+":"+pr.GetNamespace()+":"+pr.Spec.PipelineRef.Name)
	}
	taskPipelineRunsParam := strings.Join(pipelineRunsDetailsArray, ",")

	saName := os.Getenv("SERVICEACCOUNT")
	if saName == "" {
		saName = "default"
	}

	if taskTemplateName == "" {
		taskTemplateName = "monitor-result-task"
	}

	if onSuccessComment == "" {
		onSuccessComment = "Success"
	}

	if onFailureComment == "" {
		onFailureComment = "Failed"
	}

	logging.Log.Debugf("Build information: %+v.", buildInformation)

	// Unique names are required so timestamp them.
	pullRequestResourceNamePrefix := fmt.Sprintf("pull-request-")

	task, err := r.getTaskImpl(taskTemplateName, installNs)
	if err != nil {
		logging.Log.Errorf("could not find the task template %s in namespace %s.", taskTemplateName, installNs)
		return
	}
	logging.Log.Debugf("Found the task template %s OK.", taskTemplateName)

	logging.Log.Debug("Creating PipelineResource.")

	paramsForPullRequestResource := []v1alpha1.Param{{Name: "url", Value: buildInformation.PULLURL}}
	secretParamsForPullRequestResource := []v1alpha1.SecretParam{{FieldName: "githubToken", SecretKey: "accessToken", SecretName: accessTokenRef}}
	pipelinePullRequestResource := definePipelineResource(pullRequestResourceNamePrefix, installNs, paramsForPullRequestResource, secretParamsForPullRequestResource, "pullRequest")
	createdPipelinePullRequestResource, err := r.TektonClient.TektonV1alpha1().PipelineResources(installNs).Create(pipelinePullRequestResource)
	if err != nil {
		logging.Log.Errorf("error creating pipeline pull request resource: %s.", err.Error())
		return
	}
	logging.Log.Infof("Created pipeline pull request resource %s successfully.", createdPipelinePullRequestResource.Name)

	pullRequestResourceRef := v1alpha1.PipelineResourceRef{Name: createdPipelinePullRequestResource.Name}

	resources := []v1alpha1.TaskResourceBinding{{Name: "pull-request", ResourceRef: pullRequestResourceRef}}

	params := []v1alpha1.Param{{Name: "commentsuccess", Value: onSuccessComment},
		{Name: "commentfailure", Value: onFailureComment},
		{Name: "pipelineruns", Value: taskPipelineRunsParam},
		{Name: "dashboard-url", Value: getDashboardURL(r, installNs)},
		{Name: "secret", Value: accessTokenRef}}
	// The "secret" in the params could optionally be deleted after fixing the pull request pending status
	// issue - we could argue for keeping it if we thought other people want custom tasks that have access to it

	// TaskRun yml defines the references to the above named resources.
	taskRunData, err := defineTaskRun(generatedTaskRunNamePrefix, installNs, saName, buildInformation.REPOURL, buildInformation.BRANCH,
		task, pipelineruns, resources, params)

	logging.Log.Infof("Creating a new TaskRun in the namespace %s using the service account %s.", installNs, saName)

	taskRun, err := r.TektonClient.TektonV1alpha1().TaskRuns(installNs).Create(taskRunData)
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
func definePipelineResource(namePrefix, namespace string, params []v1alpha1.Param, secrets []v1alpha1.SecretParam, resourceType v1alpha1.PipelineResourceType) *v1alpha1.PipelineResource {
	pipelineResource := v1alpha1.PipelineResource{
		ObjectMeta: metav1.ObjectMeta{GenerateName: namePrefix, Namespace: namespace},
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
func definePipelineRun(pipelineRunNamePrefix, namespace, saName, repoURL, branch string,
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
			GenerateName: pipelineRunNamePrefix + "-",
			Namespace:    namespace,
			Labels: map[string]string{
				"app":          "tekton-webhook-handler",
				gitServerLabel: gitServer,
				gitOrgLabel:    gitOrg,
				gitRepoLabel:   gitRepo,
				gitBranchLabel: branch,
			},
		},

		Spec: v1alpha1.PipelineRunSpec{
			PipelineRef:    v1alpha1.PipelineRef{Name: pipeline.Name},
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
func defineTaskRun(taskRunNamePrefix, namespace, saName, repoURL string, branch string,
	task v1alpha1.Task,
	pipelineRunRefs []*v1alpha1.PipelineRun,
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
			GenerateName: taskRunNamePrefix,
			Namespace:    namespace,
			Labels: map[string]string{
				"app":          "tekton-webhook-handler",
				gitServerLabel: gitServer,
				gitOrgLabel:    gitOrg,
				gitRepoLabel:   gitRepo,
				gitBranchLabel: branch,
			},
			//  The following lines cause an issue with the persistent volume claim in the Docker for Desktop
			//  The task created in the installed namespace must be garbage collected until the issue is fixed.
			//  This issue is related to https://github.com/tektoncd/pipeline/issues/1076
			//
			//  Note this would also need reworking as we now have an array of pipelineruns which need to need
			//  adding as children
			//
			//			OwnerReferences: []metav1.OwnerReference{
			//				{
			//					APIVersion: "tekton.dev/v1alpha1",
			//					Kind: "PipelineRun",
			//					Name: pipelineRunRef.GetName(),
			//					UID:  pipelineRunRef.GetUID(),
			//				},
			//			},
		},

		Spec: v1alpha1.TaskRunSpec{
			TaskRef:        &v1alpha1.TaskRef{Name: task.Name},
			ServiceAccount: saName,
			Timeout:        &metav1.Duration{Duration: 1 * time.Hour},
			Inputs: v1alpha1.TaskRunInputs{
				Resources: resourceBinding,
				Params:    params,
			},
			Outputs: v1alpha1.TaskRunOutputs{
				Resources: resourceBinding,
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
