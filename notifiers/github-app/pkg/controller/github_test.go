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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-github/v32/github"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
	duck "knative.dev/pkg/apis/duck/v1beta1"
)

func TestUpsertCheckRun(t *testing.T) {
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	client := github.NewClient(srv.Client())
	client.BaseURL = mustParseURL(srv.URL + "/")

	ctx := context.Background()
	tr := taskrun()
	output := &github.CheckRunOutput{
		Summary: github.String("foo"),
	}

	t.Run("Create", func(t *testing.T) {
		mux.HandleFunc("/repos/tektoncd/test/check-runs", validateCheckRun(t, &github.CheckRun{
			Name:        github.String("default/echo-6b4fn-echo-xrxq4"),
			HeadSHA:     github.String("db165c3a71dc45d096aebd0f49f07ec565ad1e08"),
			ExternalID:  github.String("/apis/tekton.dev/v1beta1/namespaces/default/taskruns/echo-6b4fn-echo-xrxq4"),
			DetailsURL:  github.String("https://tekton.dev"),
			Status:      github.String("completed"),
			Conclusion:  github.String("success"),
			StartedAt:   &github.Timestamp{Time: time.Date(2020, 8, 27, 15, 21, 37, 0, time.FixedZone("Z", 0))},
			CompletedAt: &github.Timestamp{Time: time.Date(2020, 8, 27, 15, 21, 46, 0, time.FixedZone("Z", 0))},
			Output:      output,
		}))
		if _, err := UpsertCheckRun(ctx, client, tr, output); err != nil {
			t.Fatalf("UpsertCheckRun: %v", err)
		}
	})

	t.Run("Update", func(t *testing.T) {
		tr.Annotations[key("checkrun")] = "1234"
		mux.HandleFunc("/repos/tektoncd/test/check-runs/1234", validateCheckRun(t, &github.CheckRun{
			Name:        github.String("echo-6b4fn-echo-xrxq4"),
			HeadSHA:     github.String("db165c3a71dc45d096aebd0f49f07ec565ad1e08"),
			ExternalID:  github.String("/apis/tekton.dev/v1beta1/namespaces/default/taskruns/echo-6b4fn-echo-xrxq4"),
			DetailsURL:  github.String("https://tekton.dev"),
			Status:      github.String("completed"),
			Conclusion:  github.String("success"),
			CompletedAt: &github.Timestamp{Time: time.Date(2020, 8, 27, 15, 21, 46, 0, time.FixedZone("Z", 0))},
			Output:      output,
		}))
		if _, err := UpsertCheckRun(ctx, client, tr, output); err != nil {
			t.Fatalf("UpsertCheckRun: %v", err)
		}
	})
}

func mustParseURL(s string) *url.URL {
	u, err := url.Parse(s)
	if err != nil {
		panic(fmt.Errorf("error parsing URL %s: %v", s, err))
	}
	return u
}

func validateCheckRun(t *testing.T, want *github.CheckRun) func(rw http.ResponseWriter, r *http.Request) {
	t.Helper()

	return func(rw http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("error reading HTTP body: %v", err)
		}
		got := new(github.CheckRun)
		if err := json.Unmarshal(body, got); err != nil {
			t.Fatalf("error unmarshalling HTTP body: %v", err)
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("-want,+got: %s", diff)
		}
		enc := json.NewEncoder(rw)
		if err := enc.Encode(got); err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(err.Error()))
		}
	}
}

func TestGitHubStatus(t *testing.T) {
	// Test cases pulled from https://github.com/tektoncd/pipeline/blob/master/docs/taskruns.md#monitoring-execution-status
	for _, tc := range []struct {
		condStatus     corev1.ConditionStatus
		reason         string
		completionTime bool

		status, conclusion string
	}{
		{
			condStatus: corev1.ConditionUnknown,
			reason:     v1beta1.TaskRunReasonStarted.String(),
			status:     CheckRunStatusInProgress,
		},
		{
			condStatus: corev1.ConditionUnknown,
			// Exists in documentation, but not Tekton const.
			reason: "Pending",
			status: CheckRunStatusQueued,
		},
		{
			condStatus: corev1.ConditionUnknown,
			reason:     v1beta1.TaskRunReasonRunning.String(),
			status:     CheckRunStatusInProgress,
		},
		{
			condStatus: corev1.ConditionUnknown,
			reason:     v1beta1.TaskRunReasonCancelled.String(),
			status:     CheckRunStatusCompleted,
			conclusion: CheckRunConclusionCancelled,
		},
		{
			condStatus: corev1.ConditionFalse,
			reason:     v1beta1.TaskRunReasonCancelled.String(),
			status:     CheckRunStatusCompleted,
			conclusion: CheckRunConclusionCancelled,
		},
		{
			condStatus: corev1.ConditionTrue,
			reason:     v1beta1.TaskRunReasonSuccessful.String(),
			status:     CheckRunStatusCompleted,
			conclusion: CheckRunConclusionSuccess,
		},
		{
			condStatus: corev1.ConditionTrue,
			reason:     v1beta1.TaskRunReasonFailed.String(),
			status:     CheckRunStatusCompleted,
			conclusion: CheckRunConclusionFailure,
		},
		{
			condStatus: corev1.ConditionFalse,
			reason:     "non-permanent error",
			status:     CheckRunStatusInProgress,
		},
		{
			condStatus:     corev1.ConditionFalse,
			reason:         "permanent error",
			completionTime: true,
			status:         CheckRunStatusCompleted,
			conclusion:     CheckRunConclusionActionRequired,
		},
		{
			condStatus: corev1.ConditionFalse,
			reason:     v1beta1.TaskRunReasonTimedOut.String(),
			status:     CheckRunStatusCompleted,
			conclusion: CheckRunConclusionTimeout,
		},
	} {
		t.Run(fmt.Sprintf("%s_%s", tc.condStatus, tc.reason), func(t *testing.T) {
			s := v1beta1.TaskRunStatus{
				Status: duck.Status{
					Conditions: []apis.Condition{{
						Type:   apis.ConditionSucceeded,
						Reason: tc.reason,
						Status: tc.condStatus,
					}},
				},
			}
			if tc.completionTime {
				s.TaskRunStatusFields = v1beta1.TaskRunStatusFields{
					CompletionTime: &metav1.Time{Time: time.Now()},
				}
			}
			status, conclusion := status(s)
			if tc.status != status {
				t.Errorf("status: want %s, got %s", tc.status, status)
			}
			if tc.conclusion != conclusion {
				t.Errorf("conclusion: want %s, got %s", tc.conclusion, conclusion)
			}
		})
	}
}
