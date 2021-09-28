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

package pipelineruns

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	clientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

type metadataPatch struct {
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
}

type taskRunStatusPatch struct {
	TaskSpec v1beta1.TaskSpec `json:"taskSpec"`
}

type pipelineRunSpecPatch struct {
	Status string `json:"status"`
}

type pipelineRunStatusPatch struct {
	PipelineSpec v1beta1.PipelineSpec `json:"pipelineSpec"`
}

// PatchResolvedTaskRun accepts a TaskRun with its task spec resolved and updates the
// stored resource with the labels, annotations and resolved spec.
func PatchResolvedTaskRun(ctx context.Context, kClient kubernetes.Interface, pClient clientset.Interface, tr *v1beta1.TaskRun) (*v1beta1.TaskRun, error) {
	metadataBytes, err := json.Marshal(map[string]metadataPatch{
		"metadata": {
			Labels:      tr.ObjectMeta.Labels,
			Annotations: tr.ObjectMeta.Annotations,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("error constructing metadata patch: %w", err)
	}

	statusBytes, err := json.Marshal(map[string]taskRunStatusPatch{
		"status": {
			TaskSpec: *tr.Status.TaskSpec,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("error constructing status patch: %w", err)
	}

	tr, err = pClient.TektonV1beta1().TaskRuns(tr.Namespace).Patch(ctx, tr.Name, types.MergePatchType, metadataBytes, metav1.PatchOptions{})
	if err != nil {
		return nil, fmt.Errorf("error patching metadata: %w", err)
	}

	tr, err = pClient.TektonV1beta1().TaskRuns(tr.Namespace).Patch(ctx, tr.Name, types.MergePatchType, statusBytes, metav1.PatchOptions{}, "status")
	if err != nil {
		return nil, fmt.Errorf("error patching status: %w", err)
	}

	return tr, nil
}

// PatchResolvedPipelineRun accepts a PipelineRun with its pipeline spec
// resolved and updates the stored resource with the labels, annotations
// and resolved spec. Any PipelineRun Pending status is removed as well.
func PatchResolvedPipelineRun(ctx context.Context, kClient kubernetes.Interface, pClient clientset.Interface, pr *v1beta1.PipelineRun) (*v1beta1.PipelineRun, error) {
	latestGeneration, err := patchPipelineRunMetadata(ctx, pClient, pr)
	if err != nil {
		return nil, fmt.Errorf("error patching metadata: %w", err)
	}

	latestGeneration, err = patchPipelineRunStatus(ctx, pClient, pr)
	if err != nil {
		return nil, fmt.Errorf("error patching status: %w", err)
	}
	if latestGeneration.Status.PipelineSpec == nil {
		return nil, fmt.Errorf("patching pipelinerun status failed for unknown reason")
	}

	latestGeneration, err = patchRemovePipelineRunPending(ctx, pClient, pr)
	if err != nil {
		return nil, fmt.Errorf("error patching spec: %w", err)
	}

	return latestGeneration, nil
}

func patchPipelineRunMetadata(ctx context.Context, pClient clientset.Interface, pr *v1beta1.PipelineRun) (*v1beta1.PipelineRun, error) {
	pipelinerunBytes, err := json.Marshal(map[string]interface{}{
		"metadata": &metadataPatch{
			Labels:      pr.ObjectMeta.Labels,
			Annotations: pr.ObjectMeta.Annotations,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("error constructing metadata patch: %w", err)
	}
	return pClient.TektonV1beta1().PipelineRuns(pr.Namespace).Patch(ctx, pr.Name, types.MergePatchType, pipelinerunBytes, metav1.PatchOptions{})
}

func patchPipelineRunStatus(ctx context.Context, pClient clientset.Interface, pr *v1beta1.PipelineRun) (*v1beta1.PipelineRun, error) {
	statusBytes, err := json.Marshal(map[string]pipelineRunStatusPatch{
		"status": {
			PipelineSpec: *pr.Status.PipelineSpec,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("error constructing status patch: %w", err)
	}
	return pClient.TektonV1beta1().PipelineRuns(pr.Namespace).Patch(ctx, pr.Name, types.MergePatchType, statusBytes, metav1.PatchOptions{}, "status")
}

func patchRemovePipelineRunPending(ctx context.Context, pClient clientset.Interface, pr *v1beta1.PipelineRun) (*v1beta1.PipelineRun, error) {
	if pr.Spec.Status == v1beta1.PipelineRunSpecStatusPending {
		removePendingBytes, err := json.Marshal([]map[string]string{{
			"op":   "remove",
			"path": "/spec/status",
		}})
		if err != nil {
			return nil, fmt.Errorf("error constructing pending removal patch: %w", err)
		}
		return pClient.TektonV1beta1().PipelineRuns(pr.Namespace).Patch(ctx, pr.Name, types.JSONPatchType, removePendingBytes, metav1.PatchOptions{})
	}
	return pr, nil
}
