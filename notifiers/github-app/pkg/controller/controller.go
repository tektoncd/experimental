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
	Logger        *zap.SugaredLogger
	TaskRunLister listers.TaskRunLister
	GitHub        *GitHubClientFactory
	Kubernetes    kubernetes.Interface
	Tekton        tektonclient.TektonV1beta1Interface
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

	owner, repo, err := getRepoMetadata(tr)
	if owner == "" || repo == "" || err != nil {
		log.Info("no GitHub annotations found, skipping")
		return nil
	}

	log.Info("Sending update")

	// If no installation is associated, assume a non-GitHub App status.
	if id := tr.Annotations[key("installation")]; id == "" {
		return r.HandleStatus(ctx, tr)
	}
	// Create Check Run with GitHub App
	return r.HandleCheckRun(ctx, log, tr)
}

func getLogs(ctx context.Context, client kubernetes.Interface, tr *v1beta1.TaskRun) (string, error) {
	pod, err := client.CoreV1().Pods(tr.Namespace).Get(ctx, tr.Status.PodName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	b := new(bytes.Buffer)
	for _, c := range pod.Spec.Containers {
		b.WriteString(fmt.Sprintf("# %s\n```\n", c.Name))
		rc, err := client.CoreV1().Pods(tr.Namespace).GetLogs(tr.Status.PodName, &corev1.PodLogOptions{Container: c.Name}).Stream(ctx)
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
