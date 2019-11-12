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
	"bytes"
	"encoding/json"
	"fmt"
	restful "github.com/emicklei/go-restful"
	pipelinesv1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	v1alpha1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
)

var server *httptest.Server

// All event sources will be created in the "default" namespace because the INSTALLED_NAMESPACE env variable is not set
const installNs = "default"
const defaultRegistry = "default.docker.reg:8500/foo"

func setUpServer() *Resource {
	wsContainer := restful.NewContainer()
	resource := dummyResource()
	resource.K8sClient.CoreV1().Namespaces().Delete(installNs, &metav1.DeleteOptions{})
	resource.K8sClient.CoreV1().Namespaces().Create(&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: installNs}})
	server = httptest.NewServer(wsContainer)
	resource.RegisterExtensionWebService(wsContainer)
	return resource
}

func TestGetNoServiceDashboardURL(t *testing.T) {
	r := dummyResource()
	dashboard := r.getDashboardURL(installNs)
	if dashboard != "http://localhost:9097/" {
		t.Errorf("Dashboard URL not http://localhost:9097/ when no dashboard service found.  URL was %s", dashboard)
	}
}

func TestGetServiceDashboardURL(t *testing.T) {
	r := dummyResource()
	svc := createDashboardService("fake-dashboard", "tekton-dashboard")
	_, err := r.K8sClient.CoreV1().Services(installNs).Create(svc)
	if err != nil {
		t.Errorf("Error registering service")
	}
	dashboard := r.getDashboardURL(installNs)

	if dashboard != "http://fake-dashboard:1234/v1/namespaces/default/endpoints" {
		t.Errorf("Dashboard URL not http://fake-dashboard:1234/v1/namespaces/default/endpoints.  URL was %s", dashboard)
	}
}

func TestGetOpenshiftServiceDashboardURL(t *testing.T) {
	r := dummyResource()
	svc := createDashboardService("fake-openshift-dashboard", "tekton-dashboard-internal")
	_, err := r.K8sClient.CoreV1().Services(installNs).Create(svc)
	if err != nil {
		t.Errorf("Error registering service")
	}
	os.Setenv("PLATFORM", "openshift")
	dashboard := r.getDashboardURL(installNs)

	if dashboard != "http://fake-openshift-dashboard:1234/v1/namespaces/default/endpoints" {
		t.Errorf("Dashboard URL not http://fake-dashboard:1234/v1/namespaces/default/endpoints.  URL was %s", dashboard)
	}
}

func TestNewTrigger(t *testing.T) {
	r := dummyResource()

	params := []pipelinesv1alpha1.Param{
		{Name: "My-Param1", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: "myParam1Value"}},
		{Name: "My-Param2", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: "myParam2Value"}},
	}

	trigger := r.newTrigger("myName", "myBindingName", "myTemplateName", "myRepoURL", "myEvent", "mySecretName", params)
	expectedTrigger := createTrigger("myName", "myBindingName", "myTemplateName", "myRepoURL", "myEvent", "mySecretName", params, r)

	if !reflect.DeepEqual(trigger, expectedTrigger) {
		t.Errorf("Eventlistener trigger did not match expectation")
		t.Errorf("got: %+v", trigger)
		t.Errorf("expected: %+v", expectedTrigger)
	}
}

func TestGetParams(t *testing.T) {
	var webHooks = []webhook{
		{
			Name:             "name1",
			Namespace:        installNs,
			GitRepositoryURL: "https://github.com/owner/repo",
			AccessTokenRef:   "token1",
			Pipeline:         "pipeline1",
			DockerRegistry:   "registry1",
			HelmSecret:       "helmsecret1",
			ReleaseName:      "releasename1",
			PullTask:         "pulltask1",
			OnSuccessComment: "onsuccesscomment1",
			OnFailureComment: "onfailurecomment1",
		},
		{
			Name:             "name2",
			Namespace:        "foo",
			GitRepositoryURL: "https://github.com/owner/repo2",
			AccessTokenRef:   "token2",
			Pipeline:         "pipeline2",
			DockerRegistry:   "registry2",
			OnSuccessComment: "onsuccesscomment2",
			OnFailureComment: "onfailurecomment2",
		},
		{
			Name:             "name3",
			Namespace:        "foo2",
			GitRepositoryURL: "https://github.com/owner/repo3",
			AccessTokenRef:   "token3",
			Pipeline:         "pipeline3",
			ServiceAccount:   "my-sa",
		},
	}

	r := dummyResource()
	for _, hook := range webHooks {
		hookParams, monitorParams := r.getParams(hook)
		expectedHookParams, expectedMonitorParams := getExpectedParams(hook, r)
		if !reflect.DeepEqual(hookParams, expectedHookParams) {
			t.Error("The webhook params returned from r.getParams were not as expected")
			t.Errorf("got hookParams: %+v", hookParams)
			t.Errorf("expected: %+v", expectedHookParams)
		}
		if !reflect.DeepEqual(monitorParams, expectedMonitorParams) {
			t.Error("The monitor params returned from r.getParams were not as expected")
			t.Errorf("monitorParams: %+v", monitorParams)
			t.Errorf("expected: %+v", expectedMonitorParams)
		}
	}
}

