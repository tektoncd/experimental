// /*
// Copyright 2019-20 The Tekton Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// 		http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

package endpoints

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strconv"
	"testing"

	"strings"

	restful "github.com/emicklei/go-restful"
	"github.com/google/go-cmp/cmp"
	routesv1 "github.com/openshift/api/route/v1"
	"github.com/tektoncd/experimental/webhooks-extension/pkg/utils"
	pipelinesv1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	v1alpha1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var server *httptest.Server

type testcase struct {
	Webhook            webhook
	MonitorTriggerName string
	expectedProvider   string
	expectedAPIURL     string
}

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
	labels := map[string]string{
		"app.kubernetes.io/name":      "dashboard",
		"app.kubernetes.io/component": "dashboard",
		"app.kubernetes.io/instance":  "default",
		"app.kubernetes.io/part-of":   "tekton-dashboard",
	}
	svc := createDashboardService("fake-dashboard", labels)
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
	labels := map[string]string{
		"app": "tekton-dashboard-internal",
	}
	svc := createDashboardService("fake-openshift-dashboard", labels)
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
	trigger := r.newTrigger("myName", "myBindingName", "myTemplateName", "myRepoURL", "myEvent", "mySecretName", "foo1234")
	expectedTrigger := v1alpha1.EventListenerTrigger{
		Name: "myName",
		Bindings: []*v1alpha1.EventListenerBinding{
			{
				Name:       "myBindingName",
				APIVersion: "v1alpha1",
			},
			{
				Name:       "foo1234",
				APIVersion: "v1alpha1",
			},
		},
		Template: v1alpha1.EventListenerTemplate{
			Name:       "myTemplateName",
			APIVersion: "v1alpha1",
		},
		Interceptors: []*v1alpha1.EventInterceptor{
			{
				Webhook: &v1alpha1.WebhookInterceptor{
					Header: []pipelinesv1alpha1.Param{
						{Name: "Wext-Trigger-Name", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: "myName"}},
						{Name: "Wext-Repository-Url", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: "myRepoURL"}},
						{Name: "Wext-Incoming-Event", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: "myEvent"}},
						{Name: "Wext-Secret-Name", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: "mySecretName"}}},
					ObjectRef: &corev1.ObjectReference{
						APIVersion: "v1",
						Kind:       "Service",
						Name:       "tekton-webhooks-extension-validator",
						Namespace:  r.Defaults.Namespace,
					},
				},
			},
		},
	}

	if !reflect.DeepEqual(trigger, expectedTrigger) {
		t.Errorf("Eventlistener trigger did not match expectation")
		t.Errorf("got: %+v", trigger)
		t.Errorf("expected: %+v", expectedTrigger)
	}
}

