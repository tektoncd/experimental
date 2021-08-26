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

// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	"time"

	v1alpha1 "github.com/tektoncd/experimental/celeval/pkg/apis/celeval/v1alpha1"
	scheme "github.com/tektoncd/experimental/celeval/pkg/client/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// CELEvalsGetter has a method to return a CELEvalInterface.
// A group's client should implement this interface.
type CELEvalsGetter interface {
	CELEvals(namespace string) CELEvalInterface
}

// CELEvalInterface has methods to work with CELEval resources.
type CELEvalInterface interface {
	Create(ctx context.Context, cELEval *v1alpha1.CELEval, opts v1.CreateOptions) (*v1alpha1.CELEval, error)
	Update(ctx context.Context, cELEval *v1alpha1.CELEval, opts v1.UpdateOptions) (*v1alpha1.CELEval, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.CELEval, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.CELEvalList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.CELEval, err error)
	CELEvalExpansion
}

// cELEvals implements CELEvalInterface
type cELEvals struct {
	client rest.Interface
	ns     string
}

// newCELEvals returns a CELEvals
func newCELEvals(c *CustomV1alpha1Client, namespace string) *cELEvals {
	return &cELEvals{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the cELEval, and returns the corresponding cELEval object, and an error if there is any.
func (c *cELEvals) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.CELEval, err error) {
	result = &v1alpha1.CELEval{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("celevals").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of CELEvals that match those selectors.
func (c *cELEvals) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.CELEvalList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.CELEvalList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("celevals").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested cELEvals.
func (c *cELEvals) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("celevals").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a cELEval and creates it.  Returns the server's representation of the cELEval, and an error, if there is any.
func (c *cELEvals) Create(ctx context.Context, cELEval *v1alpha1.CELEval, opts v1.CreateOptions) (result *v1alpha1.CELEval, err error) {
	result = &v1alpha1.CELEval{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("celevals").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(cELEval).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a cELEval and updates it. Returns the server's representation of the cELEval, and an error, if there is any.
func (c *cELEvals) Update(ctx context.Context, cELEval *v1alpha1.CELEval, opts v1.UpdateOptions) (result *v1alpha1.CELEval, err error) {
	result = &v1alpha1.CELEval{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("celevals").
		Name(cELEval.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(cELEval).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the cELEval and deletes it. Returns an error if one occurs.
func (c *cELEvals) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("celevals").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *cELEvals) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("celevals").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched cELEval.
func (c *cELEvals) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.CELEval, err error) {
	result = &v1alpha1.CELEval{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("celevals").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
