/*
Copyright 2020 The Knative Authors

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

// Code generated by informer-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	time "time"

	colocatedpipelinerunv1alpha1 "github.com/tektoncd/experimental/pipeline-in-pod/pkg/apis/colocatedpipelinerun/v1alpha1"
	versioned "github.com/tektoncd/experimental/pipeline-in-pod/pkg/client/clientset/versioned"
	internalinterfaces "github.com/tektoncd/experimental/pipeline-in-pod/pkg/client/informers/externalversions/internalinterfaces"
	v1alpha1 "github.com/tektoncd/experimental/pipeline-in-pod/pkg/client/listers/colocatedpipelinerun/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// ColocatedPipelineRunInformer provides access to a shared informer and lister for
// ColocatedPipelineRuns.
type ColocatedPipelineRunInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.ColocatedPipelineRunLister
}

type colocatedPipelineRunInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewColocatedPipelineRunInformer constructs a new informer for ColocatedPipelineRun type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewColocatedPipelineRunInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredColocatedPipelineRunInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredColocatedPipelineRunInformer constructs a new informer for ColocatedPipelineRun type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredColocatedPipelineRunInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.TektonV1alpha1().ColocatedPipelineRuns(namespace).List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.TektonV1alpha1().ColocatedPipelineRuns(namespace).Watch(context.TODO(), options)
			},
		},
		&colocatedpipelinerunv1alpha1.ColocatedPipelineRun{},
		resyncPeriod,
		indexers,
	)
}

func (f *colocatedPipelineRunInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredColocatedPipelineRunInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *colocatedPipelineRunInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&colocatedpipelinerunv1alpha1.ColocatedPipelineRun{}, f.defaultInformer)
}

func (f *colocatedPipelineRunInformer) Lister() v1alpha1.ColocatedPipelineRunLister {
	return v1alpha1.NewColocatedPipelineRunLister(f.Informer().GetIndexer())
}
