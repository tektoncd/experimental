/*
Copyright 2022 The Tekton Authors

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

package trustedtask

import (
	"context"
	"os"

	"github.com/tektoncd/pipeline/pkg/apis/config"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"github.com/tektoncd/pipeline/pkg/reconciler/pipelinerun/resources"
	taskrun "github.com/tektoncd/pipeline/pkg/reconciler/taskrun/resources"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"knative.dev/pkg/apis"
	kubeclient "knative.dev/pkg/client/injection/kube/client"
	"knative.dev/pkg/logging"
)

//go:generate deepcopy-gen -O zz_generated.deepcopy --go-header-file ./../../hack/boilerplate/boilerplate.go.txt  -i ./
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TrustedPipelineRun wraps the PipelineRun and verify if it is tampered or not.
type TrustedPipelineRun struct {
	v1beta1.PipelineRun
}

// Verify that TrustedPipelineRun adheres to the appropriate interfaces.
var (
	_ apis.Defaultable = (*TrustedPipelineRun)(nil)
	_ apis.Validatable = (*TrustedPipelineRun)(nil)
)

// Validate the PipelineRunRef referred pipeline and task are tampered or not.
func (pr *TrustedPipelineRun) Validate(ctx context.Context) (errs *apis.FieldError) {
	// TODO: validate only on create operation.

	k8sclient := kubeclient.Get(ctx)
	config, err := rest.InClusterConfig()
	if err != nil {
		return apis.ErrGeneric(err.Error())
	}
	tektonClient, err := versioned.NewForConfig(config)
	if err != nil {
		return apis.ErrGeneric(err.Error())
	}

	if errs := errs.Also(pr.verifyPipelineRun(ctx, k8sclient, tektonClient)); errs != nil {
		return errs
	}
	return nil
}

// SetDefaults is not used.
func (pr *TrustedPipelineRun) SetDefaults(ctx context.Context) {
}

func (pr *TrustedPipelineRun) verifyPipelineRun(
	ctx context.Context,
	k8sclient kubernetes.Interface,
	tektonClient versioned.Interface,
) (errs *apis.FieldError) {
	logger := logging.FromContext(ctx)
	logger.Info("Verifying PipelineRun")

	serviceAccountName := os.Getenv("WEBHOOK_SERVICEACCOUNT_NAME")
	if serviceAccountName == "" {
		serviceAccountName = "tekton-verify-task-webhook"
	}

	// for inline pipeline we verify the tasks
	if pr.Spec.PipelineRef == nil {
		for _, t := range pr.Spec.PipelineSpec.Tasks {
			if t.TaskRef == nil {
				continue
			}

			fn, err := taskrun.GetTaskFunc(ctx, k8sclient, tektonClient, t.TaskRef, pr.Namespace, serviceAccountName)
			if err != nil {
				return apis.ErrGeneric(err.Error(), "spec", "taskRef")
			}

			if err := verifyTask(ctx, t.TaskRef.Name, k8sclient, fn); err != nil {
				return apis.ErrGeneric(err.Error(), "spec", "taskRef")
			}
		}
		return nil
	}

	cfg := config.FromContextOrDefaults(ctx)
	cfg.FeatureFlags.EnableTektonOCIBundles = true
	ctx = config.ToContext(ctx, cfg)

	fn, err := resources.GetPipelineFunc(ctx, k8sclient, tektonClient, &pr.PipelineRun)
	if err != nil {
		return apis.ErrGeneric(err.Error(), "spec", "PipelineRef")
	}
	resolvedPipeline, err := fn(ctx, pr.Spec.PipelineRef.Name)
	if err != nil {
		return apis.ErrGeneric(err.Error(), "spec", "PipelineRef")
	}

	pm, signature, err := prepareObjectMeta(resolvedPipeline.PipelineMetadata())
	if err != nil {
		return apis.ErrGeneric(err.Error(), "spec", "PipelineRef")
	}

	pipeline := v1beta1.Pipeline{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "tekton.dev/v1beta1",
			Kind:       "Pipeline"},
		ObjectMeta: pm,
		Spec:       resolvedPipeline.PipelineSpec(),
	}

	v, err := verifier(ctx, pipeline.ObjectMeta.Annotations, k8sclient)
	if err != nil {
		return apis.ErrGeneric(err.Error(), "PipelineRef")
	}

	if err := VerifyInterface(ctx, pipeline, v, signature); err != nil {
		return apis.ErrGeneric(err.Error(), "PipelineRef")
	}

	for _, t := range pipeline.Spec.Tasks {
		if t.TaskRef == nil {
			continue
		}

		fn, err := taskrun.GetTaskFunc(ctx, k8sclient, tektonClient, t.TaskRef, pr.Namespace, serviceAccountName)
		if err != nil {
			return apis.ErrGeneric(err.Error(), "spec", "taskRef")
		}

		if err := verifyTask(ctx, t.TaskRef.Name, k8sclient, fn); err != nil {
			return apis.ErrGeneric(err.Error(), "spec", "taskRef")
		}

	}

	return nil
}
