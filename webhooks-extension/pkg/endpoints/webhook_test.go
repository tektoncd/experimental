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
	"net/http"
	"net/http/httptest"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"

	restful "github.com/emicklei/go-restful"
	v1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var server *httptest.Server

// All event sources will be created in the "default" namespace because the INSTALLED_NAMESPACE env variable is not set
const installNs = "default"
const defaultRegistry = "default.docker.reg:8500/foo"

func setUpServer() *Resource {
	wsContainer := restful.NewContainer()
	resource := dummyResource()
	resource.K8sClient.CoreV1().Namespaces().Create(&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: installNs}})
	server = httptest.NewServer(wsContainer)
	resource.RegisterExtensionWebService(wsContainer)
	return resource
}

// Test createGitHubSource
func TestGitHubSource(t *testing.T) {
	r := dummyResource()

	sources := []webhook{
		{
			Name:             "name1",
			Namespace:        installNs,
			GitRepositoryURL: "https://github.com/owner/repo",
			AccessTokenRef:   "token1",
			Pipeline:         "pipeline1",
			DockerRegistry:   "registry1",
			HelmSecret:       "helmsecret1",
			ReleaseName:      "releasename1",
		},
		{
			Name:             "name2",
			Namespace:        installNs,
			GitRepositoryURL: "https://github.company.com/owner2/repo2.git",
			AccessTokenRef:   "token2",
			Pipeline:         "pipeline2",
			DockerRegistry:   "registry2",
			HelmSecret:       "helmsecret2",
			ReleaseName:      "releasename2",
		},
		{
			Name:             "name3",
			Namespace:        installNs,
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
		DockerRegistry: defaultRegistry,
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

	// Check the second entry matches in getGitHubWebhook i.e .git URL matches without git
	testGetGitHubWebhook(sources[1].GitRepositoryURL, r, t)
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
	httpReq := dummyHTTPRequest("GET", "http://wwww.dummy.com:8080/webhooks/defaults", nil)
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
	httpReq := dummyHTTPRequest("POST", "http://wwww.dummy.com:8080/webhooks/", bytes.NewBuffer(b))
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
		t.Errorf("Incorrect OwnerAndRepository, expected %s but was: %s", expectedOwnerAndRepo, realOwnerAndRepo)
	}
	realGitHubAPIURL := ghSrc.Spec.GitHubAPIURL
	if expectedGitHubAPIURL != realGitHubAPIURL {
		t.Errorf("Incorrect GitHubAPIURL, expected %s but was: %s", expectedGitHubAPIURL, realGitHubAPIURL)
	}
}

// Check the webhooks against the GET all webhooks
func testGetAllWebhooks(expectedWebhooks []webhook, r *Resource, t *testing.T) {
	httpReq := dummyHTTPRequest("GET", "http://wwww.dummy.com:8080/webhooks/", nil)
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
			expectedWebhooks[i].DockerRegistry = defaultRegistry
		}
		expected[expectedWebhooks[i]] = true
		actual[actualWebhooks[i]] = true
	}

	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Webhook error: expected: \n%v \nbut received \n%v", expected, actual)
	}
}

// Checks that URLs without .git match a .git URL in the configmap
func testGetGitHubWebhook(gitURL string, r *Resource, t *testing.T) {

	configMapClient := r.K8sClient.CoreV1().ConfigMaps(installNs)
	_, err := configMapClient.Get(ConfigMapName, metav1.GetOptions{})
	if err != nil {
		t.Errorf("Uh oh, we got an error looking up the ConfigMap for webhooks. Error is: %s", err)
	}

	if strings.HasSuffix(gitURL, ".git") {
		blank := webhook{}
		// Trim the suffix - as this is how it comes from the actual webhook
		// r.getGitHubWebhook should still match and return the webhook from the configmap
		whookReturned, err := r.getGitHubWebhook(strings.TrimSuffix(gitURL, ".git"), installNs)
		if err != nil {
			t.Errorf("Error occurred in getGitHubWebhook: %s", err)
		} else {
			if whookReturned == blank {
				t.Errorf("Weirdly an empty webhook was returned without an error from getGitHubWebhook")
				t.Fail()
			}
			t.Logf("Webhook found in configmap: %s", whookReturned)
		}
	} else {
		// The source needs to end in .git as this is what we need to test - and what would have been
		// stored in the configmap
		t.Error("Uh oh, someone has changed the repo URL - it needs to end in .git")
	}

}

