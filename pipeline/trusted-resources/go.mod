module github.com/tektoncd/experimental/pipelines/trusted-resources

go 1.16

require (
	github.com/sigstore/cosign v1.10.1
	github.com/sigstore/sigstore v1.2.1-0.20220614141825-9c0e2e247545
	github.com/tektoncd/pipeline v0.32.1-0.20220207152807-6cb0f4ccfce0
	k8s.io/api v0.23.5
	k8s.io/apimachinery v0.23.5
	k8s.io/client-go v0.23.5
	knative.dev/pkg v0.0.0-20220131144930-f4b57aef0006
)

require (
	github.com/google/go-cmp v0.5.8
	github.com/google/go-containerregistry v0.11.0
	github.com/google/go-containerregistry/pkg/authn/k8schain v0.0.0-20220125170349-50dfc2733d10 // indirect
	github.com/google/go-containerregistry/pkg/authn/kubernetes v0.0.0-20220125170349-50dfc2733d10 // indirect
	go.uber.org/zap v1.21.0
	sigs.k8s.io/yaml v1.3.0
)
