module github.com/tektoncd/experimental/pipeline-in-pod

go 1.13

require (
	github.com/hashicorp/go-multierror v1.1.1
	github.com/tektoncd/pipeline v0.31.0
	k8s.io/client-go v0.21.4
	knative.dev/pkg v0.0.0-20211216142117-79271798f696
)

require (
	github.com/google/go-containerregistry v0.6.0
	go.uber.org/zap v1.19.1 // indirect
	golang.org/x/term v0.0.0-20210615171337-6886f2dfbf5b // indirect
	golang.org/x/time v0.0.0-20210723032227-1f47c861a9ac // indirect
	gomodules.xyz/jsonpatch/v2 v2.2.0
	k8s.io/api v0.21.4
	k8s.io/apimachinery v0.21.4
)
