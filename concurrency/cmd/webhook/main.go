package main

import (
	"context"
	"os"

	"github.com/tektoncd/experimental/concurrency/pkg/apis/concurrency/v1alpha1"
	defaultconfig "github.com/tektoncd/experimental/concurrency/pkg/apis/config"
	"github.com/tektoncd/experimental/concurrency/pkg/mutatingwebhook"
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
	concurrencyControlKind = v1alpha1.SchemeGroupVersion.WithKind("ConcurrencyControl")
)

func newValidationAdmissionController(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
	var types = map[schema.GroupVersionKind]resourcesemantics.GenericCRD{
		concurrencyControlKind: &v1alpha1.ConcurrencyControl{},
	}
	store := defaultconfig.NewStore(logging.FromContext(ctx).Named("config-store"))
	store.WatchConfigs(cmw)
	return validation.NewAdmissionController(ctx,

		// Name of the resource webhook.
		"validation.webhook.concurrency.custom.tekton.dev",

		// The path on which to serve the webhook.
		"/resource-validation",

		// The resources to validate and default.
		types,

		// A function that infuses the context passed to Validate/SetDefaults with custom metadata.
		func(ctx context.Context) context.Context {
			return store.ToContext(ctx)
		},

		// Whether to disallow unknown fields.
		true,
	)
}

func newDefaultingAdmissionController(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
	var types = map[schema.GroupVersionKind]resourcesemantics.GenericCRD{
		concurrencyControlKind: &v1alpha1.ConcurrencyControl{},
	}
	store := defaultconfig.NewStore(logging.FromContext(ctx).Named("config-store"))
	store.WatchConfigs(cmw)
	return defaulting.NewAdmissionController(ctx,

		// Name of the resource webhook.
		"defaulting.webhook.concurrency.custom.tekton.dev",

		// The path on which to serve the webhook.
		"/defaulting",

		// The resources to validate and default.
		types,

		// A function that infuses the context passed to Validate/SetDefaults with custom metadata.
		func(ctx context.Context) context.Context {
			return store.ToContext(ctx)
		},

		// Whether to disallow unknown fields.
		false,
	)
}

func newMutatingAdmissionController(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
	store := defaultconfig.NewStore(logging.FromContext(ctx).Named("config-store"))
	store.WatchConfigs(cmw)
	return mutatingwebhook.NewAdmissionController(ctx,

		// Name of the resource webhook.
		"mutation.webhook.concurrency.custom.tekton.dev",

		// The path on which to serve the webhook.
		"/mutating",

		// A function that infuses the context passed to Validate/SetDefaults with custom metadata.
		func(ctx context.Context) context.Context {
			return store.ToContext(ctx)
		},
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
		newMutatingAdmissionController,
	)
}
