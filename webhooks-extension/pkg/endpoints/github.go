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
	github "github.com/google/go-github/github"
	logging "github.com/tektoncd/experimental/webhooks-extension/pkg/logging"
	utils "github.com/tektoncd/experimental/webhooks-extension/pkg/utils"
	"net/url"
	"os"
)

type GitHub struct {
	Client    *github.Client
	Context   context.Context
	Org       string
	Repo      string
	SSLVerify bool
	Resource  Resource
}

type GitHubWebhook struct {
	Hook *github.Hook
}

// GitHub GitProvider ----------------------------------------------------------------------------------------------------
func (r Resource) initGitHub(sslVerify bool, apiURL, secret, org, repo string) (*GitHub, error) {
	// Access token is stored as 'accessToken' and secret as 'secretToken'
	accessToken, _, err := utils.GetWebhookSecretTokens(r.K8sClient, r.Defaults.Namespace, secret)
	if err != nil {
		return nil, err
	}

	// Create the client
	ctx := context.Background()
	tc := utils.CreateOAuth2Client(ctx, accessToken)
	client := github.NewClient(tc)

	// Set api base url
	ghURL, err := url.Parse(apiURL)
	if err != nil {
		return nil, err
	}
	client.BaseURL = ghURL

	return &GitHub{Client: client, Context: ctx, Org: org, Repo: repo, SSLVerify: sslVerify, Resource: r}, nil
}

func (gh GitHub) AddWebhook(hook webhook) error {
	_, secretToken, err := utils.GetWebhookSecretTokens(gh.Resource.K8sClient, gh.Resource.Defaults.Namespace, hook.AccessTokenRef)
	if err != nil {
		return err
	}
	ssl := 0
	if !gh.SSLVerify {
		ssl = 1
	}

	// Specify webhook options
	cfg := make(map[string]interface{})
	cfg["url"] = os.Getenv("WEBHOOK_CALLBACK_URL")
	cfg["insecure_ssl"] = ssl
	cfg["secret"] = secretToken
	cfg["content_type"] = "json"
	events := []string{"push", "pull_request"}
	active := true
	hookDefinition := &github.Hook{
		Config: cfg,
		Events: events,
		Active: &active,
	}
	// Create webhook
	_, _, err = gh.Client.Repositories.CreateHook(gh.Context, gh.Org, gh.Repo, hookDefinition)
	return err
}

func (gh GitHub) DeleteWebhook(hook GitWebhook) error {
	_, err := gh.Client.Repositories.DeleteHook(gh.Context, gh.Org, gh.Repo, int64(hook.GetID()))
	return err
}

func (gh GitHub) GetAllWebhooks() ([]GitWebhook, error) {
	hooks, _, err := gh.Client.Repositories.ListHooks(gh.Context, gh.Org, gh.Repo, nil)
	if err != nil {
		return nil, err
	}
	webhooks := make([]GitWebhook, len(hooks))
	for i, hook := range hooks {
		webhooks[i] = GitHubWebhook{Hook: hook}
	}
	return webhooks, err
}

func (ghWebhook GitHubWebhook) GetID() int {
	return int(ghWebhook.Hook.GetID())
}

func (ghWebhook GitHubWebhook) GetURL() string {
	url, ok := ghWebhook.Hook.Config["url"].(string)
	if !ok {
		logging.Log.Error("webhook does not have string config 'url.' Setting webhook url to empty string.")
		url = ""
	}
	return url
}
