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

package taskrun

import (
	"context"

	lru "github.com/hashicorp/golang-lru"
	"github.com/tektoncd/experimental/cloudevents/pkg/apis/config"
	"github.com/tektoncd/experimental/cloudevents/pkg/reconciler/events"
	"github.com/tektoncd/experimental/cloudevents/pkg/reconciler/events/cache"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/reconciler/events/cloudevent"
	"k8s.io/apimachinery/pkg/util/clock"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/logging"
	kreconciler "knative.dev/pkg/reconciler"
)

type Reconciler struct {
	cloudEventClient cloudevents.Client
	cacheClient      *lru.Cache
	Clock            clock.PassiveClock
}

// ReconcileKind implements Interface.ReconcileKind.
func (c *Reconciler) ReconcileKind(ctx context.Context, tr *v1beta1.TaskRun) kreconciler.Event {
	logger := logging.FromContext(ctx)
	configs := config.FromContextOrDefaults(ctx)
	ctx = cloudevent.ToContext(ctx, c.cloudEventClient)
	ctx = cache.ToContext(ctx, c.cacheClient)
	logger.Infof("Reconciling %s", tr.Name)

	// Create a copy of the tr object, else the controller would try and sync back any change we made
	trEvents := *tr.DeepCopy()

	cloudEventsFormat := configs.Defaults.DefaultCloudEventsFormat
	if cloudEventsFormat == config.EventFormatLegacy {
		// The tekton pipelines controller (legacy) sends a "started" event when
		// the resource is seen for the first time, and it does so after setting
		// the initial resource condition. To ensure parity with the legacy
		// controller, we must emulate here the behaviour of the TaskRun
		// controller, and initialise conditions.
		if !tr.HasStarted() {
			trEvents.Status.InitializeConditions()
			// In case node time was not synchronized, when controller has been scheduled to other nodes.
			if trEvents.Status.StartTime.Sub(tr.CreationTimestamp.Time) < 0 {
				logger.Warnf("TaskRun %s createTimestamp %s is after the taskRun started %s", tr.GetNamespacedName().String(), tr.CreationTimestamp, tr.Status.StartTime)
				trEvents.Status.StartTime = &tr.CreationTimestamp
			}
		}
	}

	// Read and log the condition
	condition := trEvents.Status.GetCondition(apis.ConditionSucceeded)
	logger.Debugf("Emitting cloudevent for %s, condition: %s", trEvents.Name, condition)

	events.Emit(ctx, &trEvents)

	return nil
}
