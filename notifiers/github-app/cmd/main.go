/*
Copyright 2020 The Tekton Authors

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
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/google/go-github/v32/github"
	githubcontroller "github.com/tektoncd/experimental/notifiers/github-app/pkg/controller"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline"
	"github.com/tektoncd/pipeline/pkg/client/clientset/versioned/typed/pipeline/v1beta1"
	taskruninformer "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1beta1/taskrun"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/injection"
	"knative.dev/pkg/injection/sharedmain"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/signals"

	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

var (
	namespace  = flag.String("namespace", corev1.NamespaceAll, "Namespace to restrict informer to. Optional, defaults to all namespaces.")
	kubeconfig = flag.String("kubeconfig", filepath.Join(os.Getenv("HOME"), ".kube", "config"), "Location of kubeconfig. If not set, InCluster config is assumed.")
)

func main() {
	flag.Parse()

	appID, err := strconv.ParseInt(os.Getenv("GITHUB_APP_ID"), 10, 64)
	if err != nil {
		log.Fatalf("error parsing ${GITHUB_APP_ID} (%s): %v", os.Getenv("GITHUB_APP_ID"), err)
	}
	at, err := ghinstallation.NewAppsTransportKeyFromFile(http.DefaultTransport, appID, os.Getenv("GITHUB_APP_KEY"))
	if err != nil {
		log.Fatalf("error reading GitHub App private key: %v", err)
	}

	config, err := clientcmd.BuildConfigFromFlags("", filepath.Join(os.Getenv("HOME"), ".kube", "config"))
	if err != nil {
		log.Fatalf("error getting kubernetes client config: %v", err)
	}

	// create the clientset
	k8s, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("error creating kubernetes client: %v", err)
	}
	tekton, err := v1beta1.NewForConfig(config)
	if err != nil {
		log.Fatalf("error creating tekton client: %v", err)
	}

	sharedmain.MainWithContext(injection.WithNamespaceScope(signals.NewContext(), *namespace), "watcher", func(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
		logger := logging.FromContext(ctx)
		taskRunInformer := taskruninformer.Get(ctx)

		c := &githubcontroller.GitHubAppReconciler{
			Logger:        logger,
			TaskRunLister: taskRunInformer.Lister(),
			InstallationClient: func(installationID int64) *github.Client {
				return github.NewClient(&http.Client{Transport: ghinstallation.NewFromAppsTransport(at, installationID)})
			},
			Kubernetes: k8s,
			Tekton:     tekton,
		}
		impl := controller.NewImpl(c, c.Logger, pipeline.TaskRunControllerName)

		taskRunInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc:    impl.Enqueue,
			UpdateFunc: controller.PassNew(impl.Enqueue),
		})

		return impl
	})
}
