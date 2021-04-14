/*
Copyright 2020 The Tekton Authors

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

package controller

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-github/v32/github"
	faketekton "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/fake"
	informers "github.com/tektoncd/pipeline/pkg/client/informers/externalversions"
	"go.uber.org/zap/zaptest"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakek8s "k8s.io/client-go/kubernetes/fake"
)

func TestGitHubAppReconciler_Reconcile(t *testing.T) {
	mux := http.NewServeMux()

	// Create Fake GitHub client.
	called := false
	mux.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		called = true

		defer r.Body.Close()
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("error reading HTTP body: %v", err)
		}
		cr := new(github.CheckRun)
		if err := json.Unmarshal(body, cr); err != nil {
			t.Fatalf("error unmarshalling HTTP body: %v", err)
		}

		// Simulate CheckRun creation
		if cr.ID == nil {
			cr.ID = github.Int64(1234)
		}

		enc := json.NewEncoder(rw)
		if err := enc.Encode(cr); err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(err.Error()))
		}
	})
	srv := httptest.NewServer(mux)
	ghclient := github.NewClient(srv.Client())
	ghclient.BaseURL = mustParseURL(srv.URL + "/")

	// Create fake k8s/tekton clients.
	tr := taskrun("testdata/taskrun.yaml")
	tekton := faketekton.NewSimpleClientset(tr)
	informer := informers.NewSharedInformerFactory(tekton, 0)
	informer.Tekton().V1beta1().TaskRuns().Informer().GetIndexer().Add(tr)
	k8s := fakek8s.NewSimpleClientset(&corev1.Pod{
		ObjectMeta: v1.ObjectMeta{
			Namespace: tr.Namespace,
			Name:      tr.Status.PodName,
		},
	})
	r := &GitHubAppReconciler{
		Logger:        zaptest.NewLogger(t).Sugar(),
		TaskRunLister: informer.Tekton().V1beta1().TaskRuns().Lister(),
		InstallationClient: func(installationID int64) *github.Client {
			return ghclient
		},
		Tekton:     tekton.TektonV1beta1(),
		Kubernetes: k8s,
	}

	ctx := context.Background()
	if err := r.Reconcile(ctx, tr.GetNamespacedName().String()); err != nil {
		t.Fatalf("GitHubAppReconciler.Reconcile() = %v", err)
	}
	if !called {
		t.Fatalf("no call made to github!")
	}

	tr, err := tekton.TektonV1beta1().TaskRuns(tr.Namespace).Get(tr.Name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("could not find TaskRun post-reconcile: %v", err)
	}
	if tr.Annotations[key("checkrun")] != "1234" {
		t.Fatalf("%s: want %s, got %s", key("checkrun"), "1234", tr.Annotations[key("checkrun")])
	}
}
