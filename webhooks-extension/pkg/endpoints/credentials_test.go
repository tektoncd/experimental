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
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestCreateBadAccessToken(t *testing.T) {
	r := dummyResource()
	badAccessToken := credential{
		Name: "badToken",
	}
	expectedError := fmt.Sprintf("error: AccessToken must be specified")
	createAndCheckCredential(badAccessToken, expectedError, r, t)

	// Verify no credentials have been created
	checkCredentials([]credential{}, "", r, t)
}

func TestAccessTokenWithSecret(t *testing.T) {
	r := dummyResource()

	accessTokenWithSecret := credential{
		Name:        "accesstoken-with-secret",
		AccessToken: "alongstringofcharacters",
		SecretToken: "thisIsMySecretToken",
	}
	createAndCheckCredential(accessTokenWithSecret, "", r, t)
}

// Should be "default" which is r.dummyResource.namespace's value

func TestAccessTokenWithNoNamespaceUsesDefault(t *testing.T) {
	r := dummyResource()

	accessTokenNoNamespace := credential{
		Name:        "accesstoken",
		AccessToken: "anotherlongstringofcharacters",
		SecretToken: "thisIsMySecretToken",
	}

	jsonBody, _ := json.Marshal(accessTokenNoNamespace)
	t.Logf("json body for create cred test with no namespace: %s", jsonBody)
	httpReq := dummyHTTPRequest("POST", "http://wwww.dummy.com:8383/webhooks/credentials", bytes.NewBuffer(jsonBody))
	req := dummyRestfulRequest(httpReq, "")
	httpWriter := httptest.NewRecorder()
	resp := dummyRestfulResponse(httpWriter)
	r.createCredential(req, resp)

	httpReq = dummyHTTPRequest("GET", "http://wwww.dummy.com:8383/webhooks/credentials", bytes.NewBuffer(nil))
	req = dummyRestfulRequest(httpReq, "")
	httpWriter = httptest.NewRecorder()
	resp = dummyRestfulResponse(httpWriter)
	r.getAllCredentials(req, resp)

	result := []credential{}
	b, err := ioutil.ReadAll(httpWriter.Body)
	if err != nil {
		t.Fatalf("FAIL: ERROR - reading response body: %s", err.Error())
	}
	err = json.Unmarshal(b, &result)

	t.Logf("unmarshalled result '%+v'", result)

	if result[0].Name != accessTokenNoNamespace.Name {
		t.Fatalf("Result came back with name %s but expected %s", result[0].Name, accessTokenNoNamespace.Name)
	}

	// Finally check that result has a SecretToken set and it's starred out
	if result[0].SecretToken != "********" {
		t.Fatalf("Result came back with something other than eight chars for the secret token: '%s'", result[0].SecretToken)
	}

}

func TetAccessTokenWithNoSecret(t *testing.T) {
	r := dummyResource()

	accessTokenNoSecret := credential{
		Name:        "accesstoken-nosecret",
		AccessToken: "anotherlongstringofcharacters",
	}

	jsonBody, _ := json.Marshal(accessTokenNoSecret)
	t.Logf("json body for create cred test: %s", jsonBody)
	httpReq := dummyHTTPRequest("POST", "http://wwww.dummy.com:8383/webhooks/credentials", bytes.NewBuffer(jsonBody))
	req := dummyRestfulRequest(httpReq, "")
	httpWriter := httptest.NewRecorder()
	resp := dummyRestfulResponse(httpWriter)
	r.createCredential(req, resp)

	httpReq = dummyHTTPRequest("GET", "http://wwww.dummy.com:8383/webhooks/credentials", bytes.NewBuffer(nil))
	req = dummyRestfulRequest(httpReq, "")
	httpWriter = httptest.NewRecorder()
	resp = dummyRestfulResponse(httpWriter)
	r.getAllCredentials(req, resp)

	result := []credential{}
	b, err := ioutil.ReadAll(httpWriter.Body)
	if err != nil {
		t.Fatalf("FAIL: ERROR - reading response body: %s", err.Error())
	}
	err = json.Unmarshal(b, &result)

	t.Logf("unmarshalled result '%+v'", result)

	if result[0].Name != accessTokenNoSecret.Name {
		t.Fatalf("Result came back with name %s but expected %s", result[0].Name, accessTokenNoSecret.Name)
	}
	// Finally check that result has a SecretToken set
	if strings.Count(result[0].SecretToken, "") < 20 {
		t.Fatalf("Result came back with less than twenty chars of secret token: '%s'", result[0].SecretToken)
	}

}

func TestDeleteCredential(t *testing.T) {
	r := dummyResource()
	accessTokenToDelete := credential{
		Name:        "accesstokenToDelete",
		AccessToken: "sdkfhighregusfbliusbbbwhfwiehw8hwefhw938hf497fhw97b47",
		SecretToken: "provideASecretTokenSoThatCreateAndCheckCredCanDoDeepEquals",
	}
	createAndCheckCredential(accessTokenToDelete, "", r, t)

	httpReq := dummyHTTPRequest("DELETE", "http://wwww.dummy.com:8383/webhooks/credentials", bytes.NewBuffer(nil))

	req := dummyRestfulRequest(httpReq, "accesstokenToDelete")
	httpWriter := httptest.NewRecorder()
	resp := dummyRestfulResponse(httpWriter)
	r.deleteCredential(req, resp)

	creds := r.getK8sCredentials()
	if len(creds) > 0 {
		t.Fatalf("Namespace %s should contain no credentials, but we found %+v", r.Defaults.Namespace, creds)
	}
}

