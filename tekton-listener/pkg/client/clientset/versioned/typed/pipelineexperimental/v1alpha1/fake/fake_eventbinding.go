/*
Copyright 2019 The Knative Authors

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

package fake

import (
	v1alpha1 "github.com/tektoncd/experimental/tekton-listener/pkg/apis/pipelineexperimental/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeEventBindings implements EventBindingInterface
type FakeEventBindings struct {
	Fake *FakePipelineexperimentalV1alpha1
	ns   string
}

var eventbindingsResource = schema.GroupVersionResource{Group: "pipelineexperimental", Version: "v1alpha1", Resource: "eventbindings"}

var eventbindingsKind = schema.GroupVersionKind{Group: "pipelineexperimental", Version: "v1alpha1", Kind: "EventBinding"}

// Get takes name of the eventBinding, and returns the corresponding eventBinding object, and an error if there is any.
func (c *FakeEventBindings) Get(name string, options v1.GetOptions) (result *v1alpha1.EventBinding, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(eventbindingsResource, c.ns, name), &v1alpha1.EventBinding{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.EventBinding), err
}

// List takes label and field selectors, and returns the list of EventBindings that match those selectors.
func (c *FakeEventBindings) List(opts v1.ListOptions) (result *v1alpha1.EventBindingList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(eventbindingsResource, eventbindingsKind, c.ns, opts), &v1alpha1.EventBindingList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.EventBindingList{ListMeta: obj.(*v1alpha1.EventBindingList).ListMeta}
	for _, item := range obj.(*v1alpha1.EventBindingList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested eventBindings.
func (c *FakeEventBindings) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(eventbindingsResource, c.ns, opts))

}

// Create takes the representation of a eventBinding and creates it.  Returns the server's representation of the eventBinding, and an error, if there is any.
func (c *FakeEventBindings) Create(eventBinding *v1alpha1.EventBinding) (result *v1alpha1.EventBinding, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(eventbindingsResource, c.ns, eventBinding), &v1alpha1.EventBinding{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.EventBinding), err
}

// Update takes the representation of a eventBinding and updates it. Returns the server's representation of the eventBinding, and an error, if there is any.
func (c *FakeEventBindings) Update(eventBinding *v1alpha1.EventBinding) (result *v1alpha1.EventBinding, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(eventbindingsResource, c.ns, eventBinding), &v1alpha1.EventBinding{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.EventBinding), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeEventBindings) UpdateStatus(eventBinding *v1alpha1.EventBinding) (*v1alpha1.EventBinding, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(eventbindingsResource, "status", c.ns, eventBinding), &v1alpha1.EventBinding{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.EventBinding), err
}

// Delete takes name of the eventBinding and deletes it. Returns an error if one occurs.
func (c *FakeEventBindings) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(eventbindingsResource, c.ns, name), &v1alpha1.EventBinding{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeEventBindings) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(eventbindingsResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.EventBindingList{})
	return err
}

// Patch applies the patch and returns the patched eventBinding.
func (c *FakeEventBindings) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.EventBinding, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(eventbindingsResource, c.ns, name, data, subresources...), &v1alpha1.EventBinding{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.EventBinding), err
}
