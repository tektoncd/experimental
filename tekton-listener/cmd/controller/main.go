/*
Copyright 2018 The Knative Authors

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
	"flag"
	"log"
	"time"

	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/tektoncd/pipeline/pkg/logging"

	sharedclientset "github.com/knative/pkg/client/clientset/versioned"
	"github.com/knative/pkg/controller"

	"github.com/tektoncd/experimental/tekton-listener/pkg/reconciler"
	"github.com/tektoncd/experimental/tekton-listener/pkg/reconciler/eventbinding"
	"github.com/tektoncd/experimental/tekton-listener/pkg/reconciler/tektonlistener"
	"github.com/tektoncd/pipeline/pkg/reconciler/v1alpha1/pipelinerun"
	"github.com/tektoncd/pipeline/pkg/reconciler/v1alpha1/taskrun"
	"github.com/tektoncd/pipeline/pkg/system"

	"github.com/knative/pkg/configmap"
	"github.com/knative/pkg/signals"
	clientset "github.com/tektoncd/experimental/tekton-listener/pkg/client/clientset/versioned"
	experimentalinformers "github.com/tektoncd/experimental/tekton-listener/pkg/client/informers/externalversions"
)

const (
	threadsPerController = 2
	resyncPeriod         = 10 * time.Hour
)

var (
	masterURL  string
	kubeconfig string
)

func main() {
	flag.Parse()
	loggingConfigMap, err := configmap.Load("/etc/config-logging")
	if err != nil {
		log.Fatalf("Error loading logging configuration: %v", err)
	}
	loggingConfig, err := logging.NewConfigFromMap(loggingConfigMap)
	if err != nil {
		log.Fatalf("Error parsing logging configuration: %v", err)
	}
	logger, atomicLevel := logging.NewLoggerFromConfig(loggingConfig, logging.ControllerLogKey)
	defer logger.Sync()

	logger.Info("Starting the Experimental Controller (gasp)")

	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()

	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		logger.Fatalf("Error building kubeconfig: %v", err)
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		logger.Fatalf("Error building kubernetes clientset: %v", err)
	}

	sharedClient, err := sharedclientset.NewForConfig(cfg)
	if err != nil {
		logger.Fatalf("Error building shared clientset: %v", err)
	}

	pipelineClient, err := clientset.NewForConfig(cfg)
	if err != nil {
		logger.Fatalf("Error building experimental clientset: %v", err)
	}

	configMapWatcher := configmap.NewInformedWatcher(kubeClient, system.GetNamespace())

	opt := reconciler.Options{
		KubeClientSet:     kubeClient,
		SharedClientSet:   sharedClient,
		PipelineClientSet: pipelineClient,
		ConfigMapWatcher:  configMapWatcher,
		ResyncPeriod:      resyncPeriod,
		Logger:            logger,
	}

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, opt.ResyncPeriod)
	pipelineInformerFactory := pipelineinformers.NewSharedInformerFactory(pipelineClient, opt.ResyncPeriod)

	resourceInformer := pipelineInformerFactory.Tekton().V1alpha1().PipelineResources()
	podInformer := kubeInformerFactory.Core().V1().Pods()
	tektonListenerInformer := pipelineInformerFactory.Tekton().V1alpha1().TektonListeners()
	eventbindingInformer := pipelineInformerFactory.Tekton().V1alpha1().EventBindings()

	pipelineInformer := pipelineInformerFactory.Tekton().V1alpha1().Pipelines()
	pipelineRunInformer := pipelineInformerFactory.Tekton().V1alpha1().PipelineRuns()
	timeoutHandler := reconciler.NewTimeoutHandler(kubeClient, pipelineClient, stopCh, logger)

	lrc := tektonlistener.NewController(
		kubeClient,
		tektonListenerInformer,
	)
	ebc := eventbinding.NewController(
		opt,
		kubeClient,
		eventbindingInformer,
		tektonListenerInformer,
		pipelineInformer,
	)

	// Build all of our controllers, with the clients constructed above.
	controllers := []*controller.Impl{
		// Pipeline Controllers
		lrc,
		ebc,
	}
	timeoutHandler.SetTaskRunCallbackFunc(trc.Enqueue)
	timeoutHandler.SetPipelineRunCallbackFunc(prc.Enqueue)
	timeoutHandler.CheckTimeouts()

	// Watch the logging config map and dynamically update logging levels.
	configMapWatcher.Watch(logging.ConfigName, logging.UpdateLevelFromConfigMap(logger, atomicLevel, logging.ControllerLogKey))

	kubeInformerFactory.Start(stopCh)
	pipelineInformerFactory.Start(stopCh)
	if err := configMapWatcher.Start(stopCh); err != nil {
		logger.Fatalf("failed to start configuration manager: %v", err)
	}

	// Wait for the caches to be synced before starting controllers.
	logger.Info("Waiting for informer caches to sync")
	for i, synced := range []cache.InformerSynced{
		eventbindingInformer.Informer().HasSynced,
		tektonListenerInformer.Informer().HasSynced,
	} {
		if ok := cache.WaitForCacheSync(stopCh, synced); !ok {
			logger.Fatalf("failed to wait for cache at index %v to sync", i)
		}
	}

	logger.Info("Starting controllers")
	// Start all of the controllers.
	for _, ctrlr := range controllers {
		go func(ctrlr *controller.Impl) {
			// We don't expect this to return until stop is called,
			// but if it does, propagate it back.
			if runErr := ctrlr.Run(threadsPerController, stopCh); runErr != nil {
				logger.Fatalf("Error running controller: %v", runErr)
			}
		}(ctrlr)
	}

	<-stopCh
}

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
}