func TestGetParams(t *testing.T) {
	var testcases = []testcase{
		{
			Webhook: webhook{
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
				OnTimeoutComment: "ontimeoutcomment1",
				OnMissingComment: "onmissingcomment1",
			},
			expectedProvider: "github",
			expectedAPIURL:   "https://api.github.com/",
		},
		{
			Webhook: webhook{
				Name:             "name2",
				Namespace:        "foo",
				GitRepositoryURL: "https://github.com/owner/repo2",
				AccessTokenRef:   "token2",
				Pipeline:         "pipeline2",
				DockerRegistry:   "registry2",
				OnSuccessComment: "onsuccesscomment2",
				OnFailureComment: "onfailurecomment2",
				OnTimeoutComment: "ontimeoutcomment2",
				OnMissingComment: "onmissingcomment2",
			},
			expectedProvider: "github",
			expectedAPIURL:   "https://api.github.com/",
		},
		{
			Webhook: webhook{
				Name:             "name3",
				Namespace:        "foo2",
				GitRepositoryURL: "https://github.com/owner/repo3",
				AccessTokenRef:   "token3",
				Pipeline:         "pipeline3",
				ServiceAccount:   "my-sa",
			},
			expectedProvider: "github",
			expectedAPIURL:   "https://api.github.com/",
		},
		{
			Webhook: webhook{
				Name:             "name4",
				Namespace:        "foo2",
				GitRepositoryURL: "https://gitlab.company.com/owner/repo3",
				AccessTokenRef:   "token3",
				Pipeline:         "pipeline3",
				ServiceAccount:   "my-sa",
			},
			expectedProvider: "gitlab",
			expectedAPIURL:   "https://gitlab.company.com/api/v4",
		},
		{
			Webhook: webhook{
				Name:             "name5",
				Namespace:        "foo2",
				GitRepositoryURL: "https://github.company.com/owner/repo3",
				AccessTokenRef:   "token3",
				Pipeline:         "pipeline3",
				ServiceAccount:   "my-sa",
			},
			expectedProvider: "github",
			expectedAPIURL:   "https://github.company.com/api/v3/",
		},
	}

	r := dummyResource()
	os.Setenv("SSL_VERIFICATION_ENABLED", "true")
	for _, tt := range testcases {
		hookParams, monitorParams := r.getParams(tt.Webhook)
		expectedHookParams, expectedMonitorParams := getExpectedParams(tt.Webhook, r, tt.expectedProvider, tt.expectedAPIURL)
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

func TestCompareRepos(t *testing.T) {
	type testcase struct {
		url1          string
		url2          string
		expectedMatch bool
		expectedError string
	}
	testcases := []testcase{
		{
			url1:          "Http://GitHub.Com/foo/BAR",
			url2:          "http://github.com/foo/bar",
			expectedMatch: true,
		},
		{
			url1:          "Http://GitHub.Com/foo/BAR",
			url2:          "http://github.com/foo/bar.git",
			expectedMatch: true,
		},
		{
			url1:          "Http://github.com/foo/bar.git",
			url2:          "http://github.com/foo/bar",
			expectedMatch: true,
		},
		{
			url1:          "http://gitlab.com/foo/bar",
			url2:          "http://github.com/foo/bar",
			expectedMatch: false,
		},
		{
			url1:          "http://github.com/bar/bar",
			url2:          "http://github.com/foo/bar",
			expectedMatch: false,
		},
		{
			url1:          "http://gitlab.com/foo/bar",
			url2:          "http://gitLAB.com/FoO/bar",
			expectedMatch: true,
		},
	}
	r := dummyResource()
	for _, tt := range testcases {
		match, err := r.compareGitRepoNames(tt.url1, tt.url2)
		if tt.expectedMatch != match {
			if err != nil {
				t.Errorf("url mismatch with error %s", err.Error())
			}
			t.Errorf("url mismatch unexpected: %s, %s", tt.url1, tt.url2)
		}
	}
}

func TestGenerateMonitorTriggerName(t *testing.T) {
	r := dummyResource()
	var triggers []v1alpha1.EventListenerTrigger
	triggersMap := make(map[string]v1alpha1.EventListenerTrigger)
	for i := 0; i < 2000; i++ {
		t := r.newTrigger("foo-"+strconv.Itoa(i), "foo", "foo", "https://foo.com/foo/bar", "foo", "foo", "foo")
		triggers = append(triggers, t)
		triggersMap["foo-"+strconv.Itoa(i)] = t
	}

	for j := 0; j < 5000; j++ {
		name := r.generateMonitorTriggerName("foo-", triggers)
		if _, ok := triggersMap[name]; ok {
			t.Errorf("generateMonitorTriggerName did not provide a unique name")
		}
	}
}

func TestDoesMonitorExist(t *testing.T) {
	type testcase struct {
		Webhook           webhook
		TriggerNamePrefix string
		Expected          bool
	}
	testcases := []testcase{
		{
			Webhook: webhook{
				Name:             "name1",
				Namespace:        "foo1",
				GitRepositoryURL: "https://github.com/owner/repo1",
				AccessTokenRef:   "token1",
				Pipeline:         "pipeline1",
				ServiceAccount:   "my-sa",
			},
			TriggerNamePrefix: "name1-",
			Expected:          true,
		},
		{
			Webhook: webhook{
				Name:             "name2",
				Namespace:        "foo2",
				GitRepositoryURL: "https://github.com/owner/repo2",
				AccessTokenRef:   "token2",
				Pipeline:         "pipeline2",
				ServiceAccount:   "my-sa",
			},
			TriggerNamePrefix: "name2-",
			Expected:          false,
		},
	}

	r := dummyResource()

	// Create some pre-existing triggers to pretend to be the monitor
	// we will cheat and use webhook name as the prefix
	// for the trigger name
	var eventListenerTriggers []v1alpha1.EventListenerTrigger
	for i, tt := range testcases {
		if tt.Expected {
			t := r.newTrigger(tt.Webhook.Name+"-"+strconv.Itoa(i), "foo", "foo", tt.Webhook.GitRepositoryURL, "foo", "foo", "foo")
			eventListenerTriggers = append(eventListenerTriggers, t)
		}
	}

	// Now test
	for _, tt := range testcases {
		found, _ := r.doesMonitorExist(tt.TriggerNamePrefix, tt.Webhook, eventListenerTriggers)
		if tt.Expected != found {
			t.Errorf("Unexpected result checking existence of trigger with monitorprefix %s", tt.TriggerNamePrefix)
		}
	}
}

func TestGetMonitorBindingName(t *testing.T) {
	type testcase struct {
		repoURL             string
		monitorTask         string
		expectedBindingName string
		expectedError       string
	}
	testcases := []testcase{
		{
			repoURL:             "http://foo.github.com/wibble/fish",
			monitorTask:         "monitor-task",
			expectedBindingName: "monitor-task-github-binding",
		},
		{
			repoURL:             "https://github.bob.com/foo/dog",
			monitorTask:         "wibble",
			expectedBindingName: "wibble-binding",
		},
		{
			repoURL:             "http://foo.gitlab.com/wibble/fish",
			monitorTask:         "monitor-task",
			expectedBindingName: "monitor-task-gitlab-binding",
		},
		{
			repoURL:       "",
			monitorTask:   "monitor-task",
			expectedError: "no repository URL provided on call to GetGitProviderAndAPIURL",
		},
		{
			repoURL:             "https://hungry.dinosaur.com/wibble/fish",
			monitorTask:         "monitor-task",
			expectedBindingName: "",
			expectedError:       "Git Provider for project URL: https://hungry.dinosaur.com/wibble/fish not recognized",
		},
	}

	r := dummyResource()
	for _, tt := range testcases {
		name, err := r.getMonitorBindingName(tt.repoURL, tt.monitorTask)
		if err != nil {
			if tt.expectedError != err.Error() {
				t.Errorf("unexpected error in TestGetMonitorBindingName: %s", err.Error())
			}
		}
		if name != tt.expectedBindingName {
			t.Errorf("mismatch in expected binding name, expected %s got %s", tt.expectedBindingName, name)
		}
	}
}

func TestCreateEventListener(t *testing.T) {
	hook := webhook{
		Name:             "name1",
		Namespace:        installNs,
		GitRepositoryURL: "https://github.com/owner/repo",
		AccessTokenRef:   "token1",
		Pipeline:         "pipeline1",
		DockerRegistry:   "registry1",
		HelmSecret:       "helmsecret1",
		ReleaseName:      "releasename1",
		PullTask:         "pulltask1",
	}

	r := dummyResource()
	createTriggerResources(hook, r)

	_, owner, repo, _ := r.getGitValues(hook.GitRepositoryURL)
	monitorTriggerNamePrefix := owner + "." + repo

	GetTriggerBindingObjectMeta = FakeGetTriggerBindingObjectMeta

	el, err := r.createEventListener(hook, r.Defaults.Namespace, monitorTriggerNamePrefix)
	if err != nil {
		t.Errorf("Error creating eventlistener: %s", err)
	}

	if el.Name != "tekton-webhooks-eventlistener" {
		t.Errorf("Eventlistener name was: %s, expected: tekton-webhooks-eventlistener", el.Name)
	}
	if el.Namespace != r.Defaults.Namespace {
		t.Errorf("Eventlistener namespace was: %s, expected: %s", el.Namespace, r.Defaults.Namespace)
	}
	if el.Spec.ServiceAccountName != "tekton-webhooks-extension-eventlistener" {
		t.Errorf("Eventlistener service account was: %s, expected tekton-webhooks-extension-eventlistener", el.Spec.ServiceAccountName)
	}
	if len(el.Spec.Triggers) != 3 {
		t.Errorf("Eventlistener had %d triggers, but expected 3", len(el.Spec.Triggers))
	}

	hooks, err := r.getHooksForRepo(hook.GitRepositoryURL)
	if err != nil {
		t.Errorf("Error occurred retrieving hook in getHooksForRepo: %s", err.Error())
	}
	if len(hooks) != 1 {
		t.Errorf("Unexpected number of hooks returned from getHooksForRepo: %+v", hooks)
	}
	if !reflect.DeepEqual(hooks[0], hook) {
		t.Errorf("Hook didn't match: Got %+v, Expected %+v", hooks[0], hook)
	}

	expectedTriggers := r.getExpectedPushAndPullRequestTriggersForWebhook(hook)
	for _, trigger := range el.Spec.Triggers {
		found := false
		for _, t := range expectedTriggers {
			if reflect.DeepEqual(t, trigger) {
				found = true
				break
			}
		}
		if !found {
			// Should be the monitor, can't deep equal monitor due to created name
			if !strings.HasPrefix(trigger.Name, owner+"."+repo) {
				t.Errorf("trigger %+v unexpected", trigger)
			}
			// Check params on monitor
			os.Setenv("SSL_VERIFICATION_ENABLED", "true")
			_, expectedMonitorParams := getExpectedParams(hook, r, "github", "https://api.github.com/")
			wextMonitorBindingFound := false
			for _, monitorBinding := range trigger.Bindings {
				if strings.HasPrefix(monitorBinding.Name, "wext-") {
					wextMonitorBindingFound = true
					binding, err := r.TriggersClient.TriggersV1alpha1().TriggerBindings(r.Defaults.Namespace).Get(monitorBinding.Name, metav1.GetOptions{})
					if err != nil {
						t.Errorf("%s", err.Error())
					}
					if !reflect.DeepEqual(binding.Spec.Params, expectedMonitorParams) {
						t.Error("The monitor params returned from r.getParams were not as expected")
						t.Errorf("monitorParams: %+v", binding.Spec.Params)
						t.Errorf("expected: %+v", expectedMonitorParams)
					}
				}
			}
			if !wextMonitorBindingFound {
				t.Errorf("Did not find monitor bindings")
			}
		}
	}

	err = r.TriggersClient.TriggersV1alpha1().EventListeners(r.Defaults.Namespace).Delete(el.Name, &metav1.DeleteOptions{})
	if err != nil {
		t.Errorf("Error occurred deleting eventlistener: %s", err.Error())
	}

	err = r.deleteAllBindings()
	if err != nil {
		t.Errorf("Error occurred deleting triggerbindings: %s", err.Error())
	}
}

func TestUpdateEventListener(t *testing.T) {
	var testcases = []webhook{
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
			OnTimeoutComment: "ontimeoutcomment1",
			OnMissingComment: "onmissingcomment1",
		},
		{
			Name:             "name2",
			Namespace:        "foo",
			GitRepositoryURL: "https://github.com/owner/repo",
			AccessTokenRef:   "token2",
			Pipeline:         "pipeline2",
			DockerRegistry:   "registry2",
			PullTask:         "pulltask1",
			OnSuccessComment: "onsuccesscomment2",
			OnFailureComment: "onfailurecomment2",
			OnTimeoutComment: "ontimeoutcomment2",
			OnMissingComment: "onmissingcomment2",
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
	GetTriggerBindingObjectMeta = FakeGetTriggerBindingObjectMeta

	createTriggerResources(testcases[0], r)
	_, owner, repo, _ := r.getGitValues(testcases[0].GitRepositoryURL)
	monitorTriggerNamePrefix := owner + "." + repo

	el, err := r.createEventListener(testcases[0], r.Defaults.Namespace, monitorTriggerNamePrefix)
	if err != nil {
		t.Errorf("Error creating eventlistener: %s", err)
	}

	_, owner, repo, _ = r.getGitValues(testcases[1].GitRepositoryURL)
	monitorTriggerNamePrefix = owner + "." + repo

	el, err = r.updateEventListener(el, testcases[1], monitorTriggerNamePrefix)
	if err != nil {
		t.Errorf("Error updating eventlistener - first time: %s", err)
	}

	_, owner, repo, _ = r.getGitValues(testcases[2].GitRepositoryURL)
	monitorTriggerNamePrefix = owner + "." + repo

	el, err = r.updateEventListener(el, testcases[2], monitorTriggerNamePrefix)
	if err != nil {
		t.Errorf("Error updating eventlistener - second time: %s", err)
	}

	// Two of the webhooks are on the same repo - therefore only one monitor trigger for these
	if len(el.Spec.Triggers) != 8 {
		t.Errorf("Eventlistener had %d triggers, but expected 8", len(el.Spec.Triggers))
	}

	err = r.TriggersClient.TriggersV1alpha1().EventListeners(r.Defaults.Namespace).Delete(el.Name, &metav1.DeleteOptions{})
	if err != nil {
		t.Errorf("Error occurred deleting eventlistener: %s", err.Error())
	}

	err = r.deleteAllBindings()
	if err != nil {
		t.Errorf("Error occurred deleting triggerbindings: %s", err.Error())
	}

}

func TestDeleteFromEventListener(t *testing.T) {
	var testcases = []testcase{
		{
			Webhook: webhook{
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
				OnTimeoutComment: "ontimeoutcomment1",
				OnMissingComment: "onmissingcomment1",
			},
			expectedProvider: "github",
			expectedAPIURL:   "https://api.github.com/",
		},
		{
			Webhook: webhook{
				Name:             "name2",
				Namespace:        "foo",
				GitRepositoryURL: "https://github.com/owner/repo",
				AccessTokenRef:   "token2",
				Pipeline:         "pipeline2",
				DockerRegistry:   "registry2",
				PullTask:         "pulltask1",
				OnSuccessComment: "onsuccesscomment2",
				OnFailureComment: "onfailurecomment2",
				OnTimeoutComment: "ontimeoutcomment2",
				OnMissingComment: "onmissingcomment2",
			},
			expectedProvider: "github",
			expectedAPIURL:   "https://api.github.com/",
		},
	}

	r := dummyResource()
	GetTriggerBindingObjectMeta = FakeGetTriggerBindingObjectMeta
	os.Setenv("SERVICE_ACCOUNT", "tekton-test-service-account")

	_, owner, repo, _ := r.getGitValues(testcases[0].Webhook.GitRepositoryURL)
	monitorTriggerNamePrefix := owner + "." + repo

	el, err := r.createEventListener(testcases[0].Webhook, r.Defaults.Namespace, monitorTriggerNamePrefix)
	if err != nil {
		t.Errorf("Error creating eventlistener: %s", err)
	}
	_, owner, repo, _ = r.getGitValues(testcases[1].Webhook.GitRepositoryURL)
	monitorTriggerNamePrefix = owner + "." + repo

	el, err = r.updateEventListener(el, testcases[1].Webhook, monitorTriggerNamePrefix)
	if err != nil {
		t.Errorf("Error updating eventlistener: %s", err)
	}

	if len(el.Spec.Triggers) != 5 {
		t.Errorf("Eventlistener had %d triggers, but expected 5", len(el.Spec.Triggers))
	}

	_, gitOwner, gitRepo, _ := r.getGitValues(testcases[1].Webhook.GitRepositoryURL)
	monitorTriggerNamePrefix = gitOwner + "." + gitRepo

	err = r.deleteFromEventListener(testcases[1].Webhook.Name+"-"+testcases[1].Webhook.Namespace, r.Defaults.Namespace, monitorTriggerNamePrefix, testcases[1].Webhook)
	if err != nil {
		t.Errorf("Error deleting entry from eventlistener: %s", err)
	}

	el, err = r.TriggersClient.TriggersV1alpha1().EventListeners(r.Defaults.Namespace).Get("", metav1.GetOptions{})
	if len(el.Spec.Triggers) != 3 {
		t.Errorf("Eventlistener had %d triggers, but expected 3", len(el.Spec.Triggers))
	}

}

func TestFailToCreateWebhookNoTriggerResources(t *testing.T) {
	r := setUpServer()
	os.Setenv("SERVICE_ACCOUNT", "tekton-test-service-account")

	newDefaults := EnvDefaults{
		Namespace:      installNs,
		DockerRegistry: defaultRegistry,
	}
	r = updateResourceDefaults(r, newDefaults)

	hook := webhook{
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
		OnTimeoutComment: "ontimeoutcomment1",
		OnMissingComment: "onmissingcomment1",
	}

	resp := createWebhook(hook, r)
	if resp.StatusCode() != 400 {
		t.Errorf("Webhook creation succeeded for webhook %s but was expected to fail due to lack of triggertemplate and triggerbinding", hook.Name)
	}

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
	setUpServer()
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

// //------------------- UTILS -------------------//

func createDashboardService(name string, labels map[string]string) *corev1.Service {
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

func getExpectedParams(hook webhook, r *Resource, expectedProvider, expectedAPIURL string) (expectedHookParams, expectedMonitorParams []pipelinesv1alpha1.Param) {
	url := strings.TrimPrefix(hook.GitRepositoryURL, "https://")
	url = strings.TrimPrefix(url, "http://")

	server := url[0:strings.Index(url, "/")]
	org := strings.TrimPrefix(url, server+"/")
	org = org[0:strings.Index(org, "/")]
	repo := url[strings.LastIndex(url, "/")+1:]

	sslverify := os.Getenv("SSL_VERIFICATION_ENABLED")
	insecureAsBool, _ := strconv.ParseBool(sslverify)
	insecureAsString := strconv.FormatBool(!insecureAsBool)

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
	expectedHookParams = append(expectedHookParams, pipelinesv1alpha1.Param{Name: "webhooks-tekton-pull-task", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: hook.PullTask}})
	expectedHookParams = append(expectedHookParams, pipelinesv1alpha1.Param{Name: "webhooks-tekton-ssl-verify", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: sslverify}})
	expectedHookParams = append(expectedHookParams, pipelinesv1alpha1.Param{Name: "webhooks-tekton-insecure-skip-tls-verify", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: insecureAsString}})
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
	if hook.OnTimeoutComment != "" {
		expectedMonitorParams = append(expectedMonitorParams, pipelinesv1alpha1.Param{Name: "commenttimeout", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: hook.OnTimeoutComment}})
	} else {
		expectedMonitorParams = append(expectedMonitorParams, pipelinesv1alpha1.Param{Name: "commenttimeout", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: "Unknown"}})
	}
	if hook.OnMissingComment != "" {
		expectedMonitorParams = append(expectedMonitorParams, pipelinesv1alpha1.Param{Name: "commentmissing", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: hook.OnMissingComment}})
	} else {
		expectedMonitorParams = append(expectedMonitorParams, pipelinesv1alpha1.Param{Name: "commentmissing", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: "Missing"}})
	}
	expectedMonitorParams = append(expectedMonitorParams, pipelinesv1alpha1.Param{Name: "gitsecretname", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: hook.AccessTokenRef}})
	expectedMonitorParams = append(expectedMonitorParams, pipelinesv1alpha1.Param{Name: "gitsecretkeyname", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: "accessToken"}})
	expectedMonitorParams = append(expectedMonitorParams, pipelinesv1alpha1.Param{Name: "dashboardurl", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: r.getDashboardURL(r.Defaults.Namespace)}})
	expectedMonitorParams = append(expectedMonitorParams, pipelinesv1alpha1.Param{Name: "insecure-skip-tls-verify", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: insecureAsString}})
	expectedMonitorParams = append(expectedMonitorParams, pipelinesv1alpha1.Param{Name: "provider", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: expectedProvider}})
	expectedMonitorParams = append(expectedMonitorParams, pipelinesv1alpha1.Param{Name: "apiurl", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: expectedAPIURL}})

	return
}

