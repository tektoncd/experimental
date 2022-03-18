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
	secretPath          = "/etc/signing-secrets/cosign.pub"
	signingConfigMap    = "config-trusted-resources"
	SignatureAnnotation = "tekton.dev/signature"
	kmsAnnotation       = "tekton.dev/kms"
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

// Validate the TaskRun is tampered or not.
func (tr *TrustedTaskRun) Validate(ctx context.Context) (errs *apis.FieldError) {
	if !apis.IsInCreate(ctx) {
		return nil
	}

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

	if tr.ObjectMeta.Annotations == nil {
		return apis.ErrMissingField("annotations")
	}

	if tr.ObjectMeta.Annotations[SignatureAnnotation] == "" {
		return apis.ErrMissingField(fmt.Sprintf("annotations[%s]", SignatureAnnotation))
	}

	cp, signature, err := copyTaskRun(tr)
	if err != nil {
		return apis.ErrGeneric(err.Error(), "metadata")
	}

	verifier, err := verifier(ctx, cp.ObjectMeta.Annotations, k8sclient)
	if err != nil {
		return apis.ErrGeneric(err.Error(), "metadata")
	}


	if err := VerifyInterface(ctx, cp, verifier, signature); err != nil {
		return apis.ErrGeneric(err.Error(), "taskrun")
	}

	if tr.Spec.TaskRef != nil {
		serviceAccountName := os.Getenv("WEBHOOK_SERVICEACCOUNT_NAME")
		if serviceAccountName == "" {
			serviceAccountName = "tekton-verify-task-webhook"
		}

		getfunc, err := resources.GetTaskFunc(ctx, k8sclient, tektonClient, tr.Spec.TaskRef, tr.Namespace, serviceAccountName)
		if err != nil {
			return apis.ErrGeneric(err.Error(), "spec", "taskRef")
		}

		actualTask, err := getfunc(ctx, tr.Spec.TaskRef.Name)
		if err != nil {
			return apis.ErrGeneric(err.Error(), "spec", "taskRef")
		}

		trustedTask := copyTask(actualTask)
		return trustedTask.verifyTask(ctx, k8sclient, tektonClient)
	}

	return nil
}

func copyTask(t v1beta1.TaskObject) TrustedTask {
	task := v1beta1.Task{}
	trustedTask := TrustedTask{}
	task.TypeMeta = metav1.TypeMeta{
		APIVersion: "tekton.dev/v1beta1",
		Kind:       "Task"}
	task.Spec = t.TaskSpec()
	task.ObjectMeta = t.TaskMetadata()

	trustedTask.Task = task
	trustedTask.Task = task
	return trustedTask
}

func copyTaskRun(in *TrustedTaskRun) (v1beta1.TaskRun, []byte, error) {
	c := v1beta1.TaskRun{}
	c.TypeMeta = in.TypeMeta
	c.SetName(in.Name)
	c.SetGenerateName(in.GenerateName)
	c.SetNamespace(in.Namespace)

	// Question: do we include labels?
	/*
		c.Labels = make(map[string]string)
		for k, v := range in.Labels {
			c.Labels[k] = v
		}*/
	c.Annotations = make(map[string]string)
	for k, v := range in.Annotations {
		c.Annotations[k] = v
	}
	delete(c.ObjectMeta.Annotations, "kubectl.kubernetes.io/last-applied-configuration")

	c.Spec = in.Spec
	if c.Spec.Timeout != nil {
		c.Spec.Timeout = nil
	}

	signature, err := base64.StdEncoding.DecodeString(c.ObjectMeta.Annotations[SignatureAnnotation])
	if err != nil {
		return c, signature, err
	}

	delete(c.ObjectMeta.Annotations, SignatureAnnotation)

	return c, signature, nil
}
