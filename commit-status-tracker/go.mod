module github.com/tektoncd/experimental/commit-status-tracker

go 1.16

require (
	github.com/jenkins-x/go-scm v1.10.11
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.15.0
	github.com/spf13/pflag v1.0.5
	github.com/tektoncd/pipeline v0.31.0
	golang.org/x/oauth2 v0.0.0-20211005180243-6b3c2da341f1
	k8s.io/api v0.22.1
	k8s.io/apimachinery v0.22.1
	k8s.io/client-go v0.22.1
	k8s.io/kube-openapi v0.0.0-20211110013926-83f114cd0513 // indirect
	knative.dev/pkg v0.0.0-20211101212339-96c0204a70dc
	sigs.k8s.io/controller-runtime v0.10.0
)

replace (
	k8s.io/api v0.22.1 => k8s.io/api v0.21.4
	k8s.io/apimachinery v0.22.1 => k8s.io/apimachinery v0.21.4
	k8s.io/client-go v0.22.1 => k8s.io/client-go v0.21.4
)
