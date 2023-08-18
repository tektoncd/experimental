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
	internalinterfaces "github.com/tektoncd/experimental/metrics-operator/pkg/client/informers/externalversions/internalinterfaces"
)

// Interface provides access to all the informers in this group version.
type Interface interface {
	// TaskMonitors returns a TaskMonitorInformer.
	TaskMonitors() TaskMonitorInformer
	// TaskRunMonitors returns a TaskRunMonitorInformer.
	TaskRunMonitors() TaskRunMonitorInformer
}

type version struct {
	factory          internalinterfaces.SharedInformerFactory
	namespace        string
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// New returns a new Interface.
func New(f internalinterfaces.SharedInformerFactory, namespace string, tweakListOptions internalinterfaces.TweakListOptionsFunc) Interface {
	return &version{factory: f, namespace: namespace, tweakListOptions: tweakListOptions}
}

// TaskMonitors returns a TaskMonitorInformer.
func (v *version) TaskMonitors() TaskMonitorInformer {
	return &taskMonitorInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// TaskRunMonitors returns a TaskRunMonitorInformer.
func (v *version) TaskRunMonitors() TaskRunMonitorInformer {
	return &taskRunMonitorInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}
