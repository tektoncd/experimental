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
	"io"
	"net/http"

	restful "github.com/emicklei/go-restful"
	fakeroutesclientset "github.com/openshift/client-go/route/clientset/versioned/fake"
	fakeclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/fake"
	faketriggerclientset "github.com/tektoncd/triggers/pkg/client/clientset/versioned/fake"
	fakek8sclientset "k8s.io/client-go/kubernetes/fake"
)

func dummyK8sClientset() *fakek8sclientset.Clientset {
	result := fakek8sclientset.NewSimpleClientset()
	return result
}

func dummyClientset() *fakeclientset.Clientset {
	resultClient := fakeclientset.NewSimpleClientset()
	return resultClient
}

func dummyTriggersClientset() *faketriggerclientset.Clientset {
	result := faketriggerclientset.NewSimpleClientset()
	return result
}

func dummyRoutesClientset() *fakeroutesclientset.Clientset {
	result := fakeroutesclientset.NewSimpleClientset()
	return result
}

func dummyHTTPRequest(method string, url string, body io.Reader) *http.Request {
	httpReq, _ := http.NewRequest(method, url, body)
	httpReq.Header.Set("Content-Type", "application/json")
	return httpReq
}

func dummyRestfulResponse(httpWriter http.ResponseWriter) *restful.Response {
	result := restful.NewResponse(httpWriter)
	result.SetRequestAccepts(restful.MIME_JSON)
	return result
}
func dummyRestfulRequest(httpReq *http.Request, name string) *restful.Request {
	restfulReq := restful.NewRequest(httpReq)
	params := restfulReq.PathParameters()
	if name != "" {
		params["name"] = name
	}
	return restfulReq
}

func dummyDefaults() EnvDefaults {
	initialValues := EnvDefaults{
		Namespace:      "default",
		DockerRegistry: "",
	}
	return initialValues
}

func updateResourceDefaults(r *Resource, newDefaults EnvDefaults) *Resource {
	newResource := Resource{
		K8sClient:      r.K8sClient,
		TektonClient:   r.TektonClient,
		TriggersClient: r.TriggersClient,
		Defaults:       newDefaults,
	}
	return &newResource
}

func dummyResource() *Resource {
	resource := Resource{
		K8sClient:      dummyK8sClientset(),
		TektonClient:   dummyClientset(),
		TriggersClient: dummyTriggersClientset(),
		RoutesClient:   dummyRoutesClientset(),
		Defaults:       dummyDefaults(),
	}

	return &resource
}