func (r Resource) deleteAllBindings() error {
	tbs, err := r.TriggersClient.TriggersV1alpha1().TriggerBindings(r.Defaults.Namespace).List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, tb := range tbs.Items {
		err = r.TriggersClient.TriggersV1alpha1().TriggerBindings(r.Defaults.Namespace).Delete(tb.Name, &metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}

func (r Resource) getExpectedPushAndPullRequestTriggersForWebhook(webhook webhook) []v1alpha1.EventListenerTrigger {

	triggers := []v1alpha1.EventListenerTrigger{
		{
			Name: webhook.Name + "-" + webhook.Namespace + "-push-event",
			Bindings: []*v1alpha1.EventListenerBinding{
				{
					Name:       webhook.Pipeline + "-push-binding",
					APIVersion: "v1alpha1",
				},
				{
					// This name is not as it would be in the product, as
					// GenerateName is used.
					Name:       "wext-" + webhook.Name + "-",
					APIVersion: "v1alpha1",
				},
			},
			Template: v1alpha1.EventListenerTemplate{
				Name:       webhook.Pipeline + "-template",
				APIVersion: "v1alpha1",
			},
			Interceptors: []*v1alpha1.EventInterceptor{
				{
					Webhook: &v1alpha1.WebhookInterceptor{
						Header: []pipelinesv1alpha1.Param{
							{Name: "Wext-Trigger-Name", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: webhook.Name + "-" + webhook.Namespace + "-push-event"}},
							{Name: "Wext-Repository-Url", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: webhook.GitRepositoryURL}},
							{Name: "Wext-Incoming-Event", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: "push, Push Hook, Tag Push Hook"}},
							{Name: "Wext-Secret-Name", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: webhook.AccessTokenRef}}},
						ObjectRef: &corev1.ObjectReference{
							APIVersion: "v1",
							Kind:       "Service",
							Name:       "tekton-webhooks-extension-validator",
							Namespace:  r.Defaults.Namespace,
						},
					},
				},
			},
		},
		{
			Name: webhook.Name + "-" + webhook.Namespace + "-pullrequest-event",
			Bindings: []*v1alpha1.EventListenerBinding{
				{
					Name:       webhook.Pipeline + "-pullrequest-binding",
					APIVersion: "v1alpha1",
				},
				{
					// This name is not as it would be in the product, as
					// GenerateName is used.
					Name:       "wext-" + webhook.Name + "-",
					APIVersion: "v1alpha1",
				},
			},
			Template: v1alpha1.EventListenerTemplate{
				Name:       webhook.Pipeline + "-template",
				APIVersion: "v1alpha1",
			},
			Interceptors: []*v1alpha1.EventInterceptor{
				{
					Webhook: &v1alpha1.WebhookInterceptor{
						Header: []pipelinesv1alpha1.Param{
							{Name: "Wext-Trigger-Name", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: webhook.Name + "-" + webhook.Namespace + "-pullrequest-event"}},
							{Name: "Wext-Repository-Url", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: webhook.GitRepositoryURL}},
							{Name: "Wext-Incoming-Event", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: "pull_request, Merge Request Hook"}},
							{Name: "Wext-Secret-Name", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: webhook.AccessTokenRef}},
							{Name: "Wext-Incoming-Actions", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: "opened,reopened,synchronize"}}},
						ObjectRef: &corev1.ObjectReference{
							APIVersion: "v1",
							Kind:       "Service",
							Name:       "tekton-webhooks-extension-validator",
							Namespace:  r.Defaults.Namespace,
						},
					},
				},
			},
		},
	}

	return triggers
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