func TestDeleteByNameNoName405(t *testing.T) {
	setUpServer()
	httpReq, _ := http.NewRequest(http.MethodDelete, server.URL+"/webhooks/?namespace=foo", nil)
	response, _ := http.DefaultClient.Do(httpReq)
	if response.StatusCode != 405 {
		t.Errorf("Status code not set to 400 when deleting without a name, it's: %d", response.StatusCode)
	}
}

func TestDeleteByNameNoNamespaceBadRequest(t *testing.T) {
	setUpServer()
	httpReq, _ := http.NewRequest(http.MethodDelete, server.URL+"/webhooks/foo", nil)
	response, _ := http.DefaultClient.Do(httpReq)
	if response.StatusCode != 400 {
		t.Errorf("Status code not set to 400 when deleting without a namespace, it's: %d", response.StatusCode)
	}
}

// Verify it's gone, including from ConfigMap, and no PipelineRuns were deleted for that repo as wasn't specified
func TestDeleteByNameKeepRuns(t *testing.T) {
	t.Log("In testDeleteByNameKeepRuns")

	r := setUpServer()

	labels := make(map[string]string)
	labels["gitServer"] = "github.com"
	labels["gitOrg"] = "owner"
	labels["gitRepo"] = "repo"

	// This should get left around
	pipelineRun := v1alpha1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "imustlinger",
			Labels: labels,
		},
		Spec: v1alpha1.PipelineRunSpec{},
	}

	_, err := r.TektonClient.TektonV1alpha1().PipelineRuns(installNs).Create(&pipelineRun)
	if err != nil {
		t.Errorf("Error creating the PipelineRun: %s", err)
	}

	theWebhook := webhook{
		Name:             "webhooktodelete",
		Namespace:        installNs,
		GitRepositoryURL: "https://github.com/owner/repo",
		AccessTokenRef:   "token1",
		Pipeline:         "pipeline1",
		HelmSecret:       "helmsecret1",
		ReleaseName:      "foo",
	}

	configMapClient := r.K8sClient.CoreV1().ConfigMaps(installNs)

	resp := createWebhook(theWebhook, r)

	if resp.StatusCode() != http.StatusCreated {
		t.Error("Didn't create the webhook OK for the deletion test")
	} else {
		_, err := configMapClient.Get(ConfigMapName, metav1.GetOptions{})
		if err != nil {
			t.Errorf("Uh oh, we got an error looking up the ConfigMap for webhooks to check if our entry was removed from it. Error is: %s", err)
		}
	}

	httpReq, _ := http.NewRequest(http.MethodDelete, server.URL+"/webhooks/"+theWebhook.Name+"?namespace="+installNs, nil)

	response, _ := http.DefaultClient.Do(httpReq)

	if response.StatusCode != 204 {
		t.Errorf("Status code not set to 204 when deleting, it's: %d", response.StatusCode)
	}
	_, err = r.EventSrcClient.SourcesV1alpha1().GitHubSources(installNs).Get(theWebhook.Name, metav1.GetOptions{})

	// We get an error if it can't be found. There's no error here, so it still exists - so fail
	if err == nil {
		t.Error("GitHub source should have been deleted, wasn't")
	}

	// We get an error if it can't be found. There's no error here, so it still exists - so fail
	_, err = r.TektonClient.TektonV1alpha1().PipelineRuns(installNs).Get(pipelineRun.Name, metav1.GetOptions{})

	if err != nil {
		t.Errorf("PipelineRun %s should not have been deleted, it was", pipelineRun.Name)
	}

	// Now check it's gone from the ConfigMap

	configMapToCheck, err := configMapClient.Get(ConfigMapName, metav1.GetOptions{})
	if err != nil {
		t.Errorf("Uh oh, we got an error looking up the ConfigMap for webhooks to check if our entry was removed from it. Error is: %s", err)
		t.Fail()
	}

	if configMapToCheck == nil {
		t.Errorf("Uh oh, we didn't find the ConfigMap named %s in namespace %s", ConfigMapName, installNs)
		t.Fail()
	}

	contents := string(configMapToCheck.BinaryData["GitHubSource"])

	if contents == "" {
		t.Errorf("Uh oh, nothing in the ConfigMap for GitHubSource, failing")
		t.Fail()
	}

	if strings.Contains(contents, "webhooktodelete") || strings.Contains(contents, "https://github.com/owner/repo") {
		t.Errorf("Found the webhook name or the repository URL in the ConfigMap data when it should have been removed, data is: %s", contents)
	}
}