func TestDeleteACredentialThatDoesNotExist(t *testing.T) {
	r := dummyResource()
	httpReq := dummyHTTPRequest("DELETE", "http://wwww.dummy.com:8383/webhooks/credentials", bytes.NewBuffer(nil))
	req := dummyRestfulRequest(httpReq, "accesstokenToDelete")
	httpWriter := httptest.NewRecorder()
	resp := dummyRestfulResponse(httpWriter)
	r.deleteCredential(req, resp)
	if resp.StatusCode() != http.StatusNotFound {
		t.Fatalf("Expected 404 deleting non-existent credential but got %d", resp.StatusCode())
	}
}

//----------------------------------------
// end of Tests. Helper functions below.
//----------------------------------------

// SecretTokens are twenty characters randomly selected from a set of sixty one. 61^20=5.08e35. We should 'never' get the same token twice.
func TestRandomStringGenerator(t *testing.T) {
	tokens := make(map[string]bool)
	for i := 0; i < 100; i++ {
		token := string(getRandomSecretToken())
		if tokens[token] == true {
			t.Fatalf("Generated the same token twice in less than a hundred tries! map=%+v", tokens)
		}
		tokens[token] = true
	}
}

func createAndCheckCredential(cred credential, expectError string, r *Resource, t *testing.T) {
	t.Logf("CREATE credential %+v", cred)

	// Create dummy rest api request and response
	jsonBody, _ := json.Marshal(cred)
	t.Logf("json body for create cred test: %s", jsonBody)
	url := "http://wwww.dummy.com:8383/webhooks/credentials"
	httpReq := dummyHTTPRequest("POST", url, bytes.NewBuffer(jsonBody))
	req := dummyRestfulRequest(httpReq, "")
	httpWriter := httptest.NewRecorder()
	resp := dummyRestfulResponse(httpWriter)
	r.createCredential(req, resp)

	// Sometimes we expect an error here, in which case no credential will have been created.
	if expectError != "" {
		resultCred := credential{}
		checkResponseError(httpWriter.Body, &resultCred, expectError, t)
	} else {
		compareCredentials(r.getK8sCredential(cred.Name), cred, t)
	}
}

func (r Resource) getK8sCredentials() (credentials []credential) {
	secrets, err := r.K8sClient.CoreV1().Secrets(r.Defaults.Namespace).List(metav1.ListOptions{})
	if err != nil {
		return
	}
	for _, secret := range secrets.Items {
		credentials = append(credentials, secretToCredential(&secret, false))
	}
	return credentials
}

func (r Resource) getK8sCredential(name string) (credential credential) {
	secret, err := r.K8sClient.CoreV1().Secrets(r.Defaults.Namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return
	}
	return secretToCredential(secret, false)
}

func compareCredentials(resultCred credential, expectCred credential, t *testing.T) {
	t.Logf("Result cred: %+v\n", resultCred)
	t.Logf("Expect cred: %+v\n", expectCred)
	if !reflect.DeepEqual(resultCred, expectCred) {
		t.Fatal("Credentials not equal")
	}
}

func checkResponseError(body *bytes.Buffer, result interface{}, expectError string, t *testing.T) bool {
	b, err := ioutil.ReadAll(body)
	if err != nil {
		t.Fatalf("FAIL: ERROR - reading response body: %s", err.Error())
		return false
	}
	err = json.Unmarshal(b, result)
	if err != nil {
		fmt.Printf("checkResponseError jsonUnmarshal got err '%s' expected '%s' got '%s'", err, expectError, string(b))
		resultError := string(b)
		if !strings.HasPrefix(resultError, expectError) {
			t.Fatalf("FAIL: ERROR - Error message == '%s', want '%s', body: %s", resultError, expectError, body)
		}
		return false
	}
	if expectError != "" {
		t.Fatalf("FAIL: ERROR - Result == %+v, want error message '%s'", result, expectError)
	}
	return true
}

func checkCredentials(expectCreds []credential, expectError string, r *Resource, t *testing.T) {
	t.Logf("READ all credentials. Expecting: %+v", expectCreds)
	// Create dummy REST API request and response
	httpReq := dummyHTTPRequest("GET", "http://wwww.dummy.com:8383/v1/webhooks/credentials", nil)
	req := dummyRestfulRequest(httpReq, "")
	httpWriter := httptest.NewRecorder()
	resp := dummyRestfulResponse(httpWriter)

	// Test to get all credentials
	r.getAllCredentials(req, resp)

	// Look for password "********"
	accessTokens := []string{}
	for i, cred := range expectCreds {
		accessToken := cred.AccessToken
		accessTokens = append(accessTokens, accessToken)
		expectCreds[i].AccessToken = "********"
	}
	// Look for secret token "********"
	secretTokens := []string{}
	for i, cred := range expectCreds {
		secretToken := cred.SecretToken
		secretTokens = append(secretTokens, secretToken)
		expectCreds[i].SecretToken = "********"
	}
	// Read result
	resultCreds := []credential{}
	if !checkResponseError(httpWriter.Body, &resultCreds, expectError, t) {
		return
	}
	testCredentials(resultCreds, expectCreds, t)
	for i := range expectCreds {
		expectCreds[i].AccessToken = accessTokens[i]
	}

	// Verify against K8s client
	testCredentials(r.getK8sCredentials(), expectCreds, t)
	t.Logf("Done in READ all credentials. Expecting: %+v", expectCreds)
}

// Compares two lists of credentials
func testCredentials(result []credential, expectResult []credential, t *testing.T) {
	if len(result) != len(expectResult) {
		t.Fatalf("ERROR: Result == %+v, want %+v, different number of credentials", len(result), len(expectResult))
	}
	for i := range expectResult {
		compareCredentials(result[i], expectResult[i], t)
	}
}
