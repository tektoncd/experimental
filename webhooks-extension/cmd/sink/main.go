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

package main

import (
	"net/http"
	"os"

	restful "github.com/emicklei/go-restful"
	"github.com/tektoncd/experimental/webhooks-extension/endpoints"
	logging "github.com/tektoncd/experimental/webhooks-extension/pkg/logging"
)

func main() {
	// Create/setup resource
	r, err := endpoints.NewResource()
	if err != nil {
		logging.Log.Fatalf("Fatal error creating resource: %s.", err.Error())
	}

	// Set up routes
	wsContainer := restful.NewContainer()
	// Add sink
	wsContainer.Add(endpoints.SinkWebService(r))
	// Add liveness/readiness
	wsContainer.Add(endpoints.LivenessWebService())
	wsContainer.Add(endpoints.ReadinessWebService())

	// Serve
	logging.Log.Info("Creating server and entering wait loop.")
	port := os.Getenv("PORT")
	if port == "" {
		logging.Log.Fatal("Knative runtime contract should specify PORT env via single container port in yaml")
	}
	server := &http.Server{Addr: ":"+port, Handler: wsContainer}
	logging.Log.Fatal(server.ListenAndServe())
}
