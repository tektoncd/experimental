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

package framework

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/tektoncd/experimental/remote-resolution/pkg/apis/resolution/v1alpha1"
	rrclient "github.com/tektoncd/experimental/remote-resolution/pkg/client/clientset/versioned"
	rrv1alpha1 "github.com/tektoncd/experimental/remote-resolution/pkg/client/listers/resolution/v1alpha1"
	"github.com/tektoncd/experimental/remote-resolution/pkg/resolution"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/reconciler"
)

type Reconciler struct {
	// Implements reconciler.LeaderAware
	reconciler.LeaderAwareFuncs

	resolver                 Resolver
	kubeClientSet            kubernetes.Interface
	resourceRequestLister    rrv1alpha1.ResourceRequestLister
	resourceRequestClientSet rrclient.Interface
}

func (r *Reconciler) Reconcile(ctx context.Context, key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		err = &resolution.ErrorInvalidResourceKey{Key: key, Original: err}
		return controller.NewPermanentError(err)
	}

	rr, err := r.resourceRequestLister.ResourceRequests(namespace).Get(name)
	if err != nil {
		err := &resolution.ErrorGettingResource{Kind: "resourcerequest", Key: key, Original: err}
		return controller.NewPermanentError(err)
	}

	if rr.IsDone() {
		return nil
	}

	validationError := r.resolver.ValidateParams(rr.Spec.Parameters)
	if validationError != nil {
		err := &resolution.ErrorInvalidRequest{
			ResourceRequestKey: key,
			Message:            validationError.Error(),
		}
		return r.OnError(ctx, rr, err)
	}

	resourceString, annotations, resolveErr := r.resolver.Resolve(rr.Spec.Parameters)
	if resolveErr != nil {
		err := &resolution.ErrorGettingResource{
			Kind:     r.resolver.GetName(),
			Key:      key,
			Original: resolveErr,
		}
		return r.OnError(ctx, rr, err)
	}

	r.writeResponse(ctx, rr, resourceString, annotations)

	return nil
}

// OnError is used to handle any situation where a ResourceRequest has
// reached a terminal situation that cannot be recovered from. If
// updating the ResourceRequest fails for some reason then a
// non-permanent error is returned so that another attempt to update it
// can be made.
func (r *Reconciler) OnError(ctx context.Context, rr *v1alpha1.ResourceRequest, err error) error {
	if rr == nil {
		return controller.NewPermanentError(err)
	}
	if rr != nil && err != nil {
		updateErr := r.MarkFailed(ctx, rr, err)
		if updateErr != nil {
			return err
		}
		return controller.NewPermanentError(err)
	}
	return nil
}

// MarkFailed updates a ResourceRequest as having failed. It returns
// errors that occur during the update process or nil if the update
// appeared to succeed.
func (r *Reconciler) MarkFailed(ctx context.Context, rr *v1alpha1.ResourceRequest, resolutionErr error) error {
	key := fmt.Sprintf("%s/%s", rr.Namespace, rr.Name)
	reason, resolutionErr := resolution.ReasonError(resolutionErr)
	latestGeneration, err := r.resourceRequestClientSet.ResolutionV1alpha1().ResourceRequests(rr.Namespace).Get(ctx, rr.Name, metav1.GetOptions{})
	if err != nil {
		logging.FromContext(ctx).Warnf("error getting resourcerequest %q to update as failed: %v", key, err)
		return err
	}

	latestGeneration.Status.MarkFailed(reason, resolutionErr.Error())
	_, err = r.resourceRequestClientSet.ResolutionV1alpha1().ResourceRequests(rr.Namespace).UpdateStatus(ctx, latestGeneration, metav1.UpdateOptions{})
	if err != nil {
		logging.FromContext(ctx).Warnf("error marking resourcerequest %q as failed: %v", key, err)
		return err
	}
	return nil
}

type statusDataPatch struct {
	Annotations map[string]string `json:"annotations"`
	Data        string            `json:"data"`
}

func (r *Reconciler) writeResponse(ctx context.Context, rr *v1alpha1.ResourceRequest, data string, annotations map[string]string) error {
	patchBytes, err := json.Marshal(map[string]statusDataPatch{
		"status": statusDataPatch{
			Data:        data,
			Annotations: annotations,
		},
	})
	if err != nil {
		return r.OnError(ctx, rr, &resolution.ErrorUpdatingRequest{
			ResourceRequestKey: fmt.Sprintf("%s/%s", rr.Namespace, rr.Name),
			Original:           fmt.Errorf("error serializing resource request patch: %w", err),
		})
	}
	_, err = r.resourceRequestClientSet.ResolutionV1alpha1().ResourceRequests(rr.Namespace).Patch(ctx, rr.Name, types.MergePatchType, patchBytes, metav1.PatchOptions{}, "status")
	if err != nil {
		return r.OnError(ctx, rr, &resolution.ErrorUpdatingRequest{
			ResourceRequestKey: fmt.Sprintf("%s/%s", rr.Namespace, rr.Name),
			Original:           err,
		})
	}

	return nil
}
