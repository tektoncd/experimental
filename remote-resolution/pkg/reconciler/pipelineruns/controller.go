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

package pipelineruns

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	rrclient "github.com/tektoncd/experimental/remote-resolution/pkg/client/injection/client"
	rrinformer "github.com/tektoncd/experimental/remote-resolution/pkg/client/injection/informers/resolution/v1alpha1/resourcerequest"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	pipelineclient "github.com/tektoncd/pipeline/pkg/client/injection/client"
	pipelineruninformer "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1beta1/pipelinerun"
	clusterinterceptorinformer "github.com/tektoncd/triggers/pkg/client/injection/informers/triggers/v1alpha1/clusterinterceptor"
	"k8s.io/client-go/tools/cache"
	kubeclient "knative.dev/pkg/client/injection/kube/client"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
)

func NewPipelineRunResolverController(rmode ResolutionMode) func(context.Context, configmap.Watcher) *controller.Impl {
	return func(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
		logger := logging.FromContext(ctx)
		kubeclientset := kubeclient.Get(ctx)

		pipelineclientset := pipelineclient.Get(ctx)
		rrclientset := rrclient.Get(ctx)

		pipelinerunInformer := pipelineruninformer.Get(ctx)
		ciInformer := clusterinterceptorinformer.Get(ctx)

		rrInformer := rrinformer.Get(ctx)

		lister := pipelinerunInformer.Lister()
		cilister := ciInformer.Lister()

		r := &Reconciler{
			LeaderAwareFuncs: buildPipelineRunLeaderAwareFuncs(lister),

			kubeClientSet:            kubeclientset,
			pipelineClientSet:        pipelineclientset,
			pipelinerunLister:        pipelinerunInformer.Lister(),
			resourceRequestLister:    rrInformer.Lister(),
			resourceRequestClientSet: rrclientset,
			clusterInterceptorLister: cilister,

			mode: rmode,
		}

		ctrType := reflect.TypeOf(r).Elem()
		ctrTypeName := fmt.Sprintf("%s.%s", ctrType.PkgPath(), ctrType.Name())
		ctrTypeName = strings.ReplaceAll(ctrTypeName, "/", ".")

		impl := controller.NewContext(ctx, r, controller.ControllerOptions{
			WorkQueueName: ctrTypeName,
			Logger:        logger,
		})

		logger.Info("Attaching PipelineRun hooks")

		pipelinerunInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
			FilterFunc: filterPipelineRuns,
			Handler: cache.ResourceEventHandlerFuncs{
				AddFunc: impl.Enqueue,
				UpdateFunc: func(oldObj, newObj interface{}) {
					impl.Enqueue(newObj)
				},
				// TODO: do we need a deletefunc?
				// DeleteFunc: impl.Enqueue,
			},
		})

		logger.Info("Attaching ResourceRequest hooks")

		rrInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
			UpdateFunc: func(oldObj, newObj interface{}) {
				impl.EnqueueControllerOf(newObj)
			},
			DeleteFunc: impl.EnqueueControllerOf,
		})

		return impl
	}
}

func filterPipelineRuns(obj interface{}) bool {
	pr, ok := obj.(*v1beta1.PipelineRun)
	if !ok {
		return false
	}
	if len(pr.ObjectMeta.Annotations) == 0 {
		return false
	}
	isPending := pr.Spec.Status == v1beta1.PipelineRunSpecStatusPending
	wantsResolving := pr.ObjectMeta.Annotations[resolverAnnotationKey] != ""
	return isPending && wantsResolving
}