// Verify it's gone, including from the ConfigMap, and PipelineRuns were deleted for that repository
func TestDeleteByNameDeleteRuns(t *testing.T) {
	t.Logf("In testDeleteByNameDeleteRuns")

	r := setUpServer()

	labelsForPipelineRun1 := make(map[string]string)
	labelsForPipelineRun1["gitServer"] = "github.com"
	labelsForPipelineRun1["gitOrg"] = "foobar"
	labelsForPipelineRun1["gitRepo"] = "barfoo"

	pipelineRun1 := v1alpha1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "imustlinger1",
			Labels: labelsForPipelineRun1,
		},
		Spec: v1alpha1.PipelineRunSpec{},
	}

	_, err := r.TektonClient.TektonV1alpha1().PipelineRuns(installNs).Create(&pipelineRun1)
	if err != nil {
		t.Errorf("Error creating the first PipelineRun: %s", err)
	}

	labelsForPipelineRun2 := make(map[string]string)
	labelsForPipelineRun2["gitServer"] = "funkygithub.com"
	labelsForPipelineRun2["gitOrg"] = "iamgettingdeleted"
	labelsForPipelineRun2["gitRepo"] = "barfoobar"

	pipelineRun2 := v1alpha1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "imustgo",
			Labels: labelsForPipelineRun2,
		},
		Spec: v1alpha1.PipelineRunSpec{},
	}

	_, err = r.TektonClient.TektonV1alpha1().PipelineRuns(installNs).Create(&pipelineRun2)
	if err != nil {
		t.Errorf("Error creating the second PipelineRun: %s", err)
	}

	labelsForPipelineRun3 := make(map[string]string)
	labelsForPipelineRun3["gitServer"] = "funkygithub.com"
	labelsForPipelineRun3["gitOrg"] = "iamstaying"
	labelsForPipelineRun3["gitRepo"] = "barfoobar"

	pipelineRun3 := v1alpha1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "imustlinger2",
			Labels: labelsForPipelineRun3,
		},
		Spec: v1alpha1.PipelineRunSpec{},
	}

	_, err = r.TektonClient.TektonV1alpha1().PipelineRuns(installNs).Create(&pipelineRun3)
	if err != nil {
		t.Errorf("Error creating the third PipelineRun: %s", err)
	}

	// Maps to the second PipelineRun which must be deleted
	theWebhook := webhook{
		Name:             "DeleteByNameDeleteRunsHook",
		Namespace:        installNs,
		GitRepositoryURL: "https://funkygithub.com/iamgettingdeleted/barfoobar", // Same repo URL as pipelineRun2
		AccessTokenRef:   "token1",
		Pipeline:         "pipeline1",
		HelmSecret:       "helmsecret1",
		ReleaseName:      "foo",
	}

	resp := createWebhook(theWebhook, r)

	if resp.StatusCode() != http.StatusCreated {
		t.Error("Didn't create the webhook OK for the deletion test")
		t.Fail()
	}

	httpReq, _ := http.NewRequest(http.MethodDelete, server.URL+"/webhooks/"+theWebhook.Name+"?namespace="+installNs+"&deletepipelineruns=true", nil)

	response, _ := http.DefaultClient.Do(httpReq)

	if response.StatusCode != 204 {
		t.Error("Status code not set to 204 when deleting")
	}

	_, err = r.EventSrcClient.SourcesV1alpha1().GitHubSources(installNs).Get(theWebhook.Name, metav1.GetOptions{})

	if err == nil {
		t.Errorf("The GitHub source %s should have been deleted, wasn't", theWebhook.Name)
	}

	// This Get should fail because it's been deleted, so we want an error

	_, err = r.TektonClient.TektonV1alpha1().PipelineRuns(installNs).Get(pipelineRun2.Name, metav1.GetOptions{})

	// This Get should not give us an error because PipelineRun2 still exists

	if err == nil {
		t.Errorf("The second PipelineRun should have been deleted, it wasn't!")
	}

	// This Get should not give us an error because PipelineRun2 still exists

	_, err = r.TektonClient.TektonV1alpha1().PipelineRuns(installNs).Get(pipelineRun1.Name, metav1.GetOptions{})

	if err != nil {
		theRepoFromLabels := fmt.Sprintf("%s/%s/%s", pipelineRun1.Labels["gitServer"], pipelineRun1.Labels["gitOrg"], pipelineRun1.Labels["gitRepo"])
		t.Errorf("The first PipelineRun %s for repository %s should not have been deleted", pipelineRun1.Name, theRepoFromLabels)
	}

	// This Get should not give us an error because PipelineRun3 still exists

	_, err = r.TektonClient.TektonV1alpha1().PipelineRuns(installNs).Get(pipelineRun3.Name, metav1.GetOptions{})

	if err != nil {
		theRepoFromLabels := fmt.Sprintf("%s/%s/%s", pipelineRun3.Labels["gitServer"], pipelineRun3.Labels["gitOrg"], pipelineRun3.Labels["gitRepo"])
		t.Errorf("The third PipelineRun %s for repository %s should not have been deleted", pipelineRun3.Name, theRepoFromLabels)

		allPipelineRuns, err := r.TektonClient.TektonV1alpha1().PipelineRuns(installNs).List(metav1.ListOptions{})
		if err != nil {
			t.Errorf("Couldn't even get a list of PipelineRuns, error is: %s", err)
		}
		t.Logf("Found PipelineRuns: %v", allPipelineRuns.Items)
	}

	// Now check it's gone from the ConfigMap

	configMapClient := r.K8sClient.CoreV1().ConfigMaps(installNs)

	configMap, err := configMapClient.Get(ConfigMapName, metav1.GetOptions{})
	if err != nil {
		t.Error("Uh oh, we got an error looking up the ConfigMap for webhooks to check if our entry was removed from it")
	}

	contents := string(configMap.BinaryData["GitHubSource"])
	if strings.Contains(contents, "DeleteByNameDeleteRunsHook") || strings.Contains(contents, "https://funkygithub.com/iamgettingdeleted/barfoobar") {
		t.Errorf("Found the webhook name or the repository URL in the ConfigMap data when it should have been removed, data is: %s", contents)
	}
}

