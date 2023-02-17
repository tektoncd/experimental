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

	v1alpha1 "github.com/tektoncd/experimental/metrics-operator/pkg/apis/monitoring/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeTaskMonitors implements TaskMonitorInterface
type FakeTaskMonitors struct {
	Fake *FakeMetricsV1alpha1
	ns   string
}

var taskmonitorsResource = schema.GroupVersionResource{Group: "metrics.tekton.dev", Version: "v1alpha1", Resource: "taskmonitors"}

var taskmonitorsKind = schema.GroupVersionKind{Group: "metrics.tekton.dev", Version: "v1alpha1", Kind: "TaskMonitor"}

// Get takes name of the taskMonitor, and returns the corresponding taskMonitor object, and an error if there is any.
func (c *FakeTaskMonitors) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.TaskMonitor, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(taskmonitorsResource, c.ns, name), &v1alpha1.TaskMonitor{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.TaskMonitor), err
}

// List takes label and field selectors, and returns the list of TaskMonitors that match those selectors.
func (c *FakeTaskMonitors) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.TaskMonitorList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(taskmonitorsResource, taskmonitorsKind, c.ns, opts), &v1alpha1.TaskMonitorList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.TaskMonitorList{ListMeta: obj.(*v1alpha1.TaskMonitorList).ListMeta}
	for _, item := range obj.(*v1alpha1.TaskMonitorList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested taskMonitors.
func (c *FakeTaskMonitors) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(taskmonitorsResource, c.ns, opts))

}

// Create takes the representation of a taskMonitor and creates it.  Returns the server's representation of the taskMonitor, and an error, if there is any.
func (c *FakeTaskMonitors) Create(ctx context.Context, taskMonitor *v1alpha1.TaskMonitor, opts v1.CreateOptions) (result *v1alpha1.TaskMonitor, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(taskmonitorsResource, c.ns, taskMonitor), &v1alpha1.TaskMonitor{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.TaskMonitor), err
}

// Update takes the representation of a taskMonitor and updates it. Returns the server's representation of the taskMonitor, and an error, if there is any.
func (c *FakeTaskMonitors) Update(ctx context.Context, taskMonitor *v1alpha1.TaskMonitor, opts v1.UpdateOptions) (result *v1alpha1.TaskMonitor, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(taskmonitorsResource, c.ns, taskMonitor), &v1alpha1.TaskMonitor{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.TaskMonitor), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeTaskMonitors) UpdateStatus(ctx context.Context, taskMonitor *v1alpha1.TaskMonitor, opts v1.UpdateOptions) (*v1alpha1.TaskMonitor, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(taskmonitorsResource, "status", c.ns, taskMonitor), &v1alpha1.TaskMonitor{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.TaskMonitor), err
}

// Delete takes name of the taskMonitor and deletes it. Returns an error if one occurs.
func (c *FakeTaskMonitors) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(taskmonitorsResource, c.ns, name, opts), &v1alpha1.TaskMonitor{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeTaskMonitors) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(taskmonitorsResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.TaskMonitorList{})
	return err
}

// Patch applies the patch and returns the patched taskMonitor.
func (c *FakeTaskMonitors) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.TaskMonitor, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(taskmonitorsResource, c.ns, name, pt, data, subresources...), &v1alpha1.TaskMonitor{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.TaskMonitor), err
}
