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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	restful "github.com/emicklei/go-restful"
	logging "github.com/tektoncd/experimental/webhooks-extension/pkg/logging"
	pipelinesv1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	v1alpha1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	"golang.org/x/oauth2"
	"golang.org/x/xerrors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var (
	modifyingEventListenerLock sync.Mutex
	actions                    = pipelinesv1alpha1.Param{Name: "Wext-Incoming-Actions", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: "opened,reopened,synchronize"}}
)

const eventListenerName = "tekton-webhooks-eventlistener"

/*
	Creation of the eventlistener, called when no eventlistener exists at
	the point of webhook creation.
*/
func (r Resource) createEventListener(webhook webhook, namespace, monitorTriggerName string) (*v1alpha1.EventListener, error) {
	hookParams, monitorParams := r.getParams(webhook)

	pushTrigger := r.newTrigger(webhook.Name+"-"+webhook.Namespace+"-push-event",
		webhook.Pipeline+"-push-binding",
		webhook.Pipeline+"-template",
		webhook.GitRepositoryURL,
		"push",
		webhook.AccessTokenRef,
		hookParams)

	pullRequestTrigger := r.newTrigger(webhook.Name+"-"+webhook.Namespace+"-pullrequest-event",
		webhook.Pipeline+"-pullrequest-binding",
		webhook.Pipeline+"-template",
		webhook.GitRepositoryURL,
		"pull_request",
		webhook.AccessTokenRef,
		hookParams)
	pullRequestTrigger.Interceptor.Header = append(pullRequestTrigger.Interceptor.Header, actions)

	monitorTrigger := r.newTrigger(monitorTriggerName,
		webhook.PullTask+"-binding",
		webhook.PullTask+"-template",
		webhook.GitRepositoryURL,
		"pull_request",
		webhook.AccessTokenRef,
		monitorParams)
	monitorTrigger.Interceptor.Header = append(monitorTrigger.Interceptor.Header, actions)

	triggers := []v1alpha1.EventListenerTrigger{pushTrigger, pullRequestTrigger, monitorTrigger}

	eventListener := v1alpha1.EventListener{
		ObjectMeta: metav1.ObjectMeta{
			Name:      eventListenerName,
			Namespace: namespace,
		},
		Spec: v1alpha1.EventListenerSpec{
			ServiceAccountName: "tekton-webhooks-extension-eventlistener",
			Triggers:           triggers,
		},
	}
	return r.TriggersClient.TektonV1alpha1().EventListeners(namespace).Create(&eventListener)
}

/*
	Update of the eventlistener, called when adding additional webhooks as we
	run with a single eventlistener.
*/
func (r Resource) updateEventListener(eventListener *v1alpha1.EventListener, webhook webhook, monitorTriggerName string) (*v1alpha1.EventListener, error) {
	hookParams, monitorParams := r.getParams(webhook)
	newPushTrigger := r.newTrigger(webhook.Name+"-"+webhook.Namespace+"-push-event",
		webhook.Pipeline+"-push-binding",
		webhook.Pipeline+"-template",
		webhook.GitRepositoryURL,
		"push",
		webhook.AccessTokenRef,
		hookParams)

	newPullRequestTrigger := r.newTrigger(webhook.Name+"-"+webhook.Namespace+"-pullrequest-event",
		webhook.Pipeline+"-pullrequest-binding",
		webhook.Pipeline+"-template",
		webhook.GitRepositoryURL,
		"pull_request",
		webhook.AccessTokenRef,
		hookParams)
	newPullRequestTrigger.Interceptor.Header = append(newPullRequestTrigger.Interceptor.Header, actions)

	eventListener.Spec.Triggers = append(eventListener.Spec.Triggers, newPushTrigger)
	eventListener.Spec.Triggers = append(eventListener.Spec.Triggers, newPullRequestTrigger)

	existingMonitorFound := false
	for _, trigger := range eventListener.Spec.Triggers {
		if trigger.Name == monitorTriggerName {
			existingMonitorFound = true
			break
		}
	}
	if !existingMonitorFound {
		newMonitor := r.newTrigger(monitorTriggerName,
			webhook.PullTask+"-binding",
			webhook.PullTask+"-template",
			webhook.GitRepositoryURL,
			"pull_request",
			webhook.AccessTokenRef,
			monitorParams)
		newMonitor.Interceptor.Header = append(newMonitor.Interceptor.Header, actions)

		eventListener.Spec.Triggers = append(eventListener.Spec.Triggers, newMonitor)
	}

	return r.TriggersClient.TektonV1alpha1().EventListeners(eventListener.GetNamespace()).Update(eventListener)
}

