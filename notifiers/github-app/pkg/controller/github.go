// Copyright 2020 The Tekton Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controller

import (
	"context"
	"fmt"
	"strconv"

	"github.com/google/go-github/v32/github"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
)

// UpsertCheckRun updates or creates a check run for the given TaskRun.
func UpsertCheckRun(ctx context.Context, client *github.Client, tr *v1beta1.TaskRun, output *github.CheckRunOutput) (*github.CheckRun, error) {
	owner := tr.Annotations[key("owner")]
	repo := tr.Annotations[key("repo")]
	commit := tr.Annotations[key("commit")]

	status, conclusion := status(tr.Status)

	if id, ok := tr.Annotations[key("checkrun")]; ok {
		// A check run was already associated to the TaskRun - update.
		n, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("error converting check run id: %v", err)
		}
		cr, _, err := client.Checks.UpdateCheckRun(ctx, owner, repo, n, github.UpdateCheckRunOptions{
			ExternalID:  github.String(tr.GetSelfLink()),
			Name:        tr.Name,
			Status:      github.String(status),
			Conclusion:  github.String(conclusion),
			HeadSHA:     github.String(commit),
			Output:      output,
			CompletedAt: ghtime(tr.Status.CompletionTime),
			// TODO: Replace with Task-specific URL
			DetailsURL: github.String("https://tekton.dev"),
		})
		if err != nil {
			return nil, fmt.Errorf("CreateCheck: %w", err)
		}
		return cr, nil
	}

	// There's no existing CheckRun - create.
	cr, _, err := client.Checks.CreateCheckRun(ctx, tr.Annotations[key("owner")], tr.Annotations[key("repo")], github.CreateCheckRunOptions{
		ExternalID:  github.String(tr.GetSelfLink()),
		Name:        tr.GetNamespacedName().String(),
		Status:      github.String(status),
		Conclusion:  github.String(conclusion),
		HeadSHA:     tr.Annotations[key("commit")],
		Output:      output,
		StartedAt:   ghtime(tr.Status.StartTime),
		CompletedAt: ghtime(tr.Status.CompletionTime),
		// TODO: Replace with Task-specific URL
		DetailsURL: github.String("https://tekton.dev"),
	})
	if err != nil {
		return nil, fmt.Errorf("CreateCheck: %w", err)
	}
	return cr, nil
}

const (
	CheckRunStatusQueued     = "queued"
	CheckRunStatusInProgress = "in_progress"
	CheckRunStatusCompleted  = "completed"

	CheckRunConclusionSuccess        = "success"
	CheckRunConclusionFailure        = "failure"
	CheckRunConclusionCancelled      = "cancelled"
	CheckRunConclusionTimeout        = "timed_out"
	CheckRunConclusionActionRequired = "action_required"
)

func status(s v1beta1.TaskRunStatus) (status, conclusion string) {
	c := s.GetCondition(apis.ConditionSucceeded)
	if c == nil {
		return "", ""
	}

	switch c.Reason {
	case "Pending":
		return CheckRunStatusQueued, ""
	case "Started", "Running":
		return CheckRunStatusInProgress, ""
	case "Succeeded":
		return CheckRunStatusCompleted, CheckRunConclusionSuccess
	case "Failed":
		return CheckRunStatusCompleted, CheckRunConclusionFailure
	case "TaskRunCancelled":
		return CheckRunStatusCompleted, CheckRunConclusionCancelled
	case "TaskRunTimeout":
		return CheckRunStatusCompleted, CheckRunConclusionTimeout
	}

	if c.Status == v1.ConditionFalse {
		if s.CompletionTime == nil {
			return CheckRunStatusInProgress, ""
		}
		return CheckRunStatusCompleted, CheckRunConclusionActionRequired
	}

	return "", ""
}

func ghtime(t *metav1.Time) *github.Timestamp {
	if t == nil {
		return nil
	}
	return &github.Timestamp{Time: t.Time}
}
