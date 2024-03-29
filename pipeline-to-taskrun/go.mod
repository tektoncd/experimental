module github.com/tektoncd/experimental/pipeline-to-taskrun

go 1.15

require (
	github.com/google/go-cmp v0.5.5
	github.com/hashicorp/go-multierror v1.1.0
	github.com/tektoncd/pipeline v0.24.0
	go.uber.org/zap v1.16.0
	k8s.io/api v0.20.7
	k8s.io/apimachinery v0.20.7
	k8s.io/client-go v0.20.7
	knative.dev/pkg v0.0.0-20210510175900-4564797bf3b7
)

// Knative deps
replace (
	contrib.go.opencensus.io/exporter/stackdriver => contrib.go.opencensus.io/exporter/stackdriver v0.12.9-0.20191108183826-59d068f8d8ff
	github.com/Azure/azure-sdk-for-go => github.com/Azure/azure-sdk-for-go v38.2.0+incompatible
)