func (r Resource) newTrigger(name, bindingName, templateName, repoURL, event, secretName string, params []pipelinesv1alpha1.Param) v1alpha1.EventListenerTrigger {
	return v1alpha1.EventListenerTrigger{
		Name: name,
		Binding: v1alpha1.EventListenerBinding{
			Name:       bindingName,
			APIVersion: "v1alpha1",
		},
		Params: params,
		Template: v1alpha1.EventListenerTemplate{
			Name:       templateName,
			APIVersion: "v1alpha1",
		},
		Interceptor: &v1alpha1.EventInterceptor{
			Header: []pipelinesv1alpha1.Param{
				{Name: "Wext-Trigger-Name", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: name}},
				{Name: "Wext-Repository-Url", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: repoURL}},
				{Name: "Wext-Incoming-Event", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: event}},
				{Name: "Wext-Secret-Name", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: secretName}}},
			ObjectRef: &corev1.ObjectReference{
				APIVersion: "v1",
				Kind:       "Service",
				Name:       "tekton-webhooks-extension-validator",
				Namespace:  r.Defaults.Namespace,
			},
		},
	}
}

/*
	Processing of the inputs into the required structure for
	the eventlistener.
*/
func (r Resource) getParams(webhook webhook) (webhookParams, monitorParams []pipelinesv1alpha1.Param) {
	saName := webhook.ServiceAccount
	requestedReleaseName := webhook.ReleaseName
	if saName == "" {
		saName = "default"
	}
	server, org, repo, err := getGitValues(webhook.GitRepositoryURL)
	if err != nil {
		logging.Log.Errorf("error returned from getGitValues: %s", err)
	}
	server = strings.TrimPrefix(server, "https://")
	server = strings.TrimPrefix(server, "http://")

	releaseName := ""
	if requestedReleaseName != "" {
		logging.Log.Infof("Release name based on input: %s", requestedReleaseName)
		releaseName = requestedReleaseName
	} else {
		releaseName = repo
		logging.Log.Infof("Release name based on repository name: %s", releaseName)
	}

	hookParams := []pipelinesv1alpha1.Param{
		{Name: "webhooks-tekton-release-name", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: releaseName}},
		{Name: "webhooks-tekton-target-namespace", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: webhook.Namespace}},
		{Name: "webhooks-tekton-service-account", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: webhook.ServiceAccount}},
		{Name: "webhooks-tekton-git-server", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: server}},
		{Name: "webhooks-tekton-git-org", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: org}},
		{Name: "webhooks-tekton-git-repo", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: repo}},
		{Name: "webhooks-tekton-pull-task", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: webhook.PullTask}}}

	if webhook.DockerRegistry != "" {
		hookParams = append(hookParams, pipelinesv1alpha1.Param{Name: "webhooks-tekton-docker-registry", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: webhook.DockerRegistry}})
	}
	if webhook.HelmSecret != "" {
		hookParams = append(hookParams, pipelinesv1alpha1.Param{Name: "webhooks-tekton-helm-secret", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: webhook.HelmSecret}})
	}

	onSuccessComment := webhook.OnSuccessComment
	if onSuccessComment == "" {
		onSuccessComment = "Success"
	}
	onFailureComment := webhook.OnFailureComment
	if onFailureComment == "" {
		onFailureComment = "Failed"
	}
	onTimeoutComment := webhook.OnTimeoutComment
	if onTimeoutComment == "" {
		onTimeoutComment = "Unknown"
	}

	prMonitorParams := []pipelinesv1alpha1.Param{
		{Name: "commentsuccess", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: onSuccessComment}},
		{Name: "commentfailure", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: onFailureComment}},
		{Name: "commenttimeout", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: onTimeoutComment}},
		{Name: "gitsecretname", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: webhook.AccessTokenRef}},
		{Name: "gitsecretkeyname", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: "accessToken"}},
		{Name: "dashboardurl", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: r.getDashboardURL(r.Defaults.Namespace)}},
	}

	return hookParams, prMonitorParams
}