func TestCreateEventListener(t *testing.T) {
	var hooks = []webhook{
		{
			Name:             "name1",
			Namespace:        installNs,
			GitRepositoryURL: "https://github.com/owner/repo",
			AccessTokenRef:   "token1",
			Pipeline:         "pipeline1",
			DockerRegistry:   "registry1",
			HelmSecret:       "helmsecret1",
			ReleaseName:      "releasename1",
			PullTask:         "pulltask1",
			OnSuccessComment: "onsuccesscomment1",
			OnFailureComment: "onfailurecomment1",
		},
		{
			Name:             "name2",
			Namespace:        "foo",
			GitRepositoryURL: "https://github.com/owner/repo2",
			AccessTokenRef:   "token2",
			Pipeline:         "pipeline2",
			DockerRegistry:   "registry2",
			OnSuccessComment: "onsuccesscomment2",
			OnFailureComment: "onfailurecomment2",
		},
		{
			Name:             "name3",
			Namespace:        "foo2",
			GitRepositoryURL: "https://github.com/owner/repo3",
			AccessTokenRef:   "token3",
			Pipeline:         "pipeline3",
			ServiceAccount:   "my-sa",
			PullTask:         "check-me",
		},
	}

	r := dummyResource()

	for _, hook := range hooks {
		el, err := r.createEventListener(hook, "install-namespace", "my/monitor/name")
		if err != nil {
			t.Errorf("Error creating eventlistener: %s", err)
		}

		if el.Name != "tekton-webhooks-eventlistener" {
			t.Errorf("Eventlistener name was: %s, expected: tekton-webhooks-eventlistener", el.Name)
		}
		if el.Namespace != "install-namespace" {
			t.Errorf("Eventlistener namespace was: %s, expected: install-namespace", el.Namespace)
		}
		if el.Spec.ServiceAccountName != "tekton-webhooks-extension-eventlistener" {
			t.Errorf("Eventlistener service account was: %s, expected tekton-webhooks-extension-eventlistener", el.Spec.ServiceAccountName)
		}
		if len(el.Spec.Triggers) != 3 {
			t.Errorf("Eventlistener had %d triggers, but expected 3", len(el.Spec.Triggers))
		} else {
			expectedTriggers := getExpectedTriggers(hook, "my/monitor/name", r)
			if !reflect.DeepEqual(el.Spec.Triggers, expectedTriggers) {
				t.Errorf("Eventlistener trigger did not match expectation")
				t.Errorf("got: %+v", el.Spec.Triggers)
				t.Errorf("expected: %+v", expectedTriggers)
			}
		}
		err = r.TriggersClient.TektonV1alpha1().EventListeners("install-namespace").Delete(el.Name, &metav1.DeleteOptions{})
		if err != nil {
			t.Error("Error occurred deleting eventlistener")
		}
	}
}

