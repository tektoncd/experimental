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

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"knative.dev/pkg/apis"
	kubeclient "knative.dev/pkg/client/injection/kube/client"
	"knative.dev/pkg/logging"
)

//go:generate deepcopy-gen -O zz_generated.deepcopy --go-header-file ./../../hack/boilerplate/boilerplate.go.txt  -i ./
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TrustedTask wraps the Task and verify if it is tampered or not.
type TrustedTask struct {
	v1beta1.Task
}

// Verify that TrustedTask adheres to the appropriate interfaces.
var (
	_ apis.Defaultable = (*TrustedTask)(nil)
	_ apis.Validatable = (*TrustedTask)(nil)
)

// Validate the Task is tampered or not.
func (ts *TrustedTask) Validate(ctx context.Context) (errs *apis.FieldError) {
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

	if errs := errs.Also(ts.verifyTask(ctx, k8sclient, tektonClient)); errs != nil {
		return errs
	}
	return nil
}

// SetDefaults is not used.
func (tr *TrustedTask) SetDefaults(ctx context.Context) {
}

func (ts *TrustedTask) verifyTask(
	ctx context.Context,
	k8sclient kubernetes.Interface,
	tektonClient versioned.Interface,
) (errs *apis.FieldError) {
	logger := logging.FromContext(ctx)
	logger.Info("Verifying Task")

	if ts.ObjectMeta.Annotations == nil {
		return apis.ErrMissingField("annotations")
	}

	if ts.ObjectMeta.Annotations[SignatureAnnotation] == "" {
		return apis.ErrMissingField(fmt.Sprintf("annotations[%s]", SignatureAnnotation))
	}

	cp, signature, err := copyTrustedTask(ts)

	if err != nil {
		return apis.ErrGeneric(err.Error(), "metadata")
	}

	verifier, err := verifier(ctx, ts.ObjectMeta.Annotations, k8sclient)
	if err != nil {
		return apis.ErrGeneric(err.Error(), "metadata")
	}

	fmt.Println("")
	fmt.Println("task", cp)
	fmt.Println("task.Annotations", cp.Annotations)
	fmt.Println("task.Spec", cp.Spec)

	if err := VerifyInterface(ctx, cp, verifier, signature); err != nil {
		return apis.ErrGeneric(err.Error(), "task")
	}

	return nil
}

// copyTrustedTask will copy fields defined by users
func copyTrustedTask(in *TrustedTask) (v1beta1.Task, []byte, error) {
	c := v1beta1.Task{}
	c.TypeMeta = in.TypeMeta
	c.SetName(in.Name)
	c.SetGenerateName(in.GenerateName)
	c.SetNamespace(in.Namespace)

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

	c.Spec = *in.Spec.DeepCopy()

	signature, err := base64.StdEncoding.DecodeString(c.ObjectMeta.Annotations[SignatureAnnotation])
	if err != nil {
		return c, signature, err
	}
	delete(c.ObjectMeta.Annotations, SignatureAnnotation)

	return c, signature, nil
}
