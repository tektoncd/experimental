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
	"bytes"
	"context"
	"crypto"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/authn/k8schain"
	imgname "github.com/google/go-containerregistry/pkg/name"
	ociremote "github.com/google/go-containerregistry/pkg/v1/remote"
	cosignsignature "github.com/sigstore/cosign/pkg/signature"
	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/sigstore/sigstore/pkg/signature/kms"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"knative.dev/pkg/apis"
	kubeclient "knative.dev/pkg/client/injection/kube/client"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/system"
	"knative.dev/pkg/webhook/json"
)

const (
	secretPath          = "/etc/signing-secrets/cosign.pub"
	signgingConfigMap   = "signing-secret-path"
	signatureAnnotation = "tekton.dev/signature"
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
	k8sclient := kubeclient.Get(ctx)
	config, err := rest.InClusterConfig()
	if err != nil {
		return apis.ErrGeneric(err.Error())
	}
	tektonClient, err := versioned.NewForConfig(config)
	if err != nil {
		return apis.ErrGeneric(err.Error())
	}
	if errs := errs.Also(tr.verifyTask(ctx, k8sclient, tektonClient)); errs != nil {
		return errs
	}
	return nil
}

// SetDefaults is not used.
func (tr *TrustedTaskRun) SetDefaults(ctx context.Context) {
}

func (tr *TrustedTaskRun) verifyTask(
	ctx context.Context,
	k8sclient kubernetes.Interface,
	tektonClient versioned.Interface,
) (errs *apis.FieldError) {
	logger := logging.FromContext(ctx)
	logger.Info("Verifying Resources")

	if tr.ObjectMeta.Annotations == nil {
		return apis.ErrMissingField("annotations")
	}

	if tr.ObjectMeta.Annotations[signatureAnnotation] == "" {
		return apis.ErrMissingField(fmt.Sprintf("annotations[%s]", signatureAnnotation))
	}

	signature, err := base64.StdEncoding.DecodeString(tr.ObjectMeta.Annotations[signatureAnnotation])
	if err != nil {
		return apis.ErrGeneric(err.Error(), "metadata")
	}

	verifier, err := verifier(ctx, tr.ObjectMeta.Annotations, k8sclient)
	if err != nil {
		return apis.ErrGeneric(err.Error(), "metadata")
	}

	if tr.Spec.TaskSpec != nil {
		logger.Info("Verifying TaskSpec")
		if err := verifyTaskSpec(ctx, tr.Spec.TaskSpec, verifier, signature); err != nil {
			return apis.ErrGeneric(err.Error(), "spec")
		}
		return nil
	}

	if tr.Spec.TaskRef != nil {
		if tr.Spec.TaskRef.Bundle != "" {
			logger.Info("Verifying OCI Bundle")
			if err := verifyTaskOCIBundle(ctx, tr.Spec.TaskRef.Bundle, verifier, signature, k8sclient); err != nil {
				return apis.ErrGeneric(err.Error(), "spec", "taskRef")
			}
			return nil
		}

		ts, err := tektonClient.TektonV1beta1().Tasks(tr.Namespace).Get(ctx, tr.Spec.TaskRef.Name, metav1.GetOptions{})
		if err != nil {
			return apis.ErrGeneric(err.Error(), "spec", "taskRef")
		}
		if ts.Name != "" {
			logger.Info("Verifying TaskRef")
			if err := verifyTaskSpec(ctx, &ts.Spec, verifier, signature); err != nil {
				if err != nil {
					return apis.ErrGeneric(err.Error(), "spec", "taskRef")
				}
			}
			return nil
		}
	}

	return nil
}

func verifier(
	ctx context.Context,
	annotations map[string]string,
	k8sclient kubernetes.Interface,
) (signature.Verifier, error) {
	if annotations[kmsAnnotation] != "" {
		// Fetch key from kms.
		return kms.Get(ctx, annotations[kmsAnnotation], crypto.SHA256)
	} else {
		cosignPublicKeypath := secretPath
		// Overwrite the path if set in configmap.
		cm, err := k8sclient.CoreV1().ConfigMaps(system.Namespace()).Get(ctx, signgingConfigMap, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		if cm.Data["path"] != "" {
			cosignPublicKeypath = cm.Data["path"]
		}
		return cosignsignature.LoadPublicKey(ctx, cosignPublicKeypath)
	}
}

func verifyTaskSpec(
	ctx context.Context,
	taskspec *v1beta1.TaskSpec,
	verifier signature.Verifier,
	signature []byte,
) (errs *apis.FieldError) {
	ts, err := json.Marshal(taskspec)
	if err != nil {
		return apis.ErrGeneric(err.Error(), "taskSpec")
	}

	h := sha256.New()
	h.Write(ts)

	if err := verifier.VerifySignature(bytes.NewReader(signature), bytes.NewReader(h.Sum(nil))); err != nil {
		return apis.ErrGeneric(err.Error(), "taskSpec")
	}

	return nil
}

func verifyTaskOCIBundle(
	ctx context.Context,
	bundle string,
	verifier signature.Verifier,
	signature []byte,
	k8sclient kubernetes.Interface,
) (errs *apis.FieldError) {

	serviceAccountName := os.Getenv("WEBHOOK_SERVICEACCOUNT_NAME")
	if serviceAccountName == "" {
		serviceAccountName = "tekton-verify-task-webhook"
	}
	kc, err := k8schain.New(ctx, k8sclient, k8schain.Options{
		Namespace:          system.Namespace(),
		ServiceAccountName: serviceAccountName,
	})
	if err != nil {
		return apis.ErrGeneric(err.Error()).ViaKey(bundle)
	}

	digest, err := digest(ctx, bundle, kc)
	if err != nil {
		return apis.ErrGeneric(err.Error()).ViaKey(bundle)
	}

	if err := verifier.VerifySignature(bytes.NewReader(signature), bytes.NewReader([]byte(digest))); err != nil {
		return apis.ErrGeneric(err.Error()).ViaKey(bundle)
	}

	return nil
}

func digest(ctx context.Context, imageReference string, keychain authn.Keychain) (string, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Second*60)
	defer cancel()

	imgRef, err := imgname.ParseReference(imageReference)
	if err != nil {
		return "", err
	}

	img, err := ociremote.Image(imgRef, ociremote.WithAuthFromKeychain(keychain), ociremote.WithContext(timeoutCtx))
	if err != nil {
		return "", err
	}

	dgst, err := img.Digest()
	if err != nil {
		return "", err
	}
	return dgst.String(), nil
}
