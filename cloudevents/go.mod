module github.com/tektoncd/experimental/cloudevents

go 1.16

replace (
	k8s.io/api => k8s.io/api v0.22.5
	k8s.io/apimachinery => k8s.io/apimachinery v0.22.5
	k8s.io/client-go => k8s.io/client-go v0.22.5
)

require (
	github.com/cdfoundation/sig-events/cde/sdk/go v0.0.0-20210619194635-1b767876db95
	github.com/cloudevents/sdk-go/v2 v2.5.0
	github.com/google/go-cmp v0.5.7
	github.com/hashicorp/golang-lru v0.5.4
	github.com/tektoncd/pipeline v0.33.2
	k8s.io/api v0.23.4
	k8s.io/apimachinery v0.23.4
	k8s.io/client-go v1.5.2
	knative.dev/pkg v0.0.0-20220131144930-f4b57aef0006
)
