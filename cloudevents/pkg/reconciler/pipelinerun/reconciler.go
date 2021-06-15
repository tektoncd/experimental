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
	"github.com/tektoncd/experimental/cloudevents/pkg/reconciler/events"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/reconciler/events/cloudevent"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/logging"
	kreconciler "knative.dev/pkg/reconciler"
	"knative.dev/pkg/tracker"
)

type CDEventType string

const (
//// Environment Annotation
//cdEnv = "tekton.dev/cd.environment.v1"
//// Environment Annotation Values
//cdEnvCreated = "created"
//cdEnvModified = "modified"
//cdEnvDeleted = "deleted"
//
//// Service Annotation
//cdSvc = "tekton.dev/cd.service.v1"
//// states
//cdSvcStarted = "deployed"
//cdSvcUpgraded = "upgraded"
//cdSvcRolledback = "rolledback"
//cdSvcRemoved = "removed"
//
//// TaskRun Annotation
//cdTR = "tekton.dev/cd.taskrun.v1"
//// states
//cdTRStarted = "started"
//cdTRFinished = "finished"
//cdTRQueued = "queued"
//
//// Repository Annotation
//cdRepo = "tekton.dev/cd.repository.v1"
//// states
//cdRepoCreated = "created"
//cdRepoModified = "modified"
//cdRepoDeleted = "deleted"

//// PipelineRun Annotations
//cdPipelineRunID = "cd.tekton.dev/pipelinerun.v1/id"
//// states
//cdPRStarted = "started"
//cdPRFinished = "finished"
//cdPRQueued = "queued"
//
//// PipelineRun events
//PipelineRunStartedEventV1 CDEventType = "cd.pipelinerun.started.v1"
//PipelineRunFinishedEventV1 CDEventType = "cd.pipelinerun.finished.v1"
//PipelineRunQueuedEventV1 CDEventType = "cd.pipelinerun.queued.v1"
)

func (t CDEventType) String() string {
	return string(t)
}

type Reconciler struct {
	cloudEventClient cloudevent.CEClient
	tracker          tracker.Interface
}

// ReconcileKind implements Interface.ReconcileKind.
func (c *Reconciler) ReconcileKind(ctx context.Context, pr *v1beta1.PipelineRun) kreconciler.Event {
	logger := logging.FromContext(ctx)
	ctx = cloudevent.ToContext(ctx, c.cloudEventClient)
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
		prEvents.Status.InitializeConditions()
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