func (r Resource) getDashboardURL(installNs string) string {
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
		logging.Log.Errorf("could not find the dashboard's service - error: %s", err.Error())
		return toReturn
	}

	if len(services.Items) == 0 {
		logging.Log.Error("could not find the dashboard's service")
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
		return url
	}
	if resp.StatusCode != 200 {
		logging.Log.Errorf("return code was not 200 when hitting the endpoints REST endpoint, code returned was: %d", resp.StatusCode)
		return url
	}

	bodyJSON := []element{}
	json.NewDecoder(resp.Body).Decode(&bodyJSON)
	return bodyJSON[0].URL
}

/*
	Processes a git URL into component parts, all of which are lowercased
	to try and avoid problems matching strings.
*/
func getGitValues(url string) (gitServer, gitOwner, gitRepo string, err error) {
	repoURL := ""
	prefix := ""
	if url != "" {
		url = strings.ToLower(url)
		if strings.Contains(url, "https://") {
			repoURL = strings.TrimPrefix(url, "https://")
			prefix = "https://"
		} else {
			repoURL = strings.TrimPrefix(url, "http://")
			prefix = "http://"
		}
	}
	// example at this point: github.com/tektoncd/pipeline
	numSlashes := strings.Count(repoURL, "/")
	if numSlashes < 2 {
		return "", "", "", errors.New("URL didn't contain an owner and repository")
	}
	repoURL = strings.TrimSuffix(repoURL, "/")
	gitServer = prefix + repoURL[0:strings.Index(repoURL, "/")]
	gitOwner = repoURL[strings.Index(repoURL, "/")+1 : strings.LastIndex(repoURL, "/")]
	//need to cut off the .git
	if strings.HasSuffix(url, ".git") {
		gitRepo = repoURL[strings.LastIndex(repoURL, "/")+1 : len(repoURL)-4]
	} else {
		gitRepo = repoURL[strings.LastIndex(repoURL, "/")+1:]
	}

	return gitServer, gitOwner, gitRepo, nil
}

