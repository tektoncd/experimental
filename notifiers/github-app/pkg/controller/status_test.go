package controller

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-github/v32/github"
)

func TestHandleStatus(t *testing.T) {
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	client := github.NewClient(srv.Client())
	client.BaseURL = mustParseURL(srv.URL + "/")

	r := &GitHubAppReconciler{
		GitHub: NewStatic(client),
	}

	ctx := context.Background()
	tr := taskrun("testdata/taskrun.yaml")

	mux.HandleFunc("/repos/tektoncd/test/statuses/db165c3a71dc45d096aebd0f49f07ec565ad1e08",
		validateStatus(t, &github.RepoStatus{
			State:       github.String(StateSuccess),
			Description: github.String("All Steps have completed executing"),
			TargetURL:   github.String(dashboardURL(tr)),
			Context:     github.String("echo-6b4fn-echo-xrxq4"),
		}),
	)
	if err := r.HandleStatus(ctx, tr); err != nil {
		t.Fatalf("HandleStatus: %v", err)
	}
}

func validateStatus(t *testing.T, want *github.RepoStatus) func(rw http.ResponseWriter, r *http.Request) {
	t.Helper()

	return func(rw http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("error reading HTTP body: %v", err)
		}
		got := new(github.RepoStatus)
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

func TestDashboardURL(t *testing.T) {
	want := "https://dashboard.dogfooding.tekton.dev/#/namespaces/default/taskruns/echo-6b4fn-echo-xrxq4"
	got := dashboardURL(taskrun("testdata/taskrun.yaml"))
	if want != got {
		t.Errorf("want: %s, got: %s", want, got)
	}
}