func TestUpdateEventListenerTriggerListing(t *testing.T) {
	var hooks = []webhook{
		{
			Name:             "name1",
			Namespace:        installNs,
			GitRepositoryURL: "https://github.com/owner/repo",
			AccessTokenRef:   "token1",
			Pipeline:         "pipeline1",
			DockerRegistry:   "registry1",
			HelmSecret:       "helmsecret1",
			ReleaseName:      "releasename1",
			PullTask:         "pulltask1",
			OnSuccessComment: "onsuccesscomment1",
			OnFailureComment: "onfailurecomment1",
		},
		{
			Name:             "name2",
			Namespace:        "foo",
			GitRepositoryURL: "https://github.com/owner/repo",
			AccessTokenRef:   "token2",
			Pipeline:         "pipeline2",
			DockerRegistry:   "registry2",
			OnSuccessComment: "onsuccesscomment2",
			OnFailureComment: "onfailurecomment2",
		},
		{
			Name:             "name3",
			Namespace:        "foo2",
			GitRepositoryURL: "https://github.com/owner/repo2",
			AccessTokenRef:   "token3",
			Pipeline:         "pipeline3",
			ServiceAccount:   "my-sa",
			PullTask:         "check-me",
		},
	}

	r := dummyResource()
	os.Setenv("SERVICE_ACCOUNT", "tekton-test-service-account")

	el, err := r.createEventListener(hooks[0], "install-namespace", hooks[0].GitRepositoryURL[strings.LastIndex(hooks[0].GitRepositoryURL, ":")+3:])
	if err != nil {
		t.Errorf("Error creating eventlistener: %s", err)
	}
	el, err = r.updateEventListener(el, hooks[1], hooks[1].GitRepositoryURL[strings.LastIndex(hooks[1].GitRepositoryURL, ":")+3:])
	if err != nil {
		t.Errorf("Error updating eventlistener - first time: %s", err)
	}
	el, err = r.updateEventListener(el, hooks[2], hooks[2].GitRepositoryURL[strings.LastIndex(hooks[2].GitRepositoryURL, ":")+3:])
	if err != nil {
		t.Errorf("Error updating eventlistener - second time: %s", err)
	}

	// Two of the webhooks are on the same repo - therefore only one monitor trigger for these
	if len(el.Spec.Triggers) != 8 {
		t.Errorf("Eventlistener had %d triggers, but expected 8", len(el.Spec.Triggers))
	} else {
		// getExpectedTriggers returns 3 triggers per hook (push, pullrequest and monitor)
		// in the event that multiple webhooks have been created for the same repository we
		// would only expect one occurence of the monitor, so we need to filter the expected
		// triggers such that the monitor is only expected once.
		expectedTriggers := []v1alpha1.EventListenerTrigger{}
		triggerNamesExpected := make(map[string]string)
		for _, hook := range hooks {
			for _, t := range getExpectedTriggers(hook, hook.GitRepositoryURL[strings.LastIndex(hook.GitRepositoryURL, ":")+3:], r) {
				if triggerNamesExpected[t.Name] == "" {
					triggerNamesExpected[t.Name] = "added"
					expectedTriggers = append(expectedTriggers, t)
				}
			}
		}

		if !reflect.DeepEqual(el.Spec.Triggers, expectedTriggers) {
			t.Errorf("Eventlistener trigger did not match expectation")
			t.Errorf("got: %+v", el.Spec.Triggers)
			t.Errorf("expected: %+v", expectedTriggers)
		}
	}

	err = r.TriggersClient.TektonV1alpha1().EventListeners("install-namespace").Delete(el.Name, &metav1.DeleteOptions{})
	if err != nil {
		t.Error("Error occurred deleting eventlistener")
	}

}

