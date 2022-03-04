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

package pipelinerun

import (
	"context"

	lru "github.com/hashicorp/golang-lru"
	"github.com/tektoncd/experimental/cloudevents/pkg/reconciler/events"
	"github.com/tektoncd/experimental/cloudevents/pkg/reconciler/events/cache"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/reconciler/events/cloudevent"
	"k8s.io/apimachinery/pkg/util/clock"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/logging"
	kreconciler "knative.dev/pkg/reconciler"
)

type CDEventType string

func (t CDEventType) String() string {
	return string(t)
}

type Reconciler struct {
	cloudEventClient cloudevent.CEClient
	cacheClient      *lru.Cache
	Clock            clock.PassiveClock
}

// ReconcileKind implements Interface.ReconcileKind.
func (c *Reconciler) ReconcileKind(ctx context.Context, pr *v1beta1.PipelineRun) kreconciler.Event {
	logger := logging.FromContext(ctx)
	ctx = cloudevent.ToContext(ctx, c.cloudEventClient)
	ctx = cache.ToContext(ctx, c.cacheClient)
	logger.Infof("Reconciling %s", pr.Name)

	// Create a copy of the pr object, else the controller would try and sync back any change we made
	prEvents := *pr.DeepCopy()

	// It could be that there is no condition yet.
	// Because of the way the PipelineRun controller works, the start condition is
	// often not set, as the PipelineRun will reach "Running" before the first reconcile
	// cycle is complete. Because of that, we must emulate here the behaviour of the
	// PipelineRun controller, and initialise conditions. The fact that this is being
	// reconciled does not imply that the PipelineRun is being reconciled by the PipelineRun
	// controller has well, so this is only a temporary fix for the initial PoC.
	if !pr.HasStarted() && !pr.IsPending() {
		prEvents.Status.InitializeConditions(c.Clock)
		// In case node time was not synchronized, when controller has been scheduled to other nodes.
		if prEvents.Status.StartTime.Sub(pr.CreationTimestamp.Time) < 0 {
			logger.Warnf("PipelineRun %s createTimestamp %s is after the pipelineRun started %s", pr.GetNamespacedName().String(), pr.CreationTimestamp, pr.Status.StartTime)
			prEvents.Status.StartTime = &pr.CreationTimestamp
		}
	}

	// Read and log the condition
	condition := prEvents.Status.GetCondition(apis.ConditionSucceeded)
	logger.Debugf("Emitting cloudevent for %s, condition: %s", prEvents.Name, condition)

	events.Emit(ctx, &prEvents)

	return nil
}
