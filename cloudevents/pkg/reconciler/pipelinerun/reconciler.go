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

	// Read and log the condition
	condition := pr.Status.GetCondition(apis.ConditionSucceeded)
	logger.Debugf("Emitting cloudevent for %s, condition: %s", pr.Name, condition)

	events.Emit(ctx, pr)

	return nil
}
