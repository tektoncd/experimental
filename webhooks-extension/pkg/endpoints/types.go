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
	"os"

	routeclientset "github.com/openshift/client-go/route/clientset/versioned"
	logging "github.com/tektoncd/experimental/webhooks-extension/pkg/logging"
	tektoncdclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	triggersclientset "github.com/tektoncd/triggers/pkg/client/clientset/versioned"
	k8sclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Resource stores all types here that are reused throughout files
type Resource struct {
	TektonClient   tektoncdclientset.Interface
	K8sClient      k8sclientset.Interface
	TriggersClient triggersclientset.Interface
	RoutesClient   routeclientset.Interface
	Defaults       EnvDefaults
}

// NewResource returns a new Resource instantiated with its clientsets
func NewResource() (Resource, error) {
	// Get cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		logging.Log.Errorf("error getting in cluster config: %s.", err.Error())
		return Resource{}, err
	}

	// Setup tektoncd client
	tektonClient, err := tektoncdclientset.NewForConfig(config)
	if err != nil {
		logging.Log.Errorf("error building tekton clientset: %s.", err.Error())
		return Resource{}, err
	}

	// Setup k8s client
	k8sClient, err := k8sclientset.NewForConfig(config)
	if err != nil {
		logging.Log.Errorf("error building k8s clientset: %s.", err.Error())
		return Resource{}, err
	}

	// Setup triggers client
	triggersClient, err := triggersclientset.NewForConfig(config)
	if err != nil {
		logging.Log.Errorf("error building triggers clientset: %s.", err.Error())
		return Resource{}, err
	}
	// Currently Openshift does not have a top level client, but instead one for
	// each apiGroup
	routesClient, err := routeclientset.NewForConfig(config)
	if err != nil {
		logging.Log.Errorf("Error building routes clientset: %s.", err.Error())
		return Resource{}, err
	}

	defaults := EnvDefaults{
		Namespace:      os.Getenv("INSTALLED_NAMESPACE"),
		DockerRegistry: os.Getenv("DOCKER_REGISTRY_LOCATION"),
		CallbackURL:    os.Getenv("WEBHOOK_CALLBACK_URL"),
	}
	if defaults.Namespace == "" {
		// If no namespace provided, use "default"
		defaults.Namespace = "default"
	}

	r := Resource{
		K8sClient:      k8sClient,
		TektonClient:   tektonClient,
		TriggersClient: triggersClient,
		RoutesClient:   routesClient,
		Defaults:       defaults,
	}
	return r, nil
}

// Webhook stores the webhook information
type webhook struct {
	Name             string `json:"name"`
	Namespace        string `json:"namespace"`
	ServiceAccount   string `json:"serviceaccount,omitempty"`
	GitRepositoryURL string `json:"gitrepositoryurl"`
	AccessTokenRef   string `json:"accesstoken"`
	Pipeline         string `json:"pipeline"`
	DockerRegistry   string `json:"dockerregistry,omitempty"`
	HelmSecret       string `json:"helmsecret,omitempty"`
	ReleaseName      string `json:"releasename,omitempty"`
	PullTask         string `json:"pulltask,omitempty"`
	OnSuccessComment string `json:"onsuccesscomment,omitempty"`
	OnFailureComment string `json:"onfailurecomment,omitempty"`
	OnTimeoutComment string `json:"ontimeoutcomment,omitempty"`
	OnMissingComment string `json:"onmissingcomment,omitempty"`
}

// ConfigMapName ... the name of the ConfigMap to create
const ConfigMapName = "githubwebhook"

type EnvDefaults struct {
	Namespace      string `json:"namespace"`
	DockerRegistry string `json:"dockerregistry"`
	CallbackURL    string `json:"endpointurl"`
}
