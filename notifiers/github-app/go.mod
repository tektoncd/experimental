module github.com/tektoncd/experimental/notifiers/github-app

go 1.16

require (
	github.com/Azure/go-autorest/autorest/validation v0.2.0 // indirect
	github.com/bradleyfalzon/ghinstallation v1.1.1
	github.com/google/go-cmp v0.5.6
	github.com/google/go-github/v29 v29.0.3 // indirect
	github.com/google/go-github/v32 v32.1.0
	github.com/tektoncd/pipeline v0.25.0
	go.uber.org/zap v1.17.0
	golang.org/x/oauth2 v0.0.0-20210514164344-f6687ab2804c
	k8s.io/api v0.21.1
	k8s.io/apimachinery v0.21.1
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	knative.dev/pkg v0.0.0-20210616195222-841aa7369ca1
	sigs.k8s.io/yaml v1.2.0
)

replace (
	k8s.io/api => k8s.io/api v0.20.7
	k8s.io/apimachinery => k8s.io/apimachinery v0.20.7
	k8s.io/client-go => k8s.io/client-go v0.20.7
)
