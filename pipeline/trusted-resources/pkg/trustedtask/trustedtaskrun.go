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
	"encoding/base64"
	"fmt"
	"os"

	"github.com/tektoncd/pipeline/pkg/apis/config"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"github.com/tektoncd/pipeline/pkg/reconciler/taskrun/resources"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"knative.dev/pkg/apis"
	kubeclient "knative.dev/pkg/client/injection/kube/client"
	"knative.dev/pkg/logging"
)

const (
	SignatureAnnotation = "tekton.dev/signature"
)

//go:generate deepcopy-gen -O zz_generated.deepcopy --go-header-file ./../../hack/boilerplate/boilerplate.go.txt  -i ./
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TrustedTaskRun wraps the TaskRun and verify if it is tampered or not.
type TrustedTaskRun struct {
	v1beta1.TaskRun
}

// Verify that TrustedTaskRun adheres to the appropriate interfaces.
var (
	_ apis.Defaultable = (*TrustedTaskRun)(nil)
	_ apis.Validatable = (*TrustedTaskRun)(nil)
)

// Validate the Taskref referred task is tampered or not.
func (tr *TrustedTaskRun) Validate(ctx context.Context) (errs *apis.FieldError) {
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

	if errs := errs.Also(tr.verifyTaskRun(ctx, k8sclient, tektonClient)); errs != nil {
		return errs
	}
	return nil
}

// SetDefaults is not used.
func (tr *TrustedTaskRun) SetDefaults(ctx context.Context) {
}

func (tr *TrustedTaskRun) verifyTaskRun(
	ctx context.Context,
	k8sclient kubernetes.Interface,
	tektonClient versioned.Interface,
) (errs *apis.FieldError) {
	logger := logging.FromContext(ctx)
	logger.Info("Verifying TaskRun")

	// We focus on verifying the task in taskref not the whole taskrun.
	if tr.Spec.TaskRef == nil {
		return nil
	}

	serviceAccountName := os.Getenv("WEBHOOK_SERVICEACCOUNT_NAME")
	if serviceAccountName == "" {
		serviceAccountName = "tekton-verify-task-webhook"
	}

	// TODO: figure out how to config this.
	cfg := config.FromContextOrDefaults(ctx)
	cfg.FeatureFlags.EnableTektonOCIBundles = true
	ctx = config.ToContext(ctx, cfg)

	fn, err := resources.GetTaskFunc(ctx, k8sclient, tektonClient, tr.Spec.TaskRef, tr.Namespace, serviceAccountName)
	if err != nil {
		return apis.ErrGeneric(err.Error(), "spec", "taskRef")
	}

	if err := verifyTask(ctx, tr.Spec.TaskRef.Name, k8sclient, fn); err != nil {
		return apis.ErrGeneric(err.Error(), "spec", "taskRef")
	}
	return nil
}

func verifyTask(ctx context.Context, taskRefName string, k8sclient kubernetes.Interface, fn resources.GetTask) error {
	resolvedTask, err := fn(ctx, taskRefName)
	if err != nil {
		return err
	}

	tm, signature, err := prepareObjectMeta(resolvedTask.TaskMetadata())
	if err != nil {
		return err
	}

	task := v1beta1.Task{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "tekton.dev/v1beta1",
			Kind:       "Task"},
		ObjectMeta: tm,
		Spec:       resolvedTask.TaskSpec(),
	}

	verifier, err := verifier(ctx, task.ObjectMeta.Annotations)
	if err != nil {
		return err
	}

	if err := VerifyInterface(ctx, task, verifier, signature); err != nil {
		return err
	}
	return nil

}

// prepareObjectMeta will filter the objectmeta and extract the signature.
func prepareObjectMeta(in metav1.ObjectMeta) (metav1.ObjectMeta, []byte, error) {
	out := metav1.ObjectMeta{}

	// exclude the fields populated by system.
	out.Name = in.Name
	out.GenerateName = in.GenerateName
	out.Namespace = in.Namespace

	if in.Labels != nil {
		out.Labels = make(map[string]string)
		for k, v := range in.Labels {
			out.Labels[k] = v
		}
	}

	out.Annotations = make(map[string]string)
	for k, v := range in.Annotations {
		out.Annotations[k] = v
	}

	// exclude the annotations added by other components
	delete(out.Annotations, "kubectl-client-side-apply")
	delete(out.Annotations, "kubectl.kubernetes.io/last-applied-configuration")

	// signature should be contained in annotation
	sig, ok := in.Annotations[SignatureAnnotation]
	if !ok {
		return out, nil, fmt.Errorf("signature is missing")
	}

	// extract signature
	signature, err := base64.StdEncoding.DecodeString(sig)
	if err != nil {
		return out, nil, err
	}
	delete(out.Annotations, SignatureAnnotation)

	return out, signature, nil
}