func FakeGetTriggerBindingObjectMeta(name string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name: "wext-" + name + "-",
	}
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

	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      hook.AccessTokenRef,
			Namespace: installNs,
		},
		Data: map[string][]byte{
			"accessToken": []byte("access"),
			"secretToken": []byte("secret"),
		},
	}

	_, err := r.TriggersClient.TriggersV1alpha1().TriggerTemplates(installNs).Create(&template)
	if err != nil {
		fmt.Printf("Error creating fake triggertemplate %s", template.Name)
	}
	_, err = r.TriggersClient.TriggersV1alpha1().TriggerBindings(installNs).Create(&pushBinding)
	if err != nil {
		fmt.Printf("Error creating fake triggerbinding %s", pushBinding.Name)
	}
	_, err = r.TriggersClient.TriggersV1alpha1().TriggerBindings(installNs).Create(&pullBinding)
	if err != nil {
		fmt.Printf("Error creating fake triggerbinding %s", pullBinding.Name)
	}
	_, err = r.K8sClient.CoreV1().Secrets(installNs).Create(&secret)
	if err != nil {
		fmt.Printf("Error creating fake secret %s", secret.Name)
	}

	return

}

func Test_getWebhookSecretTokens(t *testing.T) {
	// Access token is stored as 'accessToken' and secret as 'secretToken'
	tests := []struct {
		name            string
		secret          *corev1.Secret
		wantAccessToken string
		wantSecretToken string
	}{
		{
			name: "foo",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
				},
				Data: map[string][]byte{
					"accessToken": []byte("myAccessToken"),
					"secretToken": []byte("mySecretToken"),
				},
			},
			wantAccessToken: "myAccessToken",
			wantSecretToken: "mySecretToken",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup resources
			r := dummyResource()
			if _, err := r.K8sClient.CoreV1().Secrets(r.Defaults.Namespace).Create(tt.secret); err != nil {
				t.Errorf("getWebhookSecretTokens() error creating secret: %s", err)
			}
			// Test
			gotAccessToken, gotSecretToken, err := utils.GetWebhookSecretTokens(r.K8sClient, r.Defaults.Namespace, tt.name)
			if err != nil {
				t.Errorf("getWebhookSecretTokens() returned an error: %s", err)
			}
			if tt.wantAccessToken != gotAccessToken {
				t.Errorf("getWebhookSecretTokens() accessToken = %s, want %s", gotAccessToken, tt.wantAccessToken)
			}
			if tt.wantSecretToken != gotSecretToken {
				t.Errorf("getWebhookSecretTokens() secretToken = %s, want %s", gotSecretToken, tt.wantSecretToken)
			}
		})
	}
}

