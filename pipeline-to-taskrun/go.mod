module github.com/tektoncd/experimental/pipeline-to-taskrun

go 1.15

require (
	github.com/docker/go-metrics v0.0.1 // indirect
	github.com/fsnotify/fsnotify v1.4.9 // indirect
	github.com/fvbommel/sortorder v1.0.2 // indirect
	github.com/gobuffalo/envy v1.7.1 // indirect
	github.com/google/go-cmp v0.5.5
	github.com/hashicorp/go-multierror v1.1.0
	github.com/markbates/inflect v1.0.4 // indirect
	github.com/rogpeppe/go-internal v1.5.2 // indirect
	github.com/tektoncd/pipeline v0.24.0
	github.com/theupdateframework/notary v0.7.0 // indirect
	go.uber.org/zap v1.16.0
	k8s.io/api v0.20.2
	k8s.io/apimachinery v0.19.7
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	knative.dev/pkg v0.0.0-20210331065221-952fdd90dbb0
	sigs.k8s.io/structured-merge-diff/v3 v3.0.1-0.20200706213357-43c19bbb7fba // indirect
)

// Knative deps
replace (
	contrib.go.opencensus.io/exporter/stackdriver => contrib.go.opencensus.io/exporter/stackdriver v0.12.9-0.20191108183826-59d068f8d8ff
	github.com/Azure/azure-sdk-for-go => github.com/Azure/azure-sdk-for-go v38.2.0+incompatible
)

// Pin k8s deps to v0.18.8
replace (
	k8s.io/api => k8s.io/api v0.18.8
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.18.8
	k8s.io/apimachinery => k8s.io/apimachinery v0.18.8
	k8s.io/client-go => k8s.io/client-go v0.18.8
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20200410145947-bcb3869e6f29
)