func TestDeleteFromEventListener(t *testing.T) {
	var hooks = []webhook{
		{
			Name:             "name1",
			Namespace:        installNs,
			GitRepositoryURL: "https://github.com/owner/repo",
			AccessTokenRef:   "token1",
			Pipeline:         "pipeline1",
			DockerRegistry:   "registry1",
			HelmSecret:       "helmsecret1",
			ReleaseName:      "releasename1",
			PullTask:         "pulltask1",
			OnSuccessComment: "onsuccesscomment1",
			OnFailureComment: "onfailurecomment1",
		},
		{
			Name:             "name2",
			Namespace:        "foo",
			GitRepositoryURL: "https://github.com/owner/repo",
			AccessTokenRef:   "token2",
			Pipeline:         "pipeline2",
			DockerRegistry:   "registry2",
			OnSuccessComment: "onsuccesscomment2",
			OnFailureComment: "onfailurecomment2",
		},
	}

	r := dummyResource()
	os.Setenv("SERVICE_ACCOUNT", "tekton-test-service-account")
	el, err := r.createEventListener(hooks[0], "install-namespace", hooks[0].GitRepositoryURL[strings.LastIndex(hooks[0].GitRepositoryURL, ":")+3:])
	if err != nil {
		t.Errorf("Error creating eventlistener: %s", err)
	}
	el, err = r.updateEventListener(el, hooks[1], hooks[1].GitRepositoryURL[strings.LastIndex(hooks[1].GitRepositoryURL, ":")+3:])
	if err != nil {
		t.Errorf("Error updating eventlistener: %s", err)
	}

	if len(el.Spec.Triggers) != 5 {
		t.Errorf("Eventlistener had %d triggers, but expected 5", len(el.Spec.Triggers))
	}

	// getExpectedTriggers returns 3 triggers per hook (push, pullrequest and monitor)
	// in the event that multiple webhooks have been created for the same repository we
	// would only expect one occurence of the monitor, so we need to filter the expected
	// triggers such that the monitor is only expected once.
	expectedTriggers := []v1alpha1.EventListenerTrigger{}
	triggerNamesExpected := make(map[string]string)
	for _, hook := range hooks {
		for _, t := range getExpectedTriggers(hook, hook.GitRepositoryURL[strings.LastIndex(hook.GitRepositoryURL, ":")+3:], r) {
			if triggerNamesExpected[t.Name] == "" {
				triggerNamesExpected[t.Name] = "added"
				expectedTriggers = append(expectedTriggers, t)
			}
		}
	}

	if !reflect.DeepEqual(el.Spec.Triggers, expectedTriggers) {
		t.Errorf("Eventlistener trigger did not match expectation")
		t.Errorf("got: %+v", el.Spec.Triggers)
		t.Errorf("expected: %+v", expectedTriggers)
	}

	err = r.deleteFromEventListener(hooks[1].Name+"-"+hooks[1].Namespace, "install-namespace", hooks[1].GitRepositoryURL[strings.LastIndex(hooks[1].GitRepositoryURL, ":")+3:], "https://github.com/owner/repo")
	if err != nil {
		t.Errorf("Error deleting entry from eventlistener: %s", err)
	}

	el, err = r.TriggersClient.TektonV1alpha1().EventListeners("install-namespace").Get("", metav1.GetOptions{})
	if len(el.Spec.Triggers) != 3 {
		t.Errorf("Eventlistener had %d triggers, but expected 3", len(el.Spec.Triggers))
	}

	expectedRemainingTrigger := getExpectedTriggers(hooks[0], hooks[0].GitRepositoryURL[strings.LastIndex(hooks[0].GitRepositoryURL, ":")+3:], r)
	if !reflect.DeepEqual(el.Spec.Triggers, expectedRemainingTrigger) {
		t.Errorf("Eventlistener trigger did not match expectation")
		t.Errorf("got: %+v", el.Spec.Triggers)
		t.Errorf("expected: %+v", expectedRemainingTrigger)
	}
}

func TestCreateAndDeleteWebhook(t *testing.T) {
	r := setUpServer()
	os.Setenv("SERVICE_ACCOUNT", "tekton-test-service-account")

	newDefaults := EnvDefaults{
		Namespace:      installNs,
		DockerRegistry: defaultRegistry,
	}
	r = updateResourceDefaults(r, newDefaults)

	var hooks = []webhook{
		{
			Name:             "name1",
			Namespace:        installNs,
			GitRepositoryURL: "https://github.com/owner/repo",
			AccessTokenRef:   "token1",
			Pipeline:         "pipeline1",
			DockerRegistry:   "registry1",
			HelmSecret:       "helmsecret1",
			ReleaseName:      "releasename1",
			OnSuccessComment: "onsuccesscomment1",
			OnFailureComment: "onfailurecomment1",
		},
		{
			Name:             "name2",
			Namespace:        "foo",
			GitRepositoryURL: "https://github.com/owner/repo",
			AccessTokenRef:   "token2",
			Pipeline:         "pipeline2",
			DockerRegistry:   "registry2",
			OnSuccessComment: "onsuccesscomment2",
			OnFailureComment: "onfailurecomment2",
		},
		{
			Name:             "name3",
			Namespace:        "foo2",
			GitRepositoryURL: "https://github.com/owner/repo2",
			AccessTokenRef:   "token3",
			Pipeline:         "pipeline3",
			ServiceAccount:   "my-sa",
			PullTask:         "check-me",
		},
	}

	for _, h := range hooks {
		resp := createWebhook(h, r)
		if resp.StatusCode() != 400 {
			t.Errorf("Webhook creation succeeded for webhook %s but was expected to fail due to lack of triggertemplate and triggerbinding", h.Name)
		}
	}
	testGetAllWebhooks([]webhook{}, r, t)

	for _, h := range hooks {
		createTriggerResources(h, r)

		resp := createWebhook(h, r)
		if resp.StatusCode() != 201 {
			t.Errorf("Webhook creation failed for webhook %s but was expected to succeed", h.Name)
		}
	}
	testGetAllWebhooks(hooks, r, t)

	// Delete the first webhook
	deleteRequest, _ := http.NewRequest(http.MethodDelete, server.URL+"/webhooks/"+hooks[0].Name+"?namespace="+hooks[0].Namespace+"&repository="+hooks[0].GitRepositoryURL, nil)
	http.DefaultClient.Do(deleteRequest)
	testGetAllWebhooks(hooks[1:], r, t)
}