func TestDeleteByName404NotFound(t *testing.T) {
	t.Log("In TestDeleteByName404NotFound")
	httpReq, _ := http.NewRequest(http.MethodDelete, server.URL+"/webhooks/idonotexist?namespace="+installNs, nil)
	response, _ := http.DefaultClient.Do(httpReq)
	if response.StatusCode != 404 {
		t.Errorf("Status code not set to 404 when deleting a non-existent webhook, it's: %d", response.StatusCode)
	}
}

func testGithubSourceReleaseNameTooLong(r *Resource, t *testing.T) {

	data := webhook{
		Name:             "name1",
		Namespace:        installNs,
		GitRepositoryURL: "https://github.com/owner/repo",
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

func TestMultipleDeletesCorrectData(t *testing.T) {
	t.Log("in TestMultipleDeletesCorrectData")
	r := setUpServer()

	numTimes := 20
	runtime.GOMAXPROCS(2)

	for i := 0; i < numTimes; i++ {
		theWebhook1 := webhook{
			Name:             "routine1hook-" + strconv.Itoa(i),
			Namespace:        installNs,
			GitRepositoryURL: "https://a.com/b/c", // Same repo URL as pipelineRun
			AccessTokenRef:   "token1",
			Pipeline:         "pipeline1",
		}

		resp := createWebhook(theWebhook1, r)

		if resp.StatusCode() != http.StatusCreated {
			t.Error("Didn't create the first webhook OK for the safe multiple request deletion test")
			t.Fail()
		}

		theWebhook2 := webhook{
			Name:             "routine2hook-" + strconv.Itoa(i),
			Namespace:        installNs,
			GitRepositoryURL: "https://b.com/c/d", // Same repo URL as pipelineRun
			AccessTokenRef:   "token1",
			Pipeline:         "pipeline1",
		}

		resp = createWebhook(theWebhook2, r)

		if resp.StatusCode() != http.StatusCreated {
			t.Error("Didn't create the second webhook OK for the safe multiple request deletion test")
			t.Fail()
		}

		// Fire them off at the same time, then check the resulting ConfigMap is accurate: containing no entries and not just one.

		var firstResponse *http.Response
		var secondResponse *http.Response
		firstReq, _ := http.NewRequest(http.MethodDelete, server.URL+"/webhooks/"+theWebhook1.Name+"?namespace="+installNs, nil)
		secondReq, _ := http.NewRequest(http.MethodDelete, server.URL+"/webhooks/"+theWebhook2.Name+"?namespace="+installNs, nil)

		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			firstResponse, _ = http.DefaultClient.Do(firstReq)
		}()

		go func() {
			defer wg.Done()
			secondResponse, _ = http.DefaultClient.Do(secondReq)
		}()

		wg.Wait()

		/*
			firstResponse, _ = http.DefaultClient.Do(firstReq)
			secondResponse, _ = http.DefaultClient.Do(secondReq)
		*/

		if firstResponse.StatusCode != 204 {
			t.Errorf("Should have deleted the first webhook OK, return code wasn't 204 - it's: %d", firstResponse.StatusCode)
		}

		if secondResponse.StatusCode != 204 {
			t.Errorf("Should have deleted the second webhook OK, return code wasn't 204 - it's: %d", secondResponse.StatusCode)
		}

		//	Check both are gone from the ConfigMap
		configMapClient := r.K8sClient.CoreV1().ConfigMaps(installNs)

		configMap, err := configMapClient.Get(ConfigMapName, metav1.GetOptions{})
		if err != nil {
			t.Error("Uh oh, we got an error looking up the ConfigMap for webhooks to check if our entry was removed from it")
		}

		contents := string(configMap.BinaryData["GitHubSource"])
		if strings.Contains(contents, theWebhook1.Name) || strings.Contains(contents, "https://a.com/b/c") ||
			strings.Contains(contents, theWebhook2.Name) || strings.Contains(contents, "https://b.com/c/d") {
			t.Errorf("For iteration %d, we found a webhook name or repository URL in "+
				"the ConfigMap data when both should have been removed through simultaneous deletion requests, data is: %s", i, contents)
		}

		t.Logf("Iteration %d complete", i)
	}
	t.Log("Test complete")
}

/* This test has also been seen to crash or fail under Prow
func TestMultipleCreatesCorrectData(t *testing.T) {
	t.Log("in TestMultipleCreatesCorrectData")
	r := setUpServer()

	numTimes := 100
	runtime.GOMAXPROCS(2)

	// This test is super noisy because of writeGitHubWebhooks having a logging.Log.Debugf
	os.Stdout, _ = os.Open(os.DevNull)

	for i := 0; i < numTimes; i++ {
		theWebhook1 := webhook{
			Name:             fmt.Sprintf("routine1createhook-%d", i),
			Namespace:        installNs,
			GitRepositoryURL: "https://a.com/b/c",
			AccessTokenRef:   "token1",
			Pipeline:         "pipeline1",
		}

		theWebhook2 := webhook{
			Name:             fmt.Sprintf("routine2createhook-%d", i),
			Namespace:        installNs,
			GitRepositoryURL: "https://b.com/c/d",
			AccessTokenRef:   "token1",
			Pipeline:         "pipeline1",
		}

		var firstResponse *restful.Response
		var secondResponse *restful.Response

		// Fire them off at the same time, then check the resulting ConfigMap is accurate: containing both entries and not just one.

		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			firstResponse = createWebhook(theWebhook1, r)
		}()

		go func() {
			defer wg.Done()
			secondResponse = createWebhook(theWebhook2, r)
		}()

		wg.Wait()

		if firstResponse.StatusCode() != http.StatusCreated {
			t.Errorf("Iteration %d, didn't create the first webhook OK for the safe multiple request creation test, response: %d", i, firstResponse.StatusCode())
			t.Fail()
		}

		if secondResponse.StatusCode() != http.StatusCreated {
			t.Errorf("Iteration %d, didn't create the second webhook OK for the safe multiple request creation test, response: %d", i, secondResponse.StatusCode())
			t.Fail()
		}

		//	Check both are present in the ConfigMap

		configMapClient := r.K8sClient.CoreV1().ConfigMaps(installNs)

		configMap, err := configMapClient.Get(ConfigMapName, metav1.GetOptions{})
		if err != nil {
			t.Error("Uh oh, we got an error looking up the ConfigMap for webhooks to check if our entries were added to it")
		}

		contents := string(configMap.BinaryData["GitHubSource"])
		if !!!strings.Contains(contents, theWebhook1.Name) || !!!strings.Contains(contents, "https://a.com/b/c") ||
			!!!strings.Contains(contents, theWebhook2.Name) || !!!strings.Contains(contents, "https://b.com/c/d") {
			t.Errorf("For iteration %d, we we did not find a webhook name and repository URL in "+
				"the ConfigMap data when both should have been added through simultaneous creation requests, data is: %s", i, contents)
		}
	}

}
*/

/* This third test method is also failing under Prow
func TestCreateDeleteCorrectData(t *testing.T) {
	t.Log("in TestCreateDeleteCorrectData")
	r := setUpServer()

	numTimes := 100
	runtime.GOMAXPROCS(2)

	// This test is super noisy because of writeGitHubWebhooks having a logging.Log.Debugf
	os.Stdout, _ = os.Open(os.DevNull)

	for i := 0; i < numTimes; i++ {
		theWebhook1 := webhook{
			Name:             fmt.Sprintf("routine1createhook-%d", i),
			Namespace:        installNs,
			GitRepositoryURL: "https://a.com/b/c",
			AccessTokenRef:   "token1",
			Pipeline:         "pipeline1",
		}

		theWebhook2 := webhook{
			Name:             fmt.Sprintf("routine2createhook-%d", i),
			Namespace:        installNs,
			GitRepositoryURL: "https://b.com/c/d",
			AccessTokenRef:   "token1",
			Pipeline:         "pipeline1",
		}

		var firstResponse *restful.Response
		var secondResponse *restful.Response
		var deleteResponse *http.Response

		deleteRequest, _ := http.NewRequest(http.MethodDelete, server.URL+"/webhooks/"+theWebhook1.Name+"?namespace="+installNs, nil)

		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			firstResponse = createWebhook(theWebhook1, r)
			deleteResponse, _ = http.DefaultClient.Do(deleteRequest)
		}()

		go func() {
			defer wg.Done()
			secondResponse = createWebhook(theWebhook2, r)
		}()

		wg.Wait()

		if firstResponse.StatusCode() != http.StatusCreated {
			t.Errorf("Iteration %d, didn't create the first webhook OK for the safe multiple request creation and deletion test, response: %d", i, firstResponse.StatusCode())
			t.Fail()
		}

		if secondResponse.StatusCode() != http.StatusCreated {
			t.Errorf("Iteration %d, didn't create the second webhook OK for the safe multiple request creation and deletion test, response: %d", i, secondResponse.StatusCode())
			t.Fail()
		}

		if deleteResponse.StatusCode != http.StatusNoContent {
			t.Errorf("Iteration %d, didn't delete the first webhook OK for the safe multiple request creation and deletion test, response: %d", i, deleteResponse.StatusCode)
			t.Fail()
		}

		//	Check just the second entry is present in the ConfigMap

		configMapClient := r.K8sClient.CoreV1().ConfigMaps(installNs)

		configMap, err := configMapClient.Get(ConfigMapName, metav1.GetOptions{})
		if err != nil {
			t.Error("Uh oh, we got an error looking up the ConfigMap for webhooks to check if our entry was present")
		}

		contents := string(configMap.BinaryData["GitHubSource"])

		if strings.Contains(contents, theWebhook1.Name) {
			t.Errorf("For iteration %d we found %s when it should have been deleted", i, theWebhook1.Name)
		}

		if !!!strings.Contains(contents, theWebhook2.Name) {
			t.Errorf("For iteration %d we did not find %s. It should not have been deleted", i, theWebhook2.Name)
		}
	}
}
*/
