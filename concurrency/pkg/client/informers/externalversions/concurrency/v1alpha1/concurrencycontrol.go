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

	concurrencyv1alpha1 "github.com/tektoncd/experimental/concurrency/pkg/apis/concurrency/v1alpha1"
	versioned "github.com/tektoncd/experimental/concurrency/pkg/client/clientset/versioned"
	internalinterfaces "github.com/tektoncd/experimental/concurrency/pkg/client/informers/externalversions/internalinterfaces"
	v1alpha1 "github.com/tektoncd/experimental/concurrency/pkg/client/listers/concurrency/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// ConcurrencyControlInformer provides access to a shared informer and lister for
// ConcurrencyControls.
type ConcurrencyControlInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.ConcurrencyControlLister
}

type concurrencyControlInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewConcurrencyControlInformer constructs a new informer for ConcurrencyControl type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewConcurrencyControlInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredConcurrencyControlInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredConcurrencyControlInformer constructs a new informer for ConcurrencyControl type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredConcurrencyControlInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.CustomV1alpha1().ConcurrencyControls(namespace).List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.CustomV1alpha1().ConcurrencyControls(namespace).Watch(context.TODO(), options)
			},
		},
		&concurrencyv1alpha1.ConcurrencyControl{},
		resyncPeriod,
		indexers,
	)
}

func (f *concurrencyControlInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredConcurrencyControlInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *concurrencyControlInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&concurrencyv1alpha1.ConcurrencyControl{}, f.defaultInformer)
}

func (f *concurrencyControlInformer) Lister() v1alpha1.ConcurrencyControlLister {
	return v1alpha1.NewConcurrencyControlLister(f.Informer().GetIndexer())
}
