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
	restful "github.com/emicklei/go-restful"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const default_registry = "default.docker.reg:8500/foo"

// Test createGitHubSource
func TestGitHubSource(t *testing.T) {
	r := dummyResource()
	// All event sources will be created in the "default" namespace because the INSTALLED_NAMESPACE env variable is not set
	installNs := "default"
	runNs := "test"
	sources := []webhook{
		{
			Name:             "name1",
			Namespace:        runNs,
			GitRepositoryURL: "https://github.com/owner/repo.git",
			AccessTokenRef:   "token1",
			Pipeline:         "pipeline1",
			DockerRegistry:   "registry1",
			HelmSecret:       "helmsecret1",
			ReleaseName:      "releasename1",
		},
		{
			Name:             "name2",
			Namespace:        runNs,
			GitRepositoryURL: "https://github.company.com/owner2/repo2",
			AccessTokenRef:   "token2",
			Pipeline:         "pipeline2",
			DockerRegistry:   "registry2",
			HelmSecret:       "helmsecret2",
			ReleaseName:      "releasename2",
		},
		{
			Name:             "name3",
			Namespace:        runNs,
			GitRepositoryURL: "https://github.company.com/owner3/repo3",
			AccessTokenRef:   "token3",
			Pipeline:         "pipeline3",
			DockerRegistry:   "",
			HelmSecret:       "helmsecret3",
			ReleaseName:      "releasename3",
		},
	}

	testDockerRegUnset(r, t)

	// Set default docker registry for sources that specify no registry value
	newDefaults := EnvDefaults{
		Namespace:      installNs,
		DockerRegistry: default_registry,
	}
	r = updateResourceDefaults(r, newDefaults)

	testDockerRegSet(r, t)

	// Create the first entry
	createWebhook(sources[0], r)

	// Check the first entry (check with k8s)
	testGitHubSource(sources[0].Name, "owner/repo", "", installNs, r, t)

	// Check the first entry (check with GET all webhooks)
	testGetAllWebhooks(sources[:1], r, t)

	// Create the second entry which uses GHE
	createWebhook(sources[1], r)

	// Check the second entry (check with k8s)
	testGitHubSource(sources[1].Name, "owner2/repo2", "https://github.company.com/api/v3/", installNs, r, t)

	// Create the third entry source specifies no docker registry so should use env var
	createWebhook(sources[2], r)

	// Check all entries (check with GET all webhooks)
	testGetAllWebhooks(sources, r, t)

}

func testDockerRegUnset(r *Resource, t *testing.T) {
	// Get the docker registry using the endpoint, expect ""
	defaults := getEnvDefaults(r, t)
	reg := defaults.DockerRegistry
	if reg != "" {
		t.Errorf("Incorrect defaultDockerRegistry, expected \"\" but was: %s", reg)
	}
}

func testDockerRegSet(r *Resource, t *testing.T) {
	// Get the docker registry using the endpoint, expect "default.docker.reg:8500/foo"
	defaults := getEnvDefaults(r, t)
	reg := defaults.DockerRegistry
	if reg != "default.docker.reg:8500/foo" {
		t.Errorf("Incorrect defaultDockerRegistry, expected default.docker.reg:8500/foo, but was: %s", reg)
	}
}

func getEnvDefaults(r *Resource, t *testing.T) EnvDefaults {
	httpReq := dummyHTTPRequest("GET", "http://wwww.dummy.com:8080/webhook/defaults", nil)
	req := dummyRestfulRequest(httpReq, "", "")
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
	// Create the first entry
	b, _ := json.Marshal(webhook)
	httpReq := dummyHTTPRequest("POST", "http://wwww.dummy.com:8080/webhook/", bytes.NewBuffer(b))
	req := dummyRestfulRequest(httpReq, "", "")
	httpWriter := httptest.NewRecorder()
	resp := dummyRestfulResponse(httpWriter)
	r.createWebhook(req, resp)
	return resp
}

// Check a webhook's github source against k8s
func testGitHubSource(expectedName string, expectedOwnerAndRepo string, expectedGitHubAPIURL string, namespace string, r *Resource, t *testing.T) {
	ghSrc, err := r.EventSrcClient.SourcesV1alpha1().GitHubSources(namespace).Get(expectedName, metav1.GetOptions{})
	if err != nil {
		t.Errorf("GitHubSource %s was not found in namespace %s: %s", expectedName, namespace, err.Error())
	}
	realOwnerAndRepo := ghSrc.Spec.OwnerAndRepository
	if expectedOwnerAndRepo != realOwnerAndRepo {
		t.Errorf("Incorrect OwnderAndRepository, expected %s but was: %s", expectedOwnerAndRepo, realOwnerAndRepo)
	}
	realGitHubAPIURL := ghSrc.Spec.GitHubAPIURL
	if expectedGitHubAPIURL != realGitHubAPIURL {
		t.Errorf("Incorrect GitHubAPIURL, expected %s but was: %s", expectedGitHubAPIURL, realGitHubAPIURL)
	}
}

// Check the webhooks against the GET all webhooks
func testGetAllWebhooks(expectedWebhooks []webhook, r *Resource, t *testing.T) {
	httpReq := dummyHTTPRequest("GET", "http://wwww.dummy.com:8080/webhook/", nil)
	req := dummyRestfulRequest(httpReq, "", "")
	httpWriter := httptest.NewRecorder()
	resp := dummyRestfulResponse(httpWriter)
	r.getAllWebhooks(req, resp)
	actualWebhooks := []webhook{}
	err := json.NewDecoder(httpWriter.Body).Decode(&actualWebhooks)
	if err != nil {
		t.Errorf("Error decoding result into []webhook{}: %s", err.Error())
		return
	}
	if len(expectedWebhooks) != len(actualWebhooks) {
		t.Errorf("Incorrect length of result, expected %d, but was %d", len(expectedWebhooks), len(actualWebhooks))
		return
	}

	// Now compare the arrays expectedWebhooks and actualWebhooks by turning them into maps
	expected := map[webhook]bool{}
	actual := map[webhook]bool{}
	for i := range expectedWebhooks {
		if expectedWebhooks[i].DockerRegistry == "" {
			expectedWebhooks[i].DockerRegistry = default_registry
		}
		expected[expectedWebhooks[i]] = true
		actual[actualWebhooks[i]] = true
	}

	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Webhook error: expected: \n%v \nbut received \n%v", expected, actual)
	}
}

func testGithubSourceReleaseNameTooLong(r *Resource, t *testing.T) {
	runNs := "test"

	data := webhook{
		Name:             "name1",
		Namespace:        runNs,
		GitRepositoryURL: "https://github.com/owner/repo.git",
		AccessTokenRef:   "token1",
		Pipeline:         "pipeline1",
		HelmSecret:       "helmsecret1",
		ReleaseName:      "1234567891234567891234567891234567891234567891234567891234567890", // 0 brings us to 64 char
	}

	// Create the first entry
	resp := createWebhook(data, r)

	if resp.StatusCode() != http.StatusBadRequest {
		t.Error("Expected a bad request when the release name exceeded 63 chars")
	}
}