// Creates a webhook for a given repository and populates (creating if doesn't yet exist) an eventlistener
func (r Resource) createWebhook(request *restful.Request, response *restful.Response) {
	modifyingEventListenerLock.Lock()
	defer modifyingEventListenerLock.Unlock()

	logging.Log.Infof("Webhook creation request received with request: %+v.", request)
	installNs := r.Defaults.Namespace

	webhook := webhook{}
	if err := request.ReadEntity(&webhook); err != nil {
		logging.Log.Errorf("error trying to read request entity as webhook: %s.", err)
		RespondError(response, err, http.StatusBadRequest)
		return
	}

	// Sanitize GitRepositoryURL
	webhook.GitRepositoryURL = strings.TrimSuffix(webhook.GitRepositoryURL, ".git")

	if webhook.PullTask == "" {
		webhook.PullTask = "monitor-task"
	}

	if webhook.Name != "" {
		if len(webhook.Name) > 57 {
			tooLongMessage := fmt.Sprintf("requested release name (%s) must be less than 58 characters", webhook.Name)
			err := errors.New(tooLongMessage)
			logging.Log.Errorf("error: %s", err.Error())
			RespondError(response, err, http.StatusBadRequest)
			return
		}
	}

	dockerRegDefault := r.Defaults.DockerRegistry
	// remove prefixes if any
	webhook.DockerRegistry = strings.TrimPrefix(webhook.DockerRegistry, "https://")
	webhook.DockerRegistry = strings.TrimPrefix(webhook.DockerRegistry, "http://")
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

	if !strings.HasPrefix(webhook.GitRepositoryURL, "http") {
		err := errors.New("the supplied GitRepositoryURL does not specify the protocol http:// or https://")
		logging.Log.Errorf("error: %s", err.Error())
		RespondError(response, err, http.StatusBadRequest)
		return
	}

	pieces := strings.Split(webhook.GitRepositoryURL, "/")
	if len(pieces) < 4 {
		logging.Log.Errorf("error creating webhook: GitRepositoryURL format error (%+v).", webhook.GitRepositoryURL)
		RespondError(response, errors.New("GitRepositoryURL format error"), http.StatusBadRequest)
		return
	}

	hooks, err := r.getHooksForRepo(webhook.GitRepositoryURL)
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
	}

	_, templateErr := r.TriggersClient.TektonV1alpha1().TriggerTemplates(installNs).Get(webhook.Pipeline+"-template", metav1.GetOptions{})
	_, pushErr := r.TriggersClient.TektonV1alpha1().TriggerBindings(installNs).Get(webhook.Pipeline+"-push-binding", metav1.GetOptions{})
	_, pullrequestErr := r.TriggersClient.TektonV1alpha1().TriggerBindings(installNs).Get(webhook.Pipeline+"-pullrequest-binding", metav1.GetOptions{})
	if templateErr != nil || pushErr != nil || pullrequestErr != nil {
		msg := fmt.Sprintf("Could not find the required trigger template or trigger bindings in namespace: %s. Expected to find: %s, %s and %s", installNs, webhook.Pipeline+"-template", webhook.Pipeline+"-push-binding", webhook.Pipeline+"-pullrequest-binding")
		logging.Log.Errorf("%s", msg)
		RespondError(response, errors.New(msg), http.StatusBadRequest)
		return
	}

	eventListener, err := r.TriggersClient.TektonV1alpha1().EventListeners(installNs).Get(eventListenerName, metav1.GetOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		msg := fmt.Sprintf("unable to create webhook due to error listing Tekton eventlistener: %s", err)
		logging.Log.Errorf("%s", msg)
		RespondError(response, errors.New(msg), http.StatusInternalServerError)
		return
	}

	gitServer, gitOwner, gitRepo, err := getGitValues(webhook.GitRepositoryURL)
	if err != nil {
		logging.Log.Errorf("error parsing git repository URL %s in getGitValues(): %s", webhook.GitRepositoryURL, err)
		RespondError(response, errors.New("error parsing GitRepositoryURL, check pod logs for more details"), http.StatusInternalServerError)
		return
	}
	sanitisedURL := gitServer + "/" + gitOwner + "/" + gitRepo
	// Single monitor trigger for all triggers on a repo - thus name to use for monitor is
	monitorTriggerName := strings.TrimPrefix(gitServer+"/"+gitOwner+"/"+gitRepo, "http://")
	monitorTriggerName = strings.TrimPrefix(monitorTriggerName, "https://")

	if eventListener != nil && eventListener.GetName() != "" {
		_, err := r.updateEventListener(eventListener, webhook, monitorTriggerName)
		if err != nil {
			msg := fmt.Sprintf("error creating webhook due to error updating eventlistener: %s", err)
			logging.Log.Errorf("%s", msg)
			RespondError(response, errors.New(msg), http.StatusInternalServerError)
			return
		}
	} else {
		logging.Log.Info("No existing eventlistener found, creating a new one...")
		_, err := r.createEventListener(webhook, installNs, monitorTriggerName)
		if err != nil {
			msg := fmt.Sprintf("error creating webhook due to error creating eventlistener. Error was: %s", err)
			logging.Log.Errorf("%s", msg)
			RespondError(response, errors.New(msg), http.StatusInternalServerError)
			return
		}
		_, varexists := os.LookupEnv("PLATFORM")
		if !varexists {
			err = r.createDeleteIngress("create", installNs)
			if err != nil {
				msg := fmt.Sprintf("error creating webhook due to error creating ingress. Error was: %s", err)
				logging.Log.Errorf("%s", msg)
				logging.Log.Debugf("Deleting eventlistener as failed creating Ingress")
				err2 := r.TriggersClient.TektonV1alpha1().EventListeners(installNs).Delete(eventListenerName, &metav1.DeleteOptions{})
				if err2 != nil {
					updatedMsg := fmt.Sprintf("error creating webhook due to error creating taskrun to create ingress. Also failed to cleanup and delete eventlistener. Errors were: %s and %s", err, err2)
					RespondError(response, errors.New(updatedMsg), http.StatusInternalServerError)
					return
				}
				RespondError(response, errors.New(msg), http.StatusInternalServerError)
				return
			} else {
				logging.Log.Debug("ingress creation taskrun succeeded")
			}
		} else {
			routeTaskRun, err := r.createRouteTaskRun("create", installNs)
			if err != nil {
				msg := fmt.Sprintf("error creating webhook due to error creating taskrun to create route. Error was: %s", err)
				logging.Log.Errorf("%s", msg)
				err2 := r.TriggersClient.TektonV1alpha1().EventListeners(installNs).Delete(eventListenerName, &metav1.DeleteOptions{})
				if err2 != nil {
					updatedMsg := fmt.Sprintf("error creating webhook due to error creating taskrun to create routes. Also failed to cleanup and delete eventlistener. Errors were: %s and %s", err, err2)
					RespondError(response, errors.New(updatedMsg), http.StatusInternalServerError)
					return
				}
				RespondError(response, errors.New(msg), http.StatusInternalServerError)
				return
			}

			routeTaskRunResult, err := r.checkTaskRunSucceeds(routeTaskRun, installNs)
			if !routeTaskRunResult && err != nil {
				msg := fmt.Sprintf("error creating webhook due to error in taskrun to create route. Error was: %s", err)
				logging.Log.Errorf("%s", msg)
				RespondError(response, errors.New(msg), http.StatusInternalServerError)
				return
			} else {
				logging.Log.Debug("route creation taskrun succeeded")
			}
		}

	}

	if len(hooks) == 0 {
		// Create webhook
		err = r.doGitHubWebhookRequest(webhook, "subscribe", []string{"push", "pull_request"})
		if err != nil {
			// Handle cleanup if it fails
			// - remove from EventListener
			// - delete EventListener?
			err2 := r.deleteFromEventListener(webhook.Name+"-"+webhook.Namespace, installNs, monitorTriggerName, webhook.GitRepositoryURL)
			if err2 != nil {
				updatedMsg := fmt.Sprintf("error creating webhook. Also failed to cleanup and delete entry from eventlistener. Errors were: %s and %s", err, err2)
				RespondError(response, errors.New(updatedMsg), http.StatusInternalServerError)
				return
			}
			RespondError(response, err, http.StatusInternalServerError)
			return
		}
		logging.Log.Debug("webhook creation succeeded")
	} else {
		logging.Log.Debugf("webhook already exists for repository %s - not creating new hook in GitHub", sanitisedURL)
	}

	response.WriteHeader(http.StatusCreated)
}