func Test_getWebhookSecretTokens_error(t *testing.T) {
	// Access token is stored as 'accessToken' and secret as 'secretToken'
	tests := []struct {
		name   string
		secret *corev1.Secret
	}{
		{
			name: "namenotfound",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
				},
				Data: map[string][]byte{
					"accessToken": []byte("myAccessToken"),
					"secretToken": []byte("mySecretToken"),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup resources
			r := dummyResource()
			if _, err := r.K8sClient.CoreV1().Secrets(r.Defaults.Namespace).Create(tt.secret); err != nil {
				t.Errorf("getWebhookSecretTokens() error creating secret: %s", err)
			}
			// Test
			if _, _, err := utils.GetWebhookSecretTokens(r.K8sClient, r.Defaults.Namespace, tt.name); err == nil {
				t.Errorf("getWebhookSecretTokens() did not return an error when expected")
			}
		})
	}
}

func Test_createOAuth2Client(t *testing.T) {
	// Create client
	accessToken := "foo"
	ctx := context.Background()
	client := utils.CreateOAuth2Client(ctx, accessToken, true)
	// Test
	responseText := "my response"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if !strings.Contains(authHeader, accessToken) {
			t.Errorf("createOAuth2Client() expected authHeader to contain: %s; authHeader is: %s", accessToken, authHeader)
		}
		_, err := w.Write([]byte(responseText))
		if err != nil {
			t.Errorf("createOAuth2Client() error writing response: %s", err)
		}
	}))
	defer ts.Close()
	resp, err := client.Get(ts.URL)
	if err != nil {
		t.Logf("createOAuth2Client() error sending request: %s", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Logf("createOAuth2Client() error reading response body")
	}
	if string(body) != responseText {
		t.Logf("createOAuth2Client() expected response text %s; got: %s", responseText, body)
	}
}

