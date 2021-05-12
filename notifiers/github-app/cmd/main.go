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
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/google/go-github/v32/github"
	githubcontroller "github.com/tektoncd/experimental/notifiers/github-app/pkg/controller"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline"
	"github.com/tektoncd/pipeline/pkg/client/clientset/versioned/typed/pipeline/v1beta1"
	taskruninformer "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1beta1/taskrun"
	"golang.org/x/oauth2"
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
	_ "knative.dev/pkg/system/testing"
)

var (
	namespace = flag.String("namespace", corev1.NamespaceAll, "Namespace to restrict informer to. Optional, defaults to all namespaces.")
)

func main() {
	flag.Parse()

	ctx := sharedmain.WithHADisabled(signals.NewContext())
	github, err := githubClient(ctx)
	if err != nil {
		log.Fatalf("error creating github client: %v", err)
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

	sharedmain.MainWithContext(injection.WithNamespaceScope(ctx, *namespace), "watcher", func(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
		logger := logging.FromContext(ctx)
		taskRunInformer := taskruninformer.Get(ctx)

		c := &githubcontroller.GitHubAppReconciler{
			Logger:        logger,
			TaskRunLister: taskRunInformer.Lister(),
			GitHub:        github,
			Kubernetes:    k8s,
			Tekton:        tekton,
		}
		impl := controller.NewImpl(c, c.Logger, pipeline.TaskRunControllerName)

		taskRunInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc:    impl.Enqueue,
			UpdateFunc: controller.PassNew(impl.Enqueue),
		})

		return impl
	})
}

func githubClient(ctx context.Context) (*githubcontroller.GitHubClientFactory, error) {
	if id := os.Getenv("GITHUB_APP_ID"); id != "" {
		appID, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("error parsing ${GITHUB_APP_ID} (%s): %v", os.Getenv("GITHUB_APP_ID"), err)
		}
		return githubcontroller.NewApp(http.DefaultTransport, appID, os.Getenv("GITHUB_APP_KEY"))
	} else {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN")},
		)
		tc := oauth2.NewClient(ctx, ts)
		return githubcontroller.NewStatic(github.NewClient(tc)), nil
	}
}
