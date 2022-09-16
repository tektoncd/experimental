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

// Code generated by lister-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "github.com/tektoncd/experimental/concurrency/pkg/apis/concurrency/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// ConcurrencyControlLister helps list ConcurrencyControls.
// All objects returned here must be treated as read-only.
type ConcurrencyControlLister interface {
	// List lists all ConcurrencyControls in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.ConcurrencyControl, err error)
	// ConcurrencyControls returns an object that can list and get ConcurrencyControls.
	ConcurrencyControls(namespace string) ConcurrencyControlNamespaceLister
	ConcurrencyControlListerExpansion
}

// concurrencyControlLister implements the ConcurrencyControlLister interface.
type concurrencyControlLister struct {
	indexer cache.Indexer
}

// NewConcurrencyControlLister returns a new ConcurrencyControlLister.
func NewConcurrencyControlLister(indexer cache.Indexer) ConcurrencyControlLister {
	return &concurrencyControlLister{indexer: indexer}
}

// List lists all ConcurrencyControls in the indexer.
func (s *concurrencyControlLister) List(selector labels.Selector) (ret []*v1alpha1.ConcurrencyControl, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.ConcurrencyControl))
	})
	return ret, err
}

// ConcurrencyControls returns an object that can list and get ConcurrencyControls.
func (s *concurrencyControlLister) ConcurrencyControls(namespace string) ConcurrencyControlNamespaceLister {
	return concurrencyControlNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// ConcurrencyControlNamespaceLister helps list and get ConcurrencyControls.
// All objects returned here must be treated as read-only.
type ConcurrencyControlNamespaceLister interface {
	// List lists all ConcurrencyControls in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.ConcurrencyControl, err error)
	// Get retrieves the ConcurrencyControl from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1alpha1.ConcurrencyControl, error)
	ConcurrencyControlNamespaceListerExpansion
}

// concurrencyControlNamespaceLister implements the ConcurrencyControlNamespaceLister
// interface.
type concurrencyControlNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all ConcurrencyControls in the indexer for a given namespace.
func (s concurrencyControlNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.ConcurrencyControl, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.ConcurrencyControl))
	})
	return ret, err
}

// Get retrieves the ConcurrencyControl from the indexer for a given namespace and name.
func (s concurrencyControlNamespaceLister) Get(name string) (*v1alpha1.ConcurrencyControl, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("concurrencycontrol"), name)
	}
	return obj.(*v1alpha1.ConcurrencyControl), nil
}
