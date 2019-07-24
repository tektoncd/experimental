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

	"errors"
	"fmt"
	restful "github.com/emicklei/go-restful"
	githubsource "github.com/knative/eventing-sources/pkg/apis/sources/v1alpha1"
	eventsrcclient "github.com/knative/eventing-sources/pkg/client/clientset/versioned/fake"
	"github.com/mitchellh/mapstructure"
	fakeclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/fake"
	runtime "k8s.io/apimachinery/pkg/runtime"
	fakek8sclientset "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func dummyK8sClientset() *fakek8sclientset.Clientset {
	result := fakek8sclientset.NewSimpleClientset()
	return result
}

func dummyClientset() *fakeclientset.Clientset {
	result := fakeclientset.NewSimpleClientset()
	return result
}

func dummyEventSrcClient() *eventsrcclient.Clientset {
	resultClient := eventsrcclient.NewSimpleClientset()
	sources := map[string]githubsource.GitHubSource{}

	// Need to intercept the githubsource creation as "GeneratedName" does not generate a name in the fake clients and
	// we cannot therefore obtain the githubsource with a get call.  Here we store the githubsource into a map and set
	// the name of the githubsource to tekton-int, where int is the index in the map.
	resultClient.Fake.PrependReactor("create", "githubsources", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		// get the runtime.Object
		obj := action.(k8stesting.CreateAction).GetObject()
		// convert to unstructured map[string]interface{}
		unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
		if err != nil {
			newErr := fmt.Errorf("ERROR : Problem with githubsource creation in shared-test-funcs.go")
			return false, nil, newErr
		}
		// convert to GitHubSource
		var result githubsource.GitHubSource
		mapstructure.Decode(unstructuredObj, &result)
		// store to end of map
		index := len(sources) + 1
		result.Name = fmt.Sprintf("tekton-%d", index)
		sources[result.Name] = result

		//In case debug needed, you might want to uncomment this
		//fmt.Printf("Sources: %+v", sources)
		return true, &result, nil
	})

	// Need to intercept calls to delete githubsource, so that we can return the githubsource from the map we created
	// in the above code.
	resultClient.Fake.PrependReactor("delete", "githubsources", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		srcName := action.(k8stesting.DeleteAction).GetName()
		delete(sources, srcName)
		result := sources[srcName]

		return true, &result, nil
	})

	// Need to intercept calls to get githubsource, so that we can return the githubsource from the map we created
	// in the above code.
	resultClient.Fake.PrependReactor("get", "githubsources", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		srcName := action.(k8stesting.GetAction).GetName()
		if ghs, found := sources[srcName]; found {
			return true, &ghs, nil
		}

		return true, &githubsource.GitHubSource{}, errors.New("githubsource not found")
	})

	// Need to intercept calls to list githubsource, so that we can list the githubsources from the map we created
	// in the above code.
	resultClient.Fake.PrependReactor("list", "githubsources", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		sourceArray := []githubsource.GitHubSource{}
		for _, source := range sources {
			sourceArray = append(sourceArray, source)
		}
		list := githubsource.GitHubSourceList{Items: sourceArray}
		return true, &list, nil
	})

	return resultClient
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
		EventSrcClient: r.EventSrcClient,
		Defaults:       newDefaults,
	}
	return &newResource
}

func dummyResource() *Resource {
	resource := Resource{
		K8sClient:      dummyK8sClientset(),
		TektonClient:   dummyClientset(),
		EventSrcClient: dummyEventSrcClient(),
		Defaults:       dummyDefaults(),
	}

	return &resource
}
