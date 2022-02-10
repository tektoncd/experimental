package controller

import (
	"context"

	"github.com/google/go-github/v32/github"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/pod"
	"knative.dev/pkg/apis"
)

func (r *GitHubAppReconciler) HandleStatus(ctx context.Context, tr *v1beta1.TaskRun) error {
	client, err := r.GitHub.NewClient("")
	if err != nil {
		return err
	}

	ghMetadata, err := getStatusMetadata(tr)
	if err != nil {
		return err
	}

	name, err := nameFor(tr)
	if err != nil {
		return err
	}

	url, err := dashboardURL(tr)

	if err != nil {
		return err
	}

	status := &github.RepoStatus{
		State:       state(tr.Status),
		Description: truncateDesc(tr.GetStatusCondition().GetCondition(apis.ConditionSucceeded).GetMessage()),
		TargetURL:   github.String(url),
		Context:     github.String(name),
	}
	_, _, err = client.Repositories.CreateStatus(ctx, ghMetadata["owner"], ghMetadata["repo"], ghMetadata["commit"], status)
	return err
}

// truncateDesc truncates a given string to fit within GitHub status character
// limits (140 chars).
func truncateDesc(m string) *string {
	if m == "" {
		return nil
	}
	if len(m) > 140 {
		m = (m)[:137] + "..."
	}
	return &m
}

const (
	StatePending = "pending"
	StateSuccess = "success"
	StateError   = "error"
	StateFailure = "failure"
)

//pending, success, error, or failure.
func state(s v1beta1.TaskRunStatus) *string {
	c := s.GetCondition(apis.ConditionSucceeded)
	if c == nil {
		return github.String(StatePending)
	}

	switch v1beta1.TaskRunReason(c.Reason) {
	case pod.ReasonPending, v1beta1.TaskRunReasonStarted, v1beta1.TaskRunReasonRunning:
		return github.String(StatePending)
	case v1beta1.TaskRunReasonSuccessful:
		return github.String(StateSuccess)
	case v1beta1.TaskRunReasonFailed, v1beta1.TaskRunReasonCancelled, v1beta1.TaskRunReasonTimedOut:
		return github.String(StateFailure)
	default:
		return github.String(StatePending)
	}
}
