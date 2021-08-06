module github.com/tektoncd/experimental/wait-task

go 1.15

require (
	github.com/tektoncd/pipeline v0.20.1
	k8s.io/api v0.20.7
	k8s.io/apimachinery v0.20.7
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	knative.dev/pkg v0.0.0-20210805073852-23b014724657
)

replace k8s.io/client-go => k8s.io/client-go v0.20.2
