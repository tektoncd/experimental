module github.com/tektoncd/experimental/interceptors

go 1.13

require (
	github.com/google/go-github/v31 v31.0.0
	github.com/lestrrat-go/jwx v1.1.7
	github.com/open-policy-agent/opa v0.27.1
	github.com/tektoncd/triggers v0.12.1
	go.uber.org/zap v1.16.0
	google.golang.org/grpc v1.34.0
	k8s.io/apimachinery v0.19.7
	k8s.io/client-go v0.19.7
	knative.dev/pkg v0.0.0-20210130001831-ca02ef752ac6
)

replace github.com/tektoncd/triggers => /Users/jamesmcshane/go/src/github.com/tektoncd/triggers
