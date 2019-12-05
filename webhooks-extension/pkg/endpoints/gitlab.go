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
	utils "github.com/tektoncd/experimental/webhooks-extension/pkg/utils"
	"github.com/xanzy/go-gitlab"
	"os"
)

type GitLabWebhook struct {
	Hook *gitlab.ProjectHook
}

type GitLab struct {
	Client    *gitlab.Client
	ProjectID string
	SSLVerify bool
	Resource  Resource
}

func (r Resource) initGitLab(sslVerify bool, apiURL, secret, org, repo string) (*GitLab, error) {
	// Access token is stored as 'accessToken' and secret as 'secretToken'
	accessToken, _, err := utils.GetWebhookSecretTokens(r.K8sClient, r.Defaults.Namespace, secret)
	if err != nil {
		return nil, err
	}

	// Create the client
	var glClient *gitlab.Client
	if sslVerify {
		glClient = gitlab.NewClient(nil, accessToken)
	} else {
		glClient = gitlab.NewClient(utils.GetClientAllowsSelfSigned(), accessToken)
	}
	glClient.SetBaseURL(apiURL)

	return &GitLab{Client: glClient, ProjectID: org + "/" + repo, SSLVerify: sslVerify, Resource: r}, nil
}

func (gl GitLab) GetAllWebhooks() ([]GitWebhook, error) {
	hooks, _, err := gl.Client.Projects.ListProjectHooks(gl.ProjectID, &gitlab.ListProjectHooksOptions{}, nil)
	if err != nil {
		return nil, err
	}
	webhooks := make([]GitWebhook, len(hooks))
	for i, hook := range hooks {
		webhooks[i] = GitLabWebhook{Hook: hook}
	}
	return webhooks, err
}

func (gl GitLab) AddWebhook(hook webhook) error {
	// Specify webhook options
	callback := os.Getenv("WEBHOOK_CALLBACK_URL")
	pushEvents := true
	mergeEvents := true
	tagPushEvents := true
	sslverify := gl.SSLVerify
	_, secretToken, err := utils.GetWebhookSecretTokens(gl.Resource.K8sClient, gl.Resource.Defaults.Namespace, hook.AccessTokenRef)
	if err != nil {
		return err
	}

	webhookOptions := gitlab.AddProjectHookOptions{
		URL:                   &callback,
		PushEvents:            &pushEvents,
		MergeRequestsEvents:   &mergeEvents,
		TagPushEvents:         &tagPushEvents,
		EnableSSLVerification: &sslverify,
		Token:                 &secretToken,
	}
	// Add webhook
	_, _, err = gl.Client.Projects.AddProjectHook(gl.ProjectID, &webhookOptions)
	return err
}

func (gl GitLab) DeleteWebhook(hook GitWebhook) error {
	_, err := gl.Client.Projects.DeleteProjectHook(gl.ProjectID, hook.GetID())
	return err
}

// GitLab Webhook --------------------------------------------------------------------------------------------------------
func (glWebhook GitLabWebhook) GetID() int {
	return glWebhook.Hook.ID
}

func (glWebhook GitLabWebhook) GetURL() string {
	return glWebhook.Hook.URL
}
