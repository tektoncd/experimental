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

package clusterref

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tektoncd/experimental/remote-resolution/pkg/reconciler/framework"
	pipelineinformer "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1beta1/pipeline"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/client/listers/pipeline/v1beta1"
)

// clusterref.Resolver gets pipelines from the cluster it resides in
// and returns them in response to ResourceRequests. This just reproduces
// what the pipelines reconciler already does for a `pipelineRef`.
type Resolver struct {
	pipelineLister pipelinev1beta1.PipelineLister
}

func (r *Resolver) Initialize(ctx context.Context) error {
	r.pipelineLister = pipelineinformer.Get(ctx).Lister()
	return nil
}

func (r *Resolver) GetName() string {
	return "In-Cluster"
}

func (r *Resolver) GetSelector() map[string]string {
	return map[string]string{
		"resolution.tekton.dev/type": "in-cluster",
	}
}

func (r *Resolver) ValidateParams(params map[string]string) error {
	required := []string{
		"kind",
		"name",
		"namespace",
	}
	missing := []string{}
	if params == nil {
		missing = required
	} else {
		for _, p := range required {
			v, has := params[p]
			if !has || v == "" {
				missing = append(missing, p)
			}
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing %v", strings.Join(missing, ", "))
	}
	return nil
}

func (r *Resolver) Resolve(params map[string]string) (string, map[string]string, error) {
	resourceKind := params["kind"]
	resourceName := params["name"]
	resourceNamespace := params["namespace"]
	normalizedKind := strings.TrimSpace(strings.ToLower(resourceKind))

	// only supports pipelines at the moment.
	switch normalizedKind {
	case "pipeline":
		resourceContent, err := r.resolveInClusterPipeline(resourceNamespace, resourceName)
		if err != nil {
			return "", nil, err
		}
		annotations := map[string]string{"content-type": "application/json"}
		return resourceContent, annotations, nil
	default:
		return "", nil, fmt.Errorf("unsupported kind %q", normalizedKind)
	}
}

func (r *Resolver) resolveInClusterPipeline(pipelineNamespace, pipelineName string) (string, error) {
	p, err := r.pipelineLister.Pipelines(pipelineNamespace).Get(pipelineName)
	if err != nil {
		return "", fmt.Errorf("error requesting pipeline: %w", err)
	}
	// prune less important fields that can be very large
	p.ObjectMeta.ManagedFields = nil
	delete(p.ObjectMeta.Annotations, "kubectl.kubernetes.io/last-applied-configuration")
	pipelineJSON, err := json.Marshal(p)
	if err != nil {
		return "", fmt.Errorf("error serializing pipeline: %w", err)
	}
	return string(pipelineJSON), nil
}

var _ framework.Resolver = &Resolver{}
