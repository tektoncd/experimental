package main

import (
	"context"
	"os"

	"github.com/tektoncd/experimental/concurrency/pkg/apis/concurrency/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/injection"
	"knative.dev/pkg/injection/sharedmain"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/signals"
	"knative.dev/pkg/webhook"
	"knative.dev/pkg/webhook/certificates"
	"knative.dev/pkg/webhook/resourcesemantics"
	"knative.dev/pkg/webhook/resourcesemantics/defaulting"
	"knative.dev/pkg/webhook/resourcesemantics/validation"
)

const (
	// WebhookLogKey is the name of the logger for the webhook cmd.
	// This name is also used to form lease names for the leader election of the webhook's controllers.
	WebhookLogKey = "concurrency-webhook"
)

var (
	pipelineRunKind = v1beta1.SchemeGroupVersion.WithKind("PipelineRun")

	c = defaulting.NewCallback(func(ctx context.Context, uns *unstructured.Unstructured) error {
		pr := v1beta1.PipelineRun{}
		logger := logging.FromContext(ctx)
		logger.Infof("calling defaulting callback")
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(uns.UnstructuredContent(), &pr); err != nil {
			return err
		}
		logger.Infof("pr name: %s", pr.Name)
		if pr.Spec.Status == v1beta1.PipelineRunSpecStatusPending {
			return nil
		}

		pr.Spec.Status = v1beta1.PipelineRunSpecStatusPending
		// Add a label to indicate that the webhook was responsible for patching this PipelineRun as pending,
		// and that the reconciler should start it. This is to distinguish from PipelineRuns that were created
		// as Pending, and shouldn't be started by the reconciler.
		pr.ObjectMeta.Labels[v1alpha1.LabelToStartPR] = "true"
		u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&pr)
		if err != nil {
			return err
		}
		uns.Object = u
		return nil
	}, webhook.Create)
)

func newValidationAdmissionController(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
	var types = map[schema.GroupVersionKind]resourcesemantics.GenericCRD{
		v1alpha1.SchemeGroupVersion.WithKind("ConcurrencyControl"): &v1alpha1.ConcurrencyControl{},
	}
	return validation.NewAdmissionController(ctx,

		// Name of the resource webhook.
		"validation.webhook.concurrency.custom.tekton.dev",

		// The path on which to serve the webhook.
		"/resource-validation",

		// The resources to validate and default.
		types,

		// A function that infuses the context passed to Validate/SetDefaults with custom metadata.
		func(ctx context.Context) context.Context {
			return ctx
		},

		// Whether to disallow unknown fields.
		true,
	)
}

func newDefaultingAdmissionController(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
	var types = map[schema.GroupVersionKind]resourcesemantics.GenericCRD{
		pipelineRunKind: &v1beta1.PipelineRun{},
	}
	return defaulting.NewAdmissionController(ctx,

		// Name of the resource webhook.
		"webhook.concurrency.custom.tekton.dev",

		// The path on which to serve the webhook.
		"/defaulting",

		// The resources to validate and default.
		types,

		// A function that infuses the context passed to Validate/SetDefaults with custom metadata.
		func(ctx context.Context) context.Context {
			return ctx
		},

		// Whether to disallow unknown fields.
		false,

		map[schema.GroupVersionKind]defaulting.Callback{pipelineRunKind: c},
	)
}

func main() {
	serviceName := os.Getenv("WEBHOOK_SERVICE_NAME")
	if serviceName == "" {
		serviceName = "tekton-concurrency-webhook"
	}

	secretName := os.Getenv("WEBHOOK_SECRET_NAME")
	if secretName == "" {
		secretName = "tekton-concurrency-webhook-certs" // #nosec
	}

	// Scope informers to the webhook's namespace instead of cluster-wide
	ctx := injection.WithNamespaceScope(signals.NewContext(), "tekton-concurrency")

	// Set up a signal context with our webhook options
	ctx = webhook.WithOptions(ctx, webhook.Options{
		ServiceName: serviceName,
		Port:        8443,
		SecretName:  secretName,
	})

	sharedmain.MainWithContext(ctx, WebhookLogKey,
		certificates.NewController,
		newValidationAdmissionController,
		newDefaultingAdmissionController,
	)
}
