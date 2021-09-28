/*
Copyright 2021 The Tekton Authors

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
	"os"

	"github.com/tektoncd/experimental/remote-resolution/pkg/reconciler/framework"
	"github.com/tektoncd/experimental/remote-resolution/pkg/reconciler/pipelineruns"
	"github.com/tektoncd/experimental/remote-resolution/pkg/reconciler/resourcerequest"
	"github.com/tektoncd/experimental/remote-resolution/pkg/resolvers/clusterref"
	"github.com/tektoncd/experimental/remote-resolution/pkg/resolvers/gitref"
	"github.com/tektoncd/experimental/remote-resolution/pkg/resolvers/noopref"

	// This defines the shared main for injected controllers.
	"knative.dev/pkg/injection/sharedmain"
)

func main() {
	rmode := pipelineruns.NewResolutionMode(os.Getenv("RESOLUTION_MODE"))
	sharedmain.Main("controller",
		resourcerequest.NewController,
		pipelineruns.NewPipelineRunResolverController(rmode),
		framework.NewController(&gitref.Resolver{}, "8081"),
		framework.NewController(&clusterref.Resolver{}, "8082"),
		framework.NewController(&noopref.Resolver{}, "8083"),
	)
}