func TestDockerRegUnset(t *testing.T) {
	r := dummyResource()
	// Get the docker registry using the endpoint, expect ""
	defaults := getEnvDefaults(r, t)
	reg := defaults.DockerRegistry
	if reg != "" {
		t.Errorf("Incorrect defaultDockerRegistry, expected \"\" but was: %s", reg)
	}
}

func TestDockerRegSet(t *testing.T) {
	r := dummyResource()
	newDefaults := EnvDefaults{
		Namespace:      installNs,
		DockerRegistry: defaultRegistry,
	}

	r = updateResourceDefaults(r, newDefaults)
	// Get the docker registry using the endpoint, expect ""
	defaults := getEnvDefaults(r, t)

	reg := defaults.DockerRegistry
	if reg != "default.docker.reg:8500/foo" {
		t.Errorf("Incorrect defaultDockerRegistry, expected default.docker.reg:8500/foo, but was: %s", reg)
	}
}

func TestDeleteByNameNoName405(t *testing.T) {

	httpReq, _ := http.NewRequest(http.MethodDelete, server.URL+"/webhooks/?namespace=foo&repository=bar", nil)
	response, _ := http.DefaultClient.Do(httpReq)
	if response.StatusCode != 405 {
		t.Errorf("Status code not set to 405 when deleting without a name, it's: %d", response.StatusCode)
	}
}

func TestDeleteByNameNoNamespaceOrRepoBadRequest(t *testing.T) {
	setUpServer()
	httpReq, _ := http.NewRequest(http.MethodDelete, server.URL+"/webhooks/foo", nil)
	response, _ := http.DefaultClient.Do(httpReq)
	if response.StatusCode != 400 {
		t.Errorf("Status code not set to 400 when deleting without a namespace, it's: %d", response.StatusCode)
	}
}

func TestDeleteByNameNoNamespaceBadRequest(t *testing.T) {
	setUpServer()
	httpReq, _ := http.NewRequest(http.MethodDelete, server.URL+"/webhooks/foo?repository=bar", nil)
	response, _ := http.DefaultClient.Do(httpReq)
	if response.StatusCode != 400 {
		t.Errorf("Status code not set to 400 when deleting without a namespace, it's: %d", response.StatusCode)
	}
}

func TestDeleteByNameNoRepoBadRequest(t *testing.T) {
	setUpServer()
	httpReq, _ := http.NewRequest(http.MethodDelete, server.URL+"/webhooks/foo?namespace=foo", nil)
	response, _ := http.DefaultClient.Do(httpReq)
	if response.StatusCode != 400 {
		t.Errorf("Status code not set to 400 when deleting without a repository, it's: %d", response.StatusCode)
	}
}

//------------------- UTILS -------------------//

func createDashboardService(name, labelValue string) *corev1.Service {
	labels := make(map[string]string)
	labels["app"] = labelValue

	dashSVC := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Labels:      labels,
			Annotations: map[string]string{},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				corev1.ServicePort{
					Name:       "http",
					Protocol:   "TCP",
					Port:       1234,
					NodePort:   5678,
					TargetPort: intstr.FromInt(91011),
				},
			},
		},
		Status: corev1.ServiceStatus{},
	}
	return dashSVC
}

