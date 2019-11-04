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
	"errors"
	"fmt"
	restful "github.com/emicklei/go-restful"
	"github.com/mitchellh/mapstructure"
	pipelinesv1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	fakeclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/fake"
	faketriggerclientset "github.com/tektoncd/triggers/pkg/client/clientset/versioned/fake"
	"io"
	runtime "k8s.io/apimachinery/pkg/runtime"
	fakek8sclientset "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	"knative.dev/pkg/apis"
	"net/http"
)

func dummyK8sClientset() *fakek8sclientset.Clientset {
	result := fakek8sclientset.NewSimpleClientset()
	return result
}

func dummyClientset() *fakeclientset.Clientset {
	resultClient := fakeclientset.NewSimpleClientset()

	// Need to intercept the taskrun creation as "GeneratedName" does not generate a name in the fake clients and
	// we cannot therefore obtain the taskrun with a get call.  Here we store the taskrun into a map and set
	// the name of the taskrun to tekton-int, where int is the index in the map.
	taskruns := map[string]pipelinesv1alpha1.TaskRun{}
	resultClient.Fake.PrependReactor("create", "taskruns", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		// get the runtime.Object
		obj := action.(k8stesting.CreateAction).GetObject()
		// convert to unstructured map[string]interface{}
		unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
		if err != nil {
			newErr := fmt.Errorf("ERROR : Problem with taskrun creation in shared-test-funcs.go")
			return false, nil, newErr
		}
		// convert to TaskRun
		var result pipelinesv1alpha1.TaskRun
		mapstructure.Decode(unstructuredObj, &result)
		// store to end of map
		index := len(taskruns) + 1
		result.Name = fmt.Sprintf("tekton-%d", index)
		// HARD CODING TO SUCCESSFUL TASKRUN
		condition := &apis.Condition{
			Type:   apis.ConditionSucceeded,
			Status: "True",
		}
		result.Status.SetCondition(condition)
		taskruns[result.Name] = result

		//In case debug needed, you might want to uncomment this
		//fmt.Printf("taskruns: %+v", taskruns)
		return true, &result, nil
	})

	// Need to intercept calls to get githubsource, so that we can return the githubsource from the map we created
	// in the above code.
	resultClient.Fake.PrependReactor("get", "taskruns", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		srcName := action.(k8stesting.GetAction).GetName()
		if tr, found := taskruns[srcName]; found {
			return true, &tr, nil
		}

		return true, &pipelinesv1alpha1.TaskRun{}, errors.New("taskrun not found")
	})

	return resultClient
}

func dummyTriggersClientset() *faketriggerclientset.Clientset {
	result := faketriggerclientset.NewSimpleClientset()
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
		Defaults:       dummyDefaults(),
	}

	return &resource
}
