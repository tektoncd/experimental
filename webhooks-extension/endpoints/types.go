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
	eventsrcclientset "github.com/knative/eventing-sources/pkg/client/clientset/versioned"
	logging "github.com/tektoncd/experimental/webhooks-extension/pkg/logging"
	tektoncdclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	k8sclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"os"
)

// Resource stores all types here that are reused throughout files
type Resource struct {
	EventSrcClient eventsrcclientset.Interface
	TektonClient   tektoncdclientset.Interface
	K8sClient      k8sclientset.Interface
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

	// Setup event source client
	eventSrcClient, err := eventsrcclientset.NewForConfig(config)
	if err != nil {
		logging.Log.Errorf("error building event source client: %s.", err.Error())
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

	defaults := EnvDefaults{
		Namespace:      os.Getenv("INSTALLED_NAMESPACE"),
		DockerRegistry: os.Getenv("DOCKER_REGISTRY_LOCATION"),
	}

	r := Resource{
		K8sClient:      k8sClient,
		TektonClient:   tektonClient,
		EventSrcClient: eventSrcClient,
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
}

// ConfigMapName ... the name of the ConfigMap to create
const ConfigMapName = "githubwebhook"

//
type EnvDefaults struct {
	Namespace      string `json:"namespace"`
	DockerRegistry string `json:"dockerregistry"`
}
