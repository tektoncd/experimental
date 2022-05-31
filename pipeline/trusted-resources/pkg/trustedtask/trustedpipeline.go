package trustedtask

import (
	"context"

	"github.com/tektoncd/pipeline/pkg/apis/config"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"github.com/tektoncd/pipeline/pkg/client/injection/client"
	"k8s.io/client-go/kubernetes"
	"knative.dev/pkg/apis"
	kubeclient "knative.dev/pkg/client/injection/kube/client"
	"knative.dev/pkg/logging"
)

type TrustedPipeline struct {
	v1beta1.Pipeline
}

func (p *TrustedPipeline) Validate(ctx context.Context) (errs *apis.FieldError) {
	k8sClient := kubeclient.Get(ctx)
	tektonClient := client.Get(ctx)

	if err := p.verifyPipeline(ctx, k8sClient, tektonClient); err != nil {
		return err
	}

	return nil
}

func (p *TrustedPipeline) verifyPipeline(
	ctx context.Context,
	k8sClient kubernetes.Interface,
	tektonClient versioned.Interface) (errs *apis.FieldError) {
	logger := logging.FromContext(ctx)
	logger.Info("Verifying Pipeline")

	pipeline := p.Pipeline

	cfg := config.FromContextOrDefaults(ctx)
	cfg.FeatureFlags.EnableTektonOCIBundles = true
	ctx = config.ToContext(ctx, cfg)

	verifier, err := verifier(ctx)
	if err != nil {
		return apis.ErrGeneric(err.Error())
	}

	signature, err := getSignature(pipeline.PipelineMetadata())
	if err != nil {
		return apis.ErrGeneric(err.Error())
	}

	// Create an unsignedPipeline for the verification as the signature annotation will
	// not be needed for the signing computation
	unsignedPipeline := pipeline.DeepCopy()
	delete(unsignedPipeline.Annotations, SignatureAnnotation)

	if err := VerifyInterface(ctx, unsignedPipeline, verifier, signature); err != nil {
		return err
	}

	return nil
}
