module github.com/tektoncd/experimental/workflows

go 1.16

require (
	github.com/GoogleCloudPlatform/cloud-builders/gcs-fetcher v0.0.0-20210729182058-ea1f5c7c37f1
	github.com/tektoncd/plumbing v0.0.0-20210514044347-f8a9689d5bd5
	go.uber.org/zap v1.18.1
	k8s.io/api v0.20.7
	k8s.io/apimachinery v0.20.7
	k8s.io/client-go v0.20.7
	k8s.io/code-generator v0.20.7
	k8s.io/kube-openapi v0.0.0-20201113171705-d219536bb9fd
	knative.dev/pkg v0.0.0-20210730172132-bb4aaf09c430
)

// Knative deps (release-0.20)
replace (
	contrib.go.opencensus.io/exporter/stackdriver => contrib.go.opencensus.io/exporter/stackdriver v0.13.4
	github.com/Azure/azure-sdk-for-go => github.com/Azure/azure-sdk-for-go v38.2.0+incompatible
)
