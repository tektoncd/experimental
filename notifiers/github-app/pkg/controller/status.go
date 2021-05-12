package controller

import (
	"context"
	"fmt"

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

	owner := tr.Annotations[key("owner")]
	repo := tr.Annotations[key("repo")]
	commit := tr.Annotations[key("commit")]

	var description *string
	if m := tr.GetStatusCondition().GetCondition(apis.ConditionSucceeded).GetMessage(); m != "" {
		description = github.String(m)
	}

	status := &github.RepoStatus{
		State:       state(tr.Status),
		Description: description,
		TargetURL:   github.String(dashboardURL(tr)),
		Context:     github.String(tr.GetName()),
	}
	_, _, err = client.Repositories.CreateStatus(ctx, owner, repo, commit, status)
	return err
}

func dashboardURL(tr *v1beta1.TaskRun) string {
	// TODO: generalize host, object type.
	return fmt.Sprintf("https://dashboard.dogfooding.tekton.dev/#/namespaces/%s/taskruns/%s", tr.GetNamespace(), tr.GetName())
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
