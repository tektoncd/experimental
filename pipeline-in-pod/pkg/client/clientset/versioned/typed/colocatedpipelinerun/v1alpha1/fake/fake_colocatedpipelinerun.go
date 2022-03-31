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

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	v1alpha1 "github.com/tektoncd/experimental/pipeline-in-pod/pkg/apis/colocatedpipelinerun/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeColocatedPipelineRuns implements ColocatedPipelineRunInterface
type FakeColocatedPipelineRuns struct {
	Fake *FakeTektonV1alpha1
	ns   string
}

var colocatedpipelinerunsResource = schema.GroupVersionResource{Group: "tekton.dev", Version: "v1alpha1", Resource: "colocatedpipelineruns"}

var colocatedpipelinerunsKind = schema.GroupVersionKind{Group: "tekton.dev", Version: "v1alpha1", Kind: "ColocatedPipelineRun"}

// Get takes name of the colocatedPipelineRun, and returns the corresponding colocatedPipelineRun object, and an error if there is any.
func (c *FakeColocatedPipelineRuns) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.ColocatedPipelineRun, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(colocatedpipelinerunsResource, c.ns, name), &v1alpha1.ColocatedPipelineRun{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ColocatedPipelineRun), err
}

// List takes label and field selectors, and returns the list of ColocatedPipelineRuns that match those selectors.
func (c *FakeColocatedPipelineRuns) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.ColocatedPipelineRunList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(colocatedpipelinerunsResource, colocatedpipelinerunsKind, c.ns, opts), &v1alpha1.ColocatedPipelineRunList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.ColocatedPipelineRunList{ListMeta: obj.(*v1alpha1.ColocatedPipelineRunList).ListMeta}
	for _, item := range obj.(*v1alpha1.ColocatedPipelineRunList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested colocatedPipelineRuns.
func (c *FakeColocatedPipelineRuns) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(colocatedpipelinerunsResource, c.ns, opts))

}

// Create takes the representation of a colocatedPipelineRun and creates it.  Returns the server's representation of the colocatedPipelineRun, and an error, if there is any.
func (c *FakeColocatedPipelineRuns) Create(ctx context.Context, colocatedPipelineRun *v1alpha1.ColocatedPipelineRun, opts v1.CreateOptions) (result *v1alpha1.ColocatedPipelineRun, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(colocatedpipelinerunsResource, c.ns, colocatedPipelineRun), &v1alpha1.ColocatedPipelineRun{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ColocatedPipelineRun), err
}

// Update takes the representation of a colocatedPipelineRun and updates it. Returns the server's representation of the colocatedPipelineRun, and an error, if there is any.
func (c *FakeColocatedPipelineRuns) Update(ctx context.Context, colocatedPipelineRun *v1alpha1.ColocatedPipelineRun, opts v1.UpdateOptions) (result *v1alpha1.ColocatedPipelineRun, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(colocatedpipelinerunsResource, c.ns, colocatedPipelineRun), &v1alpha1.ColocatedPipelineRun{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ColocatedPipelineRun), err
}

// Delete takes name of the colocatedPipelineRun and deletes it. Returns an error if one occurs.
func (c *FakeColocatedPipelineRuns) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(colocatedpipelinerunsResource, c.ns, name), &v1alpha1.ColocatedPipelineRun{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeColocatedPipelineRuns) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(colocatedpipelinerunsResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.ColocatedPipelineRunList{})
	return err
}

// Patch applies the patch and returns the patched colocatedPipelineRun.
func (c *FakeColocatedPipelineRuns) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.ColocatedPipelineRun, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(colocatedpipelinerunsResource, c.ns, name, pt, data, subresources...), &v1alpha1.ColocatedPipelineRun{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ColocatedPipelineRun), err
}