func getExpectedParams(hook webhook, r *Resource) (expectedHookParams, expectedMonitorParams []pipelinesv1alpha1.Param) {
	url := strings.TrimPrefix(hook.GitRepositoryURL, "https://")
	url = strings.TrimPrefix(url, "http://")

	server := url[0:strings.Index(url, "/")]
	org := strings.TrimPrefix(url, server+"/")
	org = org[0:strings.Index(org, "/")]
	repo := url[strings.LastIndex(url, "/")+1:]

	expectedHookParams = []pipelinesv1alpha1.Param{}
	if hook.ReleaseName != "" {
		expectedHookParams = append(expectedHookParams, pipelinesv1alpha1.Param{Name: "webhooks-tekton-release-name", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: hook.ReleaseName}})
	} else {
		expectedHookParams = append(expectedHookParams, pipelinesv1alpha1.Param{Name: "webhooks-tekton-release-name", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: hook.GitRepositoryURL[strings.LastIndex(hook.GitRepositoryURL, "/")+1:]}})
	}
	expectedHookParams = append(expectedHookParams, pipelinesv1alpha1.Param{Name: "webhooks-tekton-target-namespace", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: hook.Namespace}})
	expectedHookParams = append(expectedHookParams, pipelinesv1alpha1.Param{Name: "webhooks-tekton-service-account", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: hook.ServiceAccount}})
	expectedHookParams = append(expectedHookParams, pipelinesv1alpha1.Param{Name: "webhooks-tekton-git-server", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: server}})
	expectedHookParams = append(expectedHookParams, pipelinesv1alpha1.Param{Name: "webhooks-tekton-git-org", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: org}})
	expectedHookParams = append(expectedHookParams, pipelinesv1alpha1.Param{Name: "webhooks-tekton-git-repo", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: repo}})
	if hook.DockerRegistry != "" {
		expectedHookParams = append(expectedHookParams, pipelinesv1alpha1.Param{Name: "webhooks-tekton-docker-registry", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: hook.DockerRegistry}})
	}
	if hook.HelmSecret != "" {
		expectedHookParams = append(expectedHookParams, pipelinesv1alpha1.Param{Name: "webhooks-tekton-helm-secret", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: hook.HelmSecret}})
	}

	expectedMonitorParams = []pipelinesv1alpha1.Param{}
	if hook.OnSuccessComment != "" {
		expectedMonitorParams = append(expectedMonitorParams, pipelinesv1alpha1.Param{Name: "commentsuccess", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: hook.OnSuccessComment}})
	} else {
		expectedMonitorParams = append(expectedMonitorParams, pipelinesv1alpha1.Param{Name: "commentsuccess", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: "Success"}})
	}
	if hook.OnFailureComment != "" {
		expectedMonitorParams = append(expectedMonitorParams, pipelinesv1alpha1.Param{Name: "commentfailure", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: hook.OnFailureComment}})
	} else {
		expectedMonitorParams = append(expectedMonitorParams, pipelinesv1alpha1.Param{Name: "commentfailure", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: "Failed"}})
	}
	expectedMonitorParams = append(expectedMonitorParams, pipelinesv1alpha1.Param{Name: "gitsecretname", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: hook.AccessTokenRef}})
	expectedMonitorParams = append(expectedMonitorParams, pipelinesv1alpha1.Param{Name: "gitsecretkeyname", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: "accessToken"}})
	expectedMonitorParams = append(expectedMonitorParams, pipelinesv1alpha1.Param{Name: "dashboardurl", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: r.getDashboardURL(r.Defaults.Namespace)}})

	return
}

func getExpectedTriggers(hook webhook, monitorTriggerName string, r *Resource) []v1alpha1.EventListenerTrigger {
	expectedHookParams, expectedMonitorParams := getExpectedParams(hook, r)
	push := createTrigger(hook.Name+"-"+hook.Namespace+"-push-event",
		hook.Pipeline+"-push-binding",
		hook.Pipeline+"-template",
		hook.GitRepositoryURL,
		"push",
		hook.AccessTokenRef,
		expectedHookParams,
		r)

	pullRequest := createTrigger(hook.Name+"-"+hook.Namespace+"-pullrequest-event",
		hook.Pipeline+"-pullrequest-binding",
		hook.Pipeline+"-template",
		hook.GitRepositoryURL,
		"pull_request",
		hook.AccessTokenRef,
		expectedHookParams,
		r)

	monitor := createTrigger(monitorTriggerName,
		hook.PullTask+"-binding",
		hook.PullTask+"-template",
		hook.GitRepositoryURL,
		"pull_request",
		hook.AccessTokenRef,
		expectedMonitorParams,
		r)

	triggers := []v1alpha1.EventListenerTrigger{push, pullRequest, monitor}
	return triggers
}

