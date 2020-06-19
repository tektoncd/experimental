module github.com/tektoncd/experimental/commit-status-tracker

go 1.14

require (
	github.com/jenkins-x/go-scm v1.5.142
	github.com/operator-framework/operator-sdk v0.17.1
	github.com/spf13/pflag v1.0.5
	github.com/tektoncd/pipeline v0.13.2
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	k8s.io/api v0.18.2
	k8s.io/apimachinery v0.18.2
	k8s.io/client-go v12.0.0+incompatible
	knative.dev/pkg v0.0.0-20200528142800-1c6815d7e4c9
	sigs.k8s.io/controller-runtime v0.5.2
)

// Knative deps (release-0.15)
replace (
	contrib.go.opencensus.io/exporter/stackdriver => contrib.go.opencensus.io/exporter/stackdriver v0.12.9-0.20191108183826-59d068f8d8ff
	github.com/Azure/azure-sdk-for-go => github.com/Azure/azure-sdk-for-go v38.2.0+incompatible
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.4.0+incompatible
	knative.dev/caching => knative.dev/caching v0.0.0-20200521155757-e78d17bc250e
	knative.dev/pkg => knative.dev/pkg v0.0.0-20200528142800-1c6815d7e4c9
)

// Pin k8s deps to 1.16.5
replace (
	k8s.io/api => k8s.io/api v0.16.5
	k8s.io/apimachinery => k8s.io/apimachinery v0.16.5
	k8s.io/client-go => k8s.io/client-go v0.16.5
	k8s.io/code-generator => k8s.io/code-generator v0.16.5
)
