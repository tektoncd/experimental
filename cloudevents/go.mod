module github.com/tektoncd/experimental/cloudevents

go 1.15

require (
	github.com/cloudevents/sdk-go/v2 v2.1.0 // indirect
	github.com/google/go-cmp v0.5.4 // indirect
	github.com/google/go-containerregistry v0.5.1 // indirect
	github.com/satori/go.uuid v1.2.0 // indirect
	github.com/tektoncd/pipeline v0.20.1
	k8s.io/api v0.20.2
	k8s.io/apimachinery v0.20.2 // indirect
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	knative.dev/pkg v0.0.0-20210119162123-1bbf0a6436c3
)

replace k8s.io/client-go => k8s.io/client-go v0.20.2
