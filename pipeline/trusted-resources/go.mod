module github.com/tektoncd/experimental/pipelines/trusted-resources

go 1.16

require (
	github.com/sigstore/cosign v1.5.2
	github.com/sigstore/sigstore v1.1.1-0.20220130134424-bae9b66b8442
	github.com/tektoncd/pipeline v0.32.1-0.20220207152807-6cb0f4ccfce0
	k8s.io/api v0.22.5
	k8s.io/apimachinery v0.22.5
	k8s.io/client-go v0.22.5
	knative.dev/pkg v0.0.0-20220131144930-f4b57aef0006
)

require (
	cloud.google.com/go/iam v0.2.0 // indirect
	github.com/google/go-cmp v0.5.7
	github.com/google/go-containerregistry v0.8.1-0.20220202214207-9c35968ef47e
	go.uber.org/zap v1.20.0
	sigs.k8s.io/yaml v1.3.0
)
