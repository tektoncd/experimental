/*
Copyright 2019 The Knative Authors.

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

package eventbinding

import (
	"context"
	"fmt"
	"reflect"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/knative/pkg/controller"
	"github.com/tektoncd/pipeline/pkg/logging"

	pipelinev1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	pipelineinformers "github.com/tektoncd/pipeline/pkg/client/informers/externalversions/pipeline/v1alpha1"
	pipelinelisters "github.com/tektoncd/pipeline/pkg/client/listers/pipeline/v1alpha1"

	v1alpha1 "github.com/tektoncd/experimental/tekton-listener/pkg/apis/pipelineexperimental/v1alpha1"
	informers "github.com/tektoncd/experimental/tekton-listener/pkg/client/informers/externalversions/pipelineexperimental/v1alpha1"
	listers "github.com/tektoncd/experimental/tekton-listener/pkg/client/listers/pipelineexperimental/v1alpha1"
	"github.com/tektoncd/experimental/tekton-listener/pkg/reconciler"
)

const controllerAgentName = "eventbinding-controller"

// Reconciler is the controller.Reconciler implementation for eventbinding resources
type Reconciler struct {
	*reconciler.Base
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// Listing cloud event listeners
	eventBindingLister listers.EventBindingLister
	// listing pipelines associated with binding
	pipelineLister pipelinelisters.PipelineLister
	//listing tekton listeners
	tektonListenerLister listers.TektonListenerLister
	// logger for inner info
	logger *zap.SugaredLogger
}

// Check that we implement the controller.Reconciler interface.
var _ controller.Reconciler = (*Reconciler)(nil)

// NewController returns a new event binding controller
func NewController(
	opt reconciler.Options,
	kubeclientset kubernetes.Interface,
	eventBindingInformer informers.EventBindingInformer,
	tektonListenerInformer informers.TektonListenerInformer,
	pipelineListerInformer pipelineinformers.PipelineInformer,
) *controller.Impl {
	// Enrich the logs with controller name
	logger, _ := logging.NewLogger("", "event-binding")

	r := &Reconciler{
		Base:                 reconciler.NewBase(opt, controllerAgentName),
		kubeclientset:        kubeclientset,
		eventBindingLister:   eventBindingInformer.Lister(),
		tektonListenerLister: tektonListenerInformer.Lister(),
		pipelineLister:       pipelineListerInformer.Lister(),
		logger:               logger,
	}
	impl := controller.NewImpl(r, logger, "EventBinding",
		reconciler.MustNewStatsReporter("EventBinding", r.logger))

	logger.Info("Setting up event-binding event handler")
	// Set up an event handler for when EventBinding resources change
	eventBindingInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    impl.Enqueue,
		UpdateFunc: controller.PassNew(impl.Enqueue),
	})

	return impl
}

// Reconcile will
// - create the necessary resources
// - start the listener
func (c *Reconciler) Reconcile(ctx context.Context, key string) error {
	c.logger.Info("event-binding-reconcile")
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		c.logger.Errorf("invalid resource key: %s", key)
		return nil
	}

	binding, err := c.eventBindingLister.EventBindings(namespace).Get(name)
	if errors.IsNotFound(err) {
		c.logger.Errorf("eventing binding %q in work queue no longer exists", key)
		return nil
	} else if err != nil {
		return err
	}

	if binding.Spec.PipelineRef.Name == "" {
		c.logger.Error("PipelineRunSpec must not be empty")
		return nil
	}

	// Get this bindings Pipeline from PipelineRef
	c.logger.Info("retrieving associated pipeline")
	// Caveat: The Pipeline must exist within the same namespace the EventBinding was created in.
	// This seems like a reasonable expectation for now, but someday, someone might ask why and here it is below.
	// If PipelineRef carried along a namespace as well (it prob should, no?), we could handle this more gracefully.
	_, err = c.pipelineLister.Pipelines(namespace).Get(binding.Spec.PipelineRef.Name)
	if errors.IsNotFound(err) {
		c.logger.Errorf("eventing binding specifies pipeline %q which doesnt exist", binding.Spec.PipelineRef.Name)
		return err
	}
	if err != nil {
		c.logger.Errorf("error getting associated pipeline %q", err)
		return err
	}

	c.logger.Info("Creating resource templates")
	var pipelineResourceBindings []pipelinev1alpha1.PipelineResourceBinding
	// Build the resource dependancies
	for _, resource := range binding.Spec.ResourceTemplates {
		// :dog-flying-around-in-space:
		_, err := c.PipelineClientSet.TektonV1alpha1().PipelineResources(resource.Namespace).Get(resource.Name, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			_, err := c.PipelineClientSet.TektonV1alpha1().PipelineResources(resource.Namespace).Create(&resource)
			if err != nil {
				return err
			}

			c.logger.Infof("created resource %q for eventbinding %q", resource.Name, binding.Name)
		}
		if err != nil {
			return err
		}

		pipelineResourceBindings = append(pipelineResourceBindings, pipelinev1alpha1.PipelineResourceBinding{
			Name: resource.Name,
			ResourceRef: pipelinev1alpha1.PipelineResourceRef{
				Name:       string(resource.Spec.Type), // TODO: ? is this right?
				APIVersion: "v1alpha1",
			},
		})
	}

	tektonListenerName := fmt.Sprintf("%s-listener", binding.Name)

	c.logger.Infof("creating listener for event-type: %q for event-name: %q", binding.Spec.EventRef.EventType, binding.Spec.EventRef.EventName)

	// Create a tekton listener!
	newListener := &v1alpha1.TektonListener{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tektonListenerName,
			Namespace: binding.Namespace,
			Labels:    binding.Labels,
		},
		Spec: v1alpha1.TektonListenerSpec{
			PipelineRef: binding.Spec.PipelineRef,
			EventType:   binding.Spec.EventRef.EventName,
			Event:       binding.Spec.EventRef.EventType,
			Namespace:   binding.Namespace,
			PipelineRunSpec: &pipelinev1alpha1.PipelineRunSpec{
				PipelineRef: binding.Spec.PipelineRef,
				Trigger: pipelinev1alpha1.PipelineTrigger{
					Name: binding.Name,
					Type: "eventbinding",
				},
				Resources:      pipelineResourceBindings,
				Params:         binding.Spec.Params,
				ServiceAccount: binding.Spec.ServiceAccount,
			},
		},
	}

	c.logger.Info("attempting to retrieve associated tekton-listener")
	found, err := c.tektonListenerLister.TektonListeners(binding.Namespace).Get(tektonListenerName)
	if errors.IsNotFound(err) {
		created, err := c.ExperimentClientSet.Pipelineexperimental().TektonListeners(binding.Namespace).Create(newListener)
		if err != nil {
			return err
		}
		c.logger.Infof("created tekton listener %q for eventbinding %q", created.Name, binding.Name)
	}
	if err != nil {
		return err
	}

	if !reflect.DeepEqual(newListener.Spec, found.Spec) {
		found.Spec = newListener.Spec
		c.logger.Info("Updating Stateful Set", "namespace", found.Namespace, "name", found.Name)
		_, err := c.ExperimentClientSet.Pipelineexperimental().TektonListeners(binding.Namespace).Update(found)
		if err != nil {
			return err
		}
	}

	binding.Status = v1alpha1.EventBindingStatus{
		ListenerName: newListener.Name,
		Namespace:    newListener.Namespace,
	}
	return nil
}
