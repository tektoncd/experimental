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
	"bytes"
	"context"
	"fmt"
	"io"
	"strconv"

	"github.com/google/go-github/v32/github"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/typed/pipeline/v1beta1"
	listers "github.com/tektoncd/pipeline/pkg/client/listers/pipeline/v1beta1"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// GitHubAppReconciler updates CheckRun results for PipelineRun outputs.
type GitHubAppReconciler struct {
	Logger             *zap.SugaredLogger
	TaskRunLister      listers.TaskRunLister
	InstallationClient func(installationID int64) *github.Client
	Kubernetes         kubernetes.Interface
	Tekton             tektonclient.TektonV1beta1Interface
}

// Reconcile creates or updates the check run.
func (r *GitHubAppReconciler) Reconcile(ctx context.Context, reconcileKey string) error {
	log := r.Logger.With(zap.String("key", reconcileKey))
	log.Infof("reconciling resource")

	namespace, name, err := cache.SplitMetaNamespaceKey(reconcileKey)
	if err != nil {
		log.Errorf("invalid resource key: %s", reconcileKey)
		return nil
	}

	// Get the Task Run resource with this namespace/name
	tr, err := r.TaskRunLister.TaskRuns(namespace).Get(name)
	if err != nil {
		log.Errorf("Error retrieving TaskRun: %v", err)
		return err
	}
	log = log.With(zap.String("uid", string(tr.UID)))

	log.Info("Sending update")

	// Create new GitHub client from existing App Transport + Installation
	id := tr.Annotations[key("installation")]
	if id == "" {
		log.Info("not a GitHub App task, skipping")
		return nil
	}
	n, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return err
	}
	gh := r.InstallationClient(n)

	// Render out GitHub CheckRun output.
	body, err := render(tr)
	if err != nil {
		log.Errorf("error rendering TaskRun: %v", err)
		return err
	}
	logs, err := getLogs(ctx, r.Kubernetes, tr)
	if err != nil {
		log.Errorf("get logs: %v", err)
		return err
	}

	// Update or create the CheckRun.
	cr, err := UpsertCheckRun(ctx, gh, tr, &github.CheckRunOutput{
		Title:   github.String(tr.Name),
		Summary: github.String(body),
		Text:    github.String(logs),
	})
	if err != nil {
		log.Errorf("UpsertCheckRun: %v", err)
		return err
	}

	// Update TaskRun with CheckRun ID so that we can determine if there's an
	// existing CheckRun for the TaskRun in future updates.
	// TODO: Prevent a 2nd round of reconciliation for this annotation update?
	if id := strconv.FormatInt(cr.GetID(), 10); id != tr.Annotations[key("checkrun")] {
		tr.Annotations[key("checkrun")] = id
		if _, err := r.Tekton.TaskRuns(tr.GetNamespace()).Update(tr); err != nil {
			log.Errorf("TaskRun.Update: %v", err)
			return err
		}
	}
	return nil
}

func getLogs(ctx context.Context, client kubernetes.Interface, tr *v1beta1.TaskRun) (string, error) {
	pod, err := client.CoreV1().Pods(tr.Namespace).Get(tr.Status.PodName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	b := new(bytes.Buffer)
	for _, c := range pod.Spec.Containers {
		b.WriteString(fmt.Sprintf("# %s\n```\n", c.Name))
		rc, err := client.CoreV1().Pods(tr.Namespace).GetLogs(tr.Status.PodName, &corev1.PodLogOptions{Container: c.Name}).Stream()
		if err != nil {
			return "", err
		}
		defer rc.Close()
		if _, err := io.Copy(b, rc); err != nil {
			return "", err
		}
		b.WriteString("\n```\n")

	}
	return b.String(), err
}

func key(s string) string {
	return fmt.Sprintf("github.integrations.tekton.dev/%s", s)
}
