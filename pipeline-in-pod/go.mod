module github.com/tektoncd/experimental/pipeline-in-pod

go 1.16

replace (
	k8s.io/api => k8s.io/api v0.22.5
	k8s.io/apimachinery => k8s.io/apimachinery v0.22.5
	k8s.io/client-go => k8s.io/client-go v0.22.5
	k8s.io/code-generator => k8s.io/code-generator v0.22.5
)

require (
	github.com/containerd/containerd v1.5.16
	github.com/google/go-cmp v0.5.7
	github.com/google/go-containerregistry v0.8.1-0.20220216220642-00c59d91847c
	github.com/google/go-containerregistry/pkg/authn/k8schain v0.0.0-20220216220642-00c59d91847c
	github.com/google/uuid v1.3.0
	github.com/hashicorp/go-multierror v1.1.1
	github.com/hashicorp/golang-lru v0.5.4
	github.com/opencontainers/image-spec v1.0.3-0.20220114050600-8b9d41f48198
	github.com/tektoncd/pipeline v0.34.1
	github.com/tektoncd/plumbing v0.0.0-20211012143332-c7cc43d9bc0c
	go.uber.org/zap v1.21.0
	gomodules.xyz/jsonpatch/v2 v2.2.0
	k8s.io/api v0.23.4
	k8s.io/apimachinery v0.23.4
	k8s.io/client-go v1.5.2
	k8s.io/code-generator v0.22.5
	k8s.io/kube-openapi v0.0.0-20220124234850-424119656bbf
	knative.dev/pkg v0.0.0-20220131144930-f4b57aef0006
)
