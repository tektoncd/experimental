module github.com/tektoncd/experimental/task-loops

go 1.13

require (
	github.com/ghodss/yaml v1.0.0
	github.com/google/go-cmp v0.5.4
	github.com/hashicorp/go-multierror v1.1.0
	github.com/tektoncd/pipeline v0.23.0
	go.opencensus.io v0.22.5
	go.uber.org/zap v1.16.0
	gomodules.xyz/jsonpatch/v2 v2.1.0
	k8s.io/api v0.19.7
	k8s.io/apimachinery v0.19.7
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	knative.dev/pkg v0.0.0-20210127163530-0d31134d5f4e
)

// Knative deps (release-0.20)
replace (
	contrib.go.opencensus.io/exporter/stackdriver => contrib.go.opencensus.io/exporter/stackdriver v0.13.4
	github.com/Azure/azure-sdk-for-go => github.com/Azure/azure-sdk-for-go v38.2.0+incompatible
	github.com/tektoncd/pipeline => github.com/scrapcodes/pipeline v0.23.1-0.20210416113956-a5d48318ae7d
)

// Pin k8s deps to v0.18.8
replace (
	k8s.io/api => k8s.io/api v0.18.8
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.18.8
	k8s.io/apimachinery => k8s.io/apimachinery v0.18.8
	k8s.io/client-go => k8s.io/client-go v0.18.8
	k8s.io/code-generator => k8s.io/code-generator v0.18.8
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20200410145947-bcb3869e6f29
)