func createTrigger(name, bindingName, templateName, repoURL, event, secretName string, params []pipelinesv1alpha1.Param, r *Resource) v1alpha1.EventListenerTrigger {
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

func getEnvDefaults(r *Resource, t *testing.T) EnvDefaults {
	httpReq := dummyHTTPRequest("GET", "http://wwww.dummy.com:8080/webhooks/defaults", nil)
	req := dummyRestfulRequest(httpReq, "")
	httpWriter := httptest.NewRecorder()
	resp := dummyRestfulResponse(httpWriter)
	r.getDefaults(req, resp)

	defaults := EnvDefaults{}
	err := json.NewDecoder(httpWriter.Body).Decode(&defaults)
	if err != nil {
		t.Errorf("Error decoding result into defaults{}: %s", err.Error())
	}
	return defaults
}

func createWebhook(webhook webhook, r *Resource) (response *restful.Response) {

	b, err := json.Marshal(webhook)
	if err != nil {
		fmt.Println(fmt.Errorf("Marshal error when creating webhook, data is: %s, error is: %s", b, err))
		return nil
	}

	httpReq := dummyHTTPRequest("POST", "http://wwww.dummy.com:8080/webhooks/", bytes.NewBuffer(b))
	req := dummyRestfulRequest(httpReq, "")
	httpWriter := httptest.NewRecorder()
	resp := dummyRestfulResponse(httpWriter)
	r.createWebhook(req, resp)
	return resp
}

func testGetAllWebhooks(expectedWebhooks []webhook, r *Resource, t *testing.T) {
	httpReq := dummyHTTPRequest("GET", "http://wwww.dummy.com:8080/webhooks/", nil)
	req := dummyRestfulRequest(httpReq, "")
	httpWriter := httptest.NewRecorder()
	resp := dummyRestfulResponse(httpWriter)
	r.getAllWebhooks(req, resp)
	actualWebhooks := []webhook{}
	err := json.NewDecoder(httpWriter.Body).Decode(&actualWebhooks)
	if err != nil {
		t.Errorf("Error decoding result into []webhook{}: %s", err.Error())
		return
	}

	fmt.Printf("%+v", actualWebhooks)

	if len(expectedWebhooks) != len(actualWebhooks) {
		t.Errorf("Incorrect length of result, expected %d, but was %d", len(expectedWebhooks), len(actualWebhooks))
		return
	}

	// Now compare the arrays expectedWebhooks and actualWebhooks by turning them into maps
	expected := map[webhook]bool{}
	actual := map[webhook]bool{}
	for i := range expectedWebhooks {
		if expectedWebhooks[i].DockerRegistry == "" {
			expectedWebhooks[i].DockerRegistry = defaultRegistry
		}
		if expectedWebhooks[i].PullTask == "" {
			expectedWebhooks[i].PullTask = "monitor-task"
		}
		expected[expectedWebhooks[i]] = true
		actual[actualWebhooks[i]] = true
	}

	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Webhook error: expected: \n%v \nbut received \n%v", expected, actual)
	}
}

func createTriggerResources(hook webhook, r *Resource) {
	template := v1alpha1.TriggerTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      hook.Pipeline + "-template",
			Namespace: installNs,
		},
	}

	pushBinding := v1alpha1.TriggerBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      hook.Pipeline + "-push-binding",
			Namespace: installNs,
		},
	}

	pullBinding := v1alpha1.TriggerBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      hook.Pipeline + "-pullrequest-binding",
			Namespace: installNs,
		},
	}

	_, err := r.TriggersClient.TektonV1alpha1().TriggerTemplates(installNs).Create(&template)
	if err != nil {
		fmt.Printf("Error creating fake triggertemplate %s", template.Name)
	}
	_, err = r.TriggersClient.TektonV1alpha1().TriggerBindings(installNs).Create(&pushBinding)
	if err != nil {
		fmt.Printf("Error creating fake triggerbinding %s", pushBinding.Name)
	}
	_, err = r.TriggersClient.TektonV1alpha1().TriggerBindings(installNs).Create(&pullBinding)
	if err != nil {
		fmt.Printf("Error creating fake triggerbinding %s", pullBinding.Name)
	}

	return

}
