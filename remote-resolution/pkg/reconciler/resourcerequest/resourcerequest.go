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

package resourcerequest

import (
	"context"

	"github.com/tektoncd/experimental/remote-resolution/pkg/apis/resolution/v1alpha1"
	rrclient "github.com/tektoncd/experimental/remote-resolution/pkg/client/clientset/versioned"
	rrreconciler "github.com/tektoncd/experimental/remote-resolution/pkg/client/injection/reconciler/resolution/v1alpha1/resourcerequest"
	"k8s.io/client-go/kubernetes"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/reconciler"
)

type Reconciler struct {
	kubeclient kubernetes.Interface
	rrclient   rrclient.Interface
}

const waitingStatusMessage = "waiting for resolver"

func (r *Reconciler) ReconcileKind(ctx context.Context, rr *v1alpha1.ResourceRequest) reconciler.Event {
	if rr.IsDone() {
		return nil
	}

	if rr.Status.GetCondition(apis.ConditionSucceeded) == nil {
		rr.Status.InitializeConditions()
	}

	if rr.Status.Data != "" {
		rr.Status.MarkSucceeded()
	} else {
		rr.Status.MarkInProgress(waitingStatusMessage)
	}

	// TODO(sbwsg): enforce timeout of requests here

	return nil
}

var _ rrreconciler.Interface = (*Reconciler)(nil)
