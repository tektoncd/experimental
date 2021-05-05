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

package catalogtask

import (
	context "context"
	"fmt"
	"io/ioutil"
	"os"

	catalog "github.com/tektoncd/experimental/catalogtask/pkg/catalog"
	pipelineclient "github.com/tektoncd/pipeline/pkg/client/injection/client"
	run "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1alpha1/run"
	taskruninformer "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1beta1/taskrun"
	v1alpha1run "github.com/tektoncd/pipeline/pkg/client/injection/reconciler/pipeline/v1alpha1/run"
	tkncontroller "github.com/tektoncd/pipeline/pkg/controller"
	"k8s.io/client-go/tools/cache"
	configmap "knative.dev/pkg/configmap"
	controller "knative.dev/pkg/controller"
	logging "knative.dev/pkg/logging"
)

const (
	ControllerName = "catalogtask-controller"
	apiVersion     = "catalogtask.tekton.dev/v1alpha1"
	kind           = "Task"
)

var cachePath = os.Getenv("CACHE_PATH")
var catalogGitURL = os.Getenv("CATALOG_GIT_URL")

func NewController(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
	logger := logging.FromContext(ctx)

	runInformer := run.Get(ctx)
	taskRunInformer := taskruninformer.Get(ctx)

	scratchDir, err := ioutil.TempDir(cachePath, "catalog")
	if err != nil {
		logger.Errorf("Unable to create temp scratch dir: %v", err)
		return nil
	}

	if catalogGitURL == "" {
		catalogGitURL = "https://github.com/tektoncd/catalog.git"
	}

	debugLog("scratch dir: %s\ncatalog git url: %s", scratchDir, catalogGitURL)

	cat, err := catalog.New(catalogGitURL, "task", scratchDir)
	if err != nil {
		panic(fmt.Sprintf("error setting up catalog: %v", err))
	}

	reconciler := &Reconciler{
		catalog:           cat,
		pipelineClientSet: pipelineclient.Get(ctx),
		taskRunLister:     taskRunInformer.Lister(),
	}

	impl := v1alpha1run.NewImpl(ctx, reconciler, func(impl *controller.Impl) controller.Options {
		return controller.Options{
			AgentName: ControllerName,
		}
	})

	taskRunInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: controller.PassNew(impl.EnqueueControllerOf),
	})

	logger.Info("Setting up event handlers")

	runInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: tkncontroller.FilterRunRef(apiVersion, kind),
		Handler:    controller.HandleAll(impl.Enqueue),
	})

	return impl
}
