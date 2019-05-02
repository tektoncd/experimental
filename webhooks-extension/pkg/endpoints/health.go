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
	"net/http"

	restful "github.com/emicklei/go-restful"
)

func checkHealth(request *restful.Request, response *restful.Response) {
	response.WriteHeader(http.StatusNoContent)
}

// RegisterLivenessWebService registers the liveness web service
func (r Resource) RegisterLivenessWebService(container *restful.Container) {
	ws := new(restful.WebService)
	ws.Path("/liveness")
	ws.Route(ws.GET("").To(checkHealth))

	container.Add(ws)
}

// RegisterReadinessWebService registers the readiness web service
func (r Resource) RegisterReadinessWebService(container *restful.Container) {
	ws := new(restful.WebService)
	ws.Path("/readiness")
	ws.Route(ws.GET("").To(checkHealth))

	container.Add(ws)
}
