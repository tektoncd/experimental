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
	"fmt"
	"net/http"
	"net/url"

	logging "github.com/tektoncd/experimental/webhooks-extension/pkg/logging"
	"golang.org/x/xerrors"
)

// isGitHubEnterprise returns whether the url is for GitHub Enterprise or not
func isGitHubEnterprise(u *url.URL) bool {
	return (u.Host != "github.com")
}

// getGitHubHubbubAPI returns the API URL for the GitHub PubSubHubbub API
func getGitHubHubbubAPI(u *url.URL) string {
	// Public GitHub PubSubHubbub API URL is "https://api.github.com/hub"
	hubbubAPI := "https://api.github.com/hub"

	// Enterprise GitHub API URL is "https://my.company.xyz/api/v3/hub"
	if isGitHubEnterprise(u) {
		hubbubAPI = fmt.Sprintf("%s://%s/api/v3/hub", u.Scheme, u.Host)
	}

	return hubbubAPI
}

// doGitHubHubbubRequest executes a GitHub PubSubHubbub request given the specified parameters
// GitHub PubSubHubbub API documentation: https://developer.github.com/v3/repos/hooks/#pubsubhubbub
// mode: "subscribe" or "unsubscribe"
// callback: the URI to receive the updates
// secret: shared secret key to authenticate event messages
// events: the list of events to subscribe to or unsubscribe from; for example, {"push", "pull_request"}
func doGitHubHubbubRequest(client *http.Client, repoURL, mode, callback, secret string, events []string) error {
	// Get GitHub PubSubHubbub API URL
	u, err := url.Parse(repoURL)
	if err != nil {
		return xerrors.Errorf("error parsing GitHub repo URL %s. Error was: %w", repoURL, err)
	}
	hubbubAPI := getGitHubHubbubAPI(u)

	// Send request for each event type (example event type: "push")
	for _, event := range events {
		resp, err := client.PostForm(hubbubAPI, url.Values{
			"hub.mode":     {mode},
			"hub.topic":    {fmt.Sprintf("%s/events/%s", repoURL, event)},
			"hub.callback": {callback},
			"hub.secret":   {secret},
		})
		if err != nil {
			return xerrors.Errorf("error sending PubSubHubbub %s (%s) request: %w", mode, event, err)
		}
		// Should receive 204 No Content on success
		if resp.StatusCode != http.StatusNoContent {
			return xerrors.Errorf("error sending PubSubHubbub %s (%s) request. Status: %s", mode, event, resp.Status)
		}
		logging.Log.Debugf("PubSubHubbub %s (%s) response: %s", mode, event, resp)
	}
	return nil
}
