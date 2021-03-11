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
	v1alpha1 "github.com/tektoncd/experimental/pipeline-loops/pkg/apis/pipelineloop/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// PipelineLoopLister helps list PipelineLoops.
type PipelineLoopLister interface {
	// List lists all PipelineLoops in the indexer.
	List(selector labels.Selector) (ret []*v1alpha1.PipelineLoop, err error)
	// PipelineLoops returns an object that can list and get PipelineLoops.
	PipelineLoops(namespace string) PipelineLoopNamespaceLister
	PipelineLoopListerExpansion
}

// pipelineLoopLister implements the PipelineLoopLister interface.
type pipelineLoopLister struct {
	indexer cache.Indexer
}

// NewPipelineLoopLister returns a new PipelineLoopLister.
func NewPipelineLoopLister(indexer cache.Indexer) PipelineLoopLister {
	return &pipelineLoopLister{indexer: indexer}
}

// List lists all PipelineLoops in the indexer.
func (s *pipelineLoopLister) List(selector labels.Selector) (ret []*v1alpha1.PipelineLoop, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.PipelineLoop))
	})
	return ret, err
}

// PipelineLoops returns an object that can list and get PipelineLoops.
func (s *pipelineLoopLister) PipelineLoops(namespace string) PipelineLoopNamespaceLister {
	return pipelineLoopNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// PipelineLoopNamespaceLister helps list and get PipelineLoops.
type PipelineLoopNamespaceLister interface {
	// List lists all PipelineLoops in the indexer for a given namespace.
	List(selector labels.Selector) (ret []*v1alpha1.PipelineLoop, err error)
	// Get retrieves the PipelineLoop from the indexer for a given namespace and name.
	Get(name string) (*v1alpha1.PipelineLoop, error)
	PipelineLoopNamespaceListerExpansion
}

// pipelineLoopNamespaceLister implements the PipelineLoopNamespaceLister
// interface.
type pipelineLoopNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all PipelineLoops in the indexer for a given namespace.
func (s pipelineLoopNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.PipelineLoop, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.PipelineLoop))
	})
	return ret, err
}

// Get retrieves the PipelineLoop from the indexer for a given namespace and name.
func (s pipelineLoopNamespaceLister) Get(name string) (*v1alpha1.PipelineLoop, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("pipelineloop"), name)
	}
	return obj.(*v1alpha1.PipelineLoop), nil
}
