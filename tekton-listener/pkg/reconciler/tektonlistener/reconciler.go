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

package tektonlistener

import (
	"context"
	"flag"
	"reflect"

	//"encoding/base64"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/knative/pkg/controller"
	"github.com/tektoncd/pipeline/pkg/logging"

	pipelinev1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"

	v1alpha1 "github.com/tektoncd/experimental/tekton-listener/pkg/apis/pipelineexperimental/v1alpha1"
	informers "github.com/tektoncd/experimental/tekton-listener/pkg/client/informers/externalversions/pipelineexperimental/v1alpha1"
	listers "github.com/tektoncd/experimental/tekton-listener/pkg/client/listers/pipelineexperimental/v1alpha1"
	"github.com/tektoncd/experimental/tekton-listener/pkg/reconciler"
	appsv1 "k8s.io/api/apps/v1"
)

const controllerAgentName = "tektonlistener-controller"

var (
	// The container used to accept cloud events and generate builds.
	listenerImage = flag.String("listener-image", "override:latest",
		"The container image for the cloud event listener.")
)

// Reconciler is the controller.Reconciler implementation for CloudEventsListener resources
type Reconciler struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// Listing cloud event listeners
	tektonListenerLister listers.TektonListenerLister
	// logger for inner info
	logger *zap.SugaredLogger
}

// Check that we implement the controller.Reconciler interface.
var _ controller.Reconciler = (*Reconciler)(nil)

// NewController returns a new cloud events listener controller
func NewController(
	kubeclientset kubernetes.Interface,
	tektonListenerInformer informers.TektonListenerInformer,
) *controller.Impl {
	// Enrich the logs with controller name
	logger, _ := logging.NewLogger("", "tekton-listener")

	r := &Reconciler{
		kubeclientset:        kubeclientset,
		tektonListenerLister: tektonListenerInformer.Lister(),
		logger:               logger,
	}
	impl := controller.NewImpl(r, logger, "TektonListener",
		reconciler.MustNewStatsReporter("TektonListener", r.logger))

	logger.Info("Setting up tekton-listener event handler")
	// Set up an event handler for when TektonListener resources change
	tektonListenerInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    impl.Enqueue,
		UpdateFunc: controller.PassNew(impl.Enqueue),
	})

	return impl
}

// Reconcile will create the necessary statefulset to manage the listener process
func (c *Reconciler) Reconcile(ctx context.Context, key string) error {
	c.logger.Info("tekton-listener-reconcile")

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		c.logger.Errorf("invalid resource key: %s", key)
		return nil
	}

	pl, err := c.tektonListenerLister.TektonListeners(namespace).Get(name)
	if errors.IsNotFound(err) {
		c.logger.Errorf("listener %q in work queue no longer exists", key)
		return nil
	} else if err != nil {
		return err
	}

	if pl.Spec.PipelineRunSpec == (&pipelinev1alpha1.PipelineRunSpec{}) {
		c.logger.Error("PipelineRunSpec must not be empty")
		return nil
	}

	pl = pl.DeepCopy()
	setName := pl.Name + "-statefulset"

	containerEnv := []corev1.EnvVar{
		{
			Name:  "EVENT_TYPE",
			Value: pl.Spec.EventType,
		},
		{
			Name:  "EVENT",
			Value: pl.Spec.Event,
		},
		{
			Name:  "NAMESPACE",
			Value: pl.Spec.Namespace,
		},
		{
			Name:  "LISTENER_RESOURCE",
			Value: pl.Name,
		},
		{
			Name:  "SERVICEACCOUNT",
			Value: pl.Spec.PipelineRunSpec.ServiceAccount,
		},
	}

	c.logger.Infof("launching tekton-listener %s with type: %s namespace: %s",
		pl.Name,
		pl.Spec.EventType,
		pl.Spec.Namespace,
	)

	c.logger.Infof("\n StatefulSet matchlabels: %q\n", pl.Labels)

	// Create a stateful set for the listener. It mounts a secret containing the build information.
	// The build spec may contain sensetive data and therefore the whole thing seems safest/easiest as a secret
	set := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      setName,
			Namespace: pl.Namespace,
			Labels:    pl.Labels,
		},
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: pl.Labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: pl.Labels, // the pods will inherit labels and passes them down
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: pl.Spec.PipelineRunSpec.ServiceAccount,
					Containers: []corev1.Container{
						{
							Name:  "tekton-listener",
							Image: *listenerImage,
							Env:   containerEnv,
							Ports: []corev1.ContainerPort{
								{
									Name:          "listener-port",
									ContainerPort: int32(8082),
									HostPort:      int32(8082),
								},
							},
						},
					},
				},
			},
		},
	}

	found, err := c.kubeclientset.AppsV1().StatefulSets(pl.Namespace).Get(setName, metav1.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		c.logger.Info("Creating StatefulSet", "namespace", set.Namespace, "name", set.Name)
		created, err := c.kubeclientset.AppsV1().StatefulSets(pl.Namespace).Create(set)
		pl.Status = v1alpha1.TektonListenerStatus{
			Namespace:       pl.Namespace,
			StatefulSetName: created.Name,
		}
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	if !reflect.DeepEqual(set.Spec, found.Spec) {
		found.Spec = set.Spec
		c.logger.Info("Updating Stateful Set", "namespace", set.Namespace, "name", set.Name)
		updated, err := c.kubeclientset.AppsV1().StatefulSets(pl.Namespace).Update(found)
		if err != nil {
			return err
		}
		pl.Status = v1alpha1.TektonListenerStatus{
			Namespace:       pl.Namespace,
			StatefulSetName: updated.Name,
		}
	}
	return nil
}
