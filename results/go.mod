module github.com/tektoncd/experimental/results

go 1.13

require (
	cloud.google.com/go v0.66.0
	github.com/blang/semver v3.5.1+incompatible // indirect
	github.com/go-sql-driver/mysql v1.5.0
	github.com/golang/protobuf v1.4.2
	github.com/google/cel-go v0.5.1
	github.com/google/go-cmp v0.5.2
	github.com/google/uuid v1.1.1
	github.com/mattn/go-sqlite3 v2.0.3+incompatible
	github.com/smartystreets/assertions v1.1.1 // indirect
	github.com/smartystreets/goconvey v1.6.4 // indirect
	github.com/tektoncd/pipeline v0.12.0
	go.chromium.org/luci v0.0.0-20200716065131-1f7c6da65cb2
	go.uber.org/zap v1.15.0
	gomodules.xyz/jsonpatch/v2 v2.1.0
	google.golang.org/api v0.31.0
	google.golang.org/genproto v0.0.0-20200914193844-75d14daec038
	google.golang.org/grpc v1.31.1
	google.golang.org/protobuf v1.25.0
	k8s.io/api v0.18.6
	k8s.io/apiextensions-apiserver v0.18.6 // indirect
	k8s.io/apimachinery v0.18.6
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	knative.dev/pkg v0.0.0-20200731005101-694087017879

)

// Knative deps (release-0.13)
replace (
	contrib.go.opencensus.io/exporter/stackdriver => contrib.go.opencensus.io/exporter/stackdriver v0.12.9-0.20191108183826-59d068f8d8ff
	knative.dev/caching => knative.dev/caching v0.0.0-20200116200605-67bca2c83dfa
	knative.dev/pkg => knative.dev/pkg v0.0.0-20200306230727-a56a6ea3fa56
	knative.dev/pkg/vendor/github.com/spf13/pflag => github.com/spf13/pflag v1.0.5
)

// Pin k8s deps to 1.16.5
replace (
	k8s.io/api => k8s.io/api v0.16.5
	k8s.io/apimachinery => k8s.io/apimachinery v0.16.5
	k8s.io/client-go => k8s.io/client-go v0.16.5
	k8s.io/code-generator => k8s.io/code-generator v0.16.5
	k8s.io/gengo => k8s.io/gengo v0.0.0-20190327210449-e17681d19d3a
)
