package trustedtask

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/tektoncd/pipeline/pkg/apis/config"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"github.com/tektoncd/pipeline/pkg/client/injection/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"knative.dev/pkg/apis"
	kubeclient "knative.dev/pkg/client/injection/kube/client"
	"knative.dev/pkg/logging"
)

type TrustedTask struct {
	v1beta1.Task
}

func (t *TrustedTask) Validate(ctx context.Context) (errs *apis.FieldError) {
	k8sClient := kubeclient.Get(ctx)
	tektonClient := client.Get(ctx)

	if err := t.verifyTask(ctx, k8sClient, tektonClient); err != nil {
		return err
	}

	return nil
}

func (t *TrustedTask) verifyTask(
	ctx context.Context,
	k8sClient kubernetes.Interface,
	tektonClient versioned.Interface) (errs *apis.FieldError) {
	logger := logging.FromContext(ctx)
	logger.Info("Verifying Task")

	task := t.Task

	cfg := config.FromContextOrDefaults(ctx)
	cfg.FeatureFlags.EnableTektonOCIBundles = true
	ctx = config.ToContext(ctx, cfg)

	verifier, err := verifier(ctx)
	if err != nil {
		return apis.ErrGeneric(err.Error())
	}

	signature, err := getSignature(task.TaskMetadata())
	if err != nil {
		return apis.ErrGeneric(err.Error())
	}

	// Create an unsignedTask for the verification as the signature annotation will
	// not be needed for the signing computation
	unsignedTask := task.DeepCopy()
	delete(unsignedTask.Annotations, SignatureAnnotation)

	if err := VerifyInterface(ctx, unsignedTask, verifier, signature); err != nil {
		return err
	}

	return nil
}

// getSignature fetches the signature specified in Task, Pipeline
func getSignature(metadata metav1.ObjectMeta) ([]byte, error) {
	sig, ok := metadata.Annotations[SignatureAnnotation]
	if !ok {
		return nil, fmt.Errorf("signature is missing")
	}

	signature, err := base64.StdEncoding.DecodeString(sig)
	if err != nil {
		return nil, err
	}

	return signature, nil
}