func Test_createOpenshiftRoute(t *testing.T) {
	tests := []struct {
		name        string
		serviceName string
		route       *routesv1.Route
		hasErr      bool
	}{
		{
			name:        "OpenShift Route",
			serviceName: "route",
			route: &routesv1.Route{
				ObjectMeta: metav1.ObjectMeta{
					Name: "route",
					// Namepace in the dummy resource
					Namespace:   "default",
					Annotations: map[string]string{"haproxy.router.openshift.io/timeout": "2m"},
				},
				Spec: routesv1.RouteSpec{
					To: routesv1.RouteTargetReference{
						Kind: "Service",
						Name: "route",
					},
					TLS: &routesv1.TLSConfig{
						Termination:                   "edge",
						InsecureEdgeTerminationPolicy: "Redirect",
					},
				},
			},
			hasErr: false,
		},
	}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			r := dummyResource()
			var hasErr bool
			if err := r.createOpenshiftRoute(tests[i].serviceName); err != nil {
				hasErr = true
			}
			if diff := cmp.Diff(tests[i].hasErr, hasErr); diff != "" {
				t.Fatalf("Error mismatch (-want +got):\n%s", diff)
			}
			route, err := r.RoutesClient.RouteV1().Routes(r.Defaults.Namespace).Get(tests[i].serviceName, metav1.GetOptions{})
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tests[i].route, route); diff != "" {
				t.Errorf("Route mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_deleteOpenshiftRoute(t *testing.T) {
	tests := []struct {
		name      string
		routeName string
		hasErr    bool
	}{
		{
			name:      "OpenShift Route",
			routeName: "route",
			hasErr:    false,
		},
	}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			r := dummyResource()
			// Seed route for deletion
			route := &routesv1.Route{
				ObjectMeta: metav1.ObjectMeta{
					Name: tests[i].routeName,
				},
			}
			if _, err := r.RoutesClient.RouteV1().Routes(r.Defaults.Namespace).Create(route); err != nil {
				t.Fatal(err)
			}
			// Delete
			var hasErr bool
			if err := r.deleteOpenshiftRoute(tests[i].routeName); err != nil {
				hasErr = true
			}
			if diff := cmp.Diff(tests[i].hasErr, hasErr); diff != "" {
				t.Fatalf("Error mismatch (-want +got):\n%s", diff)
			}
			_, err := r.RoutesClient.RouteV1().Routes(r.Defaults.Namespace).Get(tests[i].routeName, metav1.GetOptions{})
			if err == nil {
				t.Errorf("Route not expected")
			}
		})
	}
}

func TestCreateDeleteIngress(t *testing.T) {
	r := dummyResource()
	r.Defaults.CallbackURL = "http://wibble.com"
	expectedHost := "wibble.com"

	err := r.createDeleteIngress("create", r.Defaults.Namespace)
	if err != nil {
		t.Errorf("error creating ingress: %s", err.Error())
	}

	ingress, err := r.K8sClient.ExtensionsV1beta1().Ingresses(r.Defaults.Namespace).Get("el-tekton-webhooks-eventlistener", metav1.GetOptions{})
	if err != nil {
		t.Errorf("error getting ingress: %s", err.Error())
	}

	if ingress.Spec.Rules[0].Host != expectedHost {
		t.Error("ingress Host did not match the callback URL")
	}

	err = r.createDeleteIngress("delete", r.Defaults.Namespace)
	if err != nil {
		t.Errorf("error deleting ingress: %s", err.Error())
	}
}
