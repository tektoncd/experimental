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

package main

import (
	"context"
	"github.com/tektoncd/experimental/celeval/pkg/apis/celeval"
	"knative.dev/pkg/injection/sharedmain"
	"os"

	celevalv1alpha1 "github.com/tektoncd/experimental/celeval/pkg/apis/celeval/v1alpha1"
	defaultconfig "github.com/tektoncd/pipeline/pkg/apis/config"
	"github.com/tektoncd/pipeline/pkg/contexts"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/injection"
	"knative.dev/pkg/system"

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
	WebhookLogKey = "tekton-celeval-webhook"
)

var types = map[schema.GroupVersionKind]resourcesemantics.GenericCRD{
	celevalv1alpha1.SchemeGroupVersion.WithKind(celeval.ControllerName): &celevalv1alpha1.CELEval{},
}

func newDefaultingAdmissionController(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
	// Decorate contexts with the current state of the config.
	store := defaultconfig.NewStore(logging.FromContext(ctx).Named("config-store"))
	store.WatchConfigs(cmw)

	return defaulting.NewAdmissionController(ctx,

		// Name of the resource webhook.
		"webhook.celeval.custom.tekton.dev",

		// The path on which to serve the webhook.
		"/defaulting",

		// The resources to validate and default.
		types,

		// A function that infuses the context passed to Validate/SetDefaults with custom metadata.
		func(ctx context.Context) context.Context {
			return contexts.WithUpgradeViaDefaulting(store.ToContext(ctx))
		},

		// Whether to disallow unknown fields.
		true,
	)
}

func newValidationAdmissionController(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
	// Decorate contexts with the current state of the config.
	store := defaultconfig.NewStore(logging.FromContext(ctx).Named("config-store"))
	store.WatchConfigs(cmw)
	return validation.NewAdmissionController(ctx,

		// Name of the resource webhook.
		"validation.webhook.celeval.custom.tekton.dev",

		// The path on which to serve the webhook.
		"/resource-validation",

		// The resources to validate and default.
		types,

		// A function that infuses the context passed to Validate/SetDefaults with custom metadata.
		func(ctx context.Context) context.Context {
			return contexts.WithUpgradeViaDefaulting(store.ToContext(ctx))
		},

		// Whether to disallow unknown fields.
		true,
	)
}

func main() {
	serviceName := os.Getenv("WEBHOOK_SERVICE_NAME")
	if serviceName == "" {
		serviceName = "tekton-celeval-webhook"
	}

	secretName := os.Getenv("WEBHOOK_SECRET_NAME")
	if secretName == "" {
		secretName = "tekton-celeval-webhook-certs" // #nosec
	}

	// Scope informers to the webhook's namespace instead of cluster-wide
	ctx := injection.WithNamespaceScope(signals.NewContext(), system.Namespace())

	// Set up a signal context with our webhook options
	ctx = webhook.WithOptions(ctx, webhook.Options{
		ServiceName: serviceName,
		Port:        8443,
		SecretName:  secretName,
	})

	sharedmain.WebhookMainWithConfig(ctx, WebhookLogKey,
		sharedmain.ParseAndGetConfigOrDie(),
		certificates.NewController,
		newDefaultingAdmissionController,
		newValidationAdmissionController,
	)
}