func (r Resource) createDeleteIngress(mode, installNS string) error {
	if mode == "create" {
		// Unlike webhook creation, the ingress does not need a protocol specified
		callback := strings.TrimPrefix(r.Defaults.CallbackURL, "http://")
		callback = strings.TrimPrefix(callback, "https://")

		ingress := &v1beta1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "el-" + eventListenerName,
				Namespace: installNS,
			},
			Spec: v1beta1.IngressSpec{
				Rules: []v1beta1.IngressRule{
					{
						Host: callback,
						IngressRuleValue: v1beta1.IngressRuleValue{
							HTTP: &v1beta1.HTTPIngressRuleValue{
								Paths: []v1beta1.HTTPIngressPath{
									{
										Backend: v1beta1.IngressBackend{
											ServiceName: "el-" + eventListenerName,
											ServicePort: intstr.IntOrString{
												Type:   intstr.Int,
												IntVal: 8080,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}
		ingress, err := r.K8sClient.ExtensionsV1beta1().Ingresses(installNS).Create(ingress)
		if err != nil {
			return err
		}
		logging.Log.Debug("Ingress has been created")
		return nil
	} else if mode == "delete" {
		err := r.K8sClient.ExtensionsV1beta1().Ingresses(installNS).Delete("el-"+eventListenerName, &metav1.DeleteOptions{})
		if err != nil {
			return err
		}
		logging.Log.Debug("Ingress has been deleted")
		return nil
	} else {
		logging.Log.Debug("Wrong mode")
		return errors.New("Wrong mode for createDeleteIngress")
	}
}

func (r Resource) createRouteTaskRun(mode, installNS string) (*pipelinesv1alpha1.TaskRun, error) {

	params := []pipelinesv1alpha1.Param{
		{Name: "Mode", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: mode}},
		{Name: "EventListenerServiceName", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: "el-" + eventListenerName}}}

	routeTaskRun := pipelinesv1alpha1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: mode + "-route-",
			Namespace:    installNS,
		},
		Spec: pipelinesv1alpha1.TaskRunSpec{
			Inputs: pipelinesv1alpha1.TaskRunInputs{
				Params: params,
			},
			ServiceAccount: os.Getenv("SERVICE_ACCOUNT"),
			TaskRef: &pipelinesv1alpha1.TaskRef{
				Name: "route-task",
			},
		},
	}

	tr, err := r.TektonClient.TektonV1alpha1().TaskRuns(installNS).Create(&routeTaskRun)
	if err != nil {
		return &pipelinesv1alpha1.TaskRun{}, err
	}
	logging.Log.Debugf("Route being created under taskrun %s", tr.GetName())

	return tr, nil
}

func (r Resource) checkTaskRunSucceeds(originalTaskRun *pipelinesv1alpha1.TaskRun, installNS string) (bool, error) {
	var err error
	retries := 1
	for retries < 120 {
		taskRun, err := r.TektonClient.TektonV1alpha1().TaskRuns(installNS).Get(originalTaskRun.Name, metav1.GetOptions{})
		if err != nil {
			logging.Log.Debugf("Error occured retrieving taskrun %s.", originalTaskRun.Name)
			return false, err
		}
		if taskRun.IsDone() {
			if taskRun.IsSuccessful() {
				return true, nil
			}
			if taskRun.IsCancelled() {
				err = errors.New("taskrun " + taskRun.Name + " is in a cancelled state")
				return false, err
			}
			err = errors.New("taskrun " + taskRun.Name + " is in a failed or unknown state")
			return false, err
		}
		time.Sleep(1 * time.Second)
		retries = retries + 1
	}

	err = errors.New("taskrun " + originalTaskRun.Name + " is not reporting as successful or cancelled")
	return false, err
}

// Removes from Eventlistener, removes the webhook
func (r Resource) deleteWebhook(request *restful.Request, response *restful.Response) {
	modifyingEventListenerLock.Lock()
	defer modifyingEventListenerLock.Unlock()
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

	webhooks, err := r.getHooksForRepo(repo)
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

	gitServer, gitOwner, gitRepo, err := getGitValues(repo)
	// Single monitor trigger for all triggers on a repo - thus name to use for monitor is
	monitorTriggerName := strings.TrimPrefix(gitServer+"/"+gitOwner+"/"+gitRepo, "http://")
	monitorTriggerName = strings.TrimPrefix(monitorTriggerName, "https://")

	found := false
	for _, hook := range webhooks {
		if hook.Name == name && hook.Namespace == namespace {
			found = true
			if len(webhooks) == 1 {
				logging.Log.Debug("No other pipelines triggered by this GitHub webhook, deleting webhook")
				// Delete webhook
				err := r.doGitHubWebhookRequest(hook, "unsubscribe", []string{"push", "pull_request"})
				if err != nil {
					RespondError(response, err, http.StatusInternalServerError)
					return
				}
				logging.Log.Debug("Webhook deletion succeeded")
			}
			if toDeletePipelineRuns {
				r.deletePipelineRuns(repo, namespace, hook.Pipeline)
			}
			eventListenerEntryPrefix := name + "-" + namespace
			err = r.deleteFromEventListener(eventListenerEntryPrefix, r.Defaults.Namespace, monitorTriggerName, repo)
			if err != nil {
				logging.Log.Error(err)
				theError := errors.New("error deleting webhook from eventlistener.")
				RespondError(response, theError, http.StatusInternalServerError)
				return
			}

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

func (r Resource) deleteFromEventListener(name, installNS, monitorTriggerName, repoOnParams string) error {
	logging.Log.Debugf("Deleting triggers for %s from the eventlistener", name)
	el, err := r.TriggersClient.TektonV1alpha1().EventListeners(installNS).Get(eventListenerName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	toRemove := []string{name + "-push-event", name + "-pullrequest-event"}

	newTriggers := []v1alpha1.EventListenerTrigger{}
	currentTriggers := el.Spec.Triggers

	monitorTrigger := v1alpha1.EventListenerTrigger{}
	triggersOnRepo := 0
	triggersDeleted := 0

	for _, t := range currentTriggers {
		if t.Name == monitorTriggerName {
			monitorTrigger = t
		} else {
			interceptorParams := t.Interceptor.Header
			for _, p := range interceptorParams {
				if p.Name == "Wext-Repository-Url" && p.Value.StringVal == repoOnParams {
					triggersOnRepo++
				}
			}
			found := false
			for _, triggerName := range toRemove {
				if triggerName == t.Name {
					triggersDeleted++
					found = true
					break
				}
			}
			if !found {
				newTriggers = append(newTriggers, t)
			}
		}
	}

	if triggersOnRepo > triggersDeleted {
		newTriggers = append(newTriggers, monitorTrigger)
	}

	if len(newTriggers) == 0 {
		err = r.TriggersClient.TektonV1alpha1().EventListeners(installNS).Delete(el.GetName(), &metav1.DeleteOptions{})
		if err != nil {
			return err
		}

		_, varExists := os.LookupEnv("PLATFORM")
		if !varExists {
			err = r.createDeleteIngress("delete", installNS)
			if err != nil {
				logging.Log.Errorf("error deleting ingress: %s", err)
				return err
			} else {
				logging.Log.Debug("Ingress deleted")
				return nil
			}
		} else {
			routeTaskRun, err := r.createRouteTaskRun("delete", installNS)
			if err != nil {
				msg := fmt.Sprintf("error deleting webhook due to error creating taskrun to delete route. Error was: %s", err)
				logging.Log.Errorf("%s", msg)
				return err
			}
			routeTaskRunResult, err := r.checkTaskRunSucceeds(routeTaskRun, installNS)
			if !routeTaskRunResult && err != nil {
				msg := fmt.Sprintf("error deleting webhook due to error in taskrun to delete route. Error was: %s", err)
				logging.Log.Errorf("%s", msg)
				return err
			} else {
				logging.Log.Debug("route deletion taskrun succeeded")
			}
		}

	} else {
		el.Spec.Triggers = newTriggers
		_, err = r.TriggersClient.TektonV1alpha1().EventListeners(installNS).Update(el)
		if err != nil {
			logging.Log.Errorf("error updating eventlistener: %s", err)
			return err
		}
	}

	return err
}

func (r Resource) getAllWebhooks(request *restful.Request, response *restful.Response) {
	logging.Log.Debugf("Get all webhooks")
	webhooks, err := r.getWebhooksFromEventListener()
	if err != nil {
		logging.Log.Errorf("error trying to get webhooks: %s.", err.Error())
		RespondError(response, err, http.StatusInternalServerError)
		return
	}
	response.WriteEntity(webhooks)
}

func (r Resource) getHooksForRepo(gitURL string) ([]webhook, error) {
	hooksForRepo := []webhook{}
	allHooks, err := r.getWebhooksFromEventListener()
	if err != nil {
		return nil, err
	}

	for _, hook := range allHooks {
		if hook.GitRepositoryURL == gitURL {
			hooksForRepo = append(hooksForRepo, hook)
		}
	}

	return hooksForRepo, nil
}

func (r Resource) getWebhooksFromEventListener() ([]webhook, error) {
	logging.Log.Debugf("Getting webhooks from eventlistener")
	el, err := r.TriggersClient.TektonV1alpha1().EventListeners(r.Defaults.Namespace).Get(eventListenerName, metav1.GetOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return []webhook{}, nil
		}
		return nil, err
	}
	hooks := []webhook{}
	var hook webhook
	for _, trigger := range el.Spec.Triggers {
		checkHook := false
		if strings.HasSuffix(trigger.Name, "-push-event") {
			hook = getHookFromTrigger(trigger, "-push-event")
			checkHook = true
		} else if strings.HasSuffix(trigger.Name, "-pullrequest-event") {
			hook = getHookFromTrigger(trigger, "-pullrequest-event")
			checkHook = true
		}
		if checkHook && !containedInArray(hooks, hook) {
			hooks = append(hooks, hook)
		}
	}
	return hooks, nil
}

func getHookFromTrigger(t v1alpha1.EventListenerTrigger, suffix string) webhook {

	var releaseName, namespace, serviceaccount, pulltask, dockerreg, helmsecret, repo, gitSecret string
	for _, param := range t.Params {
		switch param.Name {
		case "webhooks-tekton-release-name":
			releaseName = param.Value.StringVal
		case "webhooks-tekton-target-namespace":
			namespace = param.Value.StringVal
		case "webhooks-tekton-service-account":
			serviceaccount = param.Value.StringVal
		case "webhooks-tekton-pull-task":
			pulltask = param.Value.StringVal
		case "webhooks-tekton-docker-registry":
			dockerreg = param.Value.StringVal
		case "webhooks-tekton-helm-secret":
			helmsecret = param.Value.StringVal
		}
	}

	for _, header := range t.Interceptor.Header {
		switch header.Name {
		case "Wext-Repository-Url":
			repo = header.Value.StringVal
		case "Wext-Secret-Name":
			gitSecret = header.Value.StringVal
		}
	}

	triggerAsHook := webhook{
		Name:             strings.TrimSuffix(t.Name, "-"+namespace+suffix),
		Namespace:        namespace,
		Pipeline:         strings.TrimSuffix(t.Template.Name, "-template"),
		GitRepositoryURL: repo,
		HelmSecret:       helmsecret,
		PullTask:         pulltask,
		DockerRegistry:   dockerreg,
		ServiceAccount:   serviceaccount,
		ReleaseName:      releaseName,
		AccessTokenRef:   gitSecret,
	}

	return triggerAsHook
}

func containedInArray(array []webhook, hook webhook) bool {
	for _, item := range array {
		if item == hook {
			return true
		}
	}
	return false
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

// getWebhookSecretTokens returns the "secretToken" and "accessToken" stored in the Secret
// with the name specified by the parameter, and in the namespace specified by r.Defaults.Namespace.
func (r Resource) getWebhookSecretTokens(name string) (accessToken string, secretToken string, err error) {
	// Access token is stored as 'accessToken' and secret as 'secretToken'
	secret, err := r.K8sClient.CoreV1().Secrets(r.Defaults.Namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return "", "", xerrors.Errorf("error getting Webhook secret. Error was: %w", err)
	}
	accessToken = string(secret.Data["accessToken"])
	secretToken = string(secret.Data["secretToken"])
	return accessToken, secretToken, nil
}

// createOAuth2Client returns an HTTP client with oauth2 authentication using the provided accessToken
func createOAuth2Client(ctx context.Context, accessToken string) *http.Client {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: accessToken})
	return oauth2.NewClient(ctx, ts)
}

// doGitHubWebhookRequest executes a GitHub PubSubHubbub request for the specified webhook.
// hubMode: "subscribe" or "unsubscribe"
// events: the list of events to subscribe to or unsubscribe from; for example, {"push", "pull_request"}
func (r Resource) doGitHubWebhookRequest(webhook webhook, hubMode string, events []string) error {
	// Access token is stored as 'accessToken' and secret as 'secretToken'
	accessToken, secretToken, err := r.getWebhookSecretTokens(webhook.AccessTokenRef)
	if err != nil {
		return err
	}

	// Create http client
	ctx := context.Background()
	client := createOAuth2Client(ctx, accessToken)

	return doGitHubHubbubRequest(client, webhook.GitRepositoryURL, hubMode, r.Defaults.CallbackURL, secretToken, events)
}
