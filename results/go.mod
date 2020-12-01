module github.com/tektoncd/experimental/results

go 1.13

require (
	cloud.google.com/go v0.66.0 // indirect
	github.com/evanphx/json-patch v4.9.0+incompatible // indirect
	github.com/golang/protobuf v1.4.3
	github.com/google/cel-go v0.5.1
	github.com/google/go-cmp v0.5.2
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/uuid v1.1.2
	github.com/mattn/go-sqlite3 v2.0.3+incompatible // indirect
	github.com/sirupsen/logrus v1.7.0 // indirect
	github.com/smartystreets/assertions v1.1.1 // indirect
	github.com/tektoncd/pipeline v0.17.1
	go.chromium.org/luci v0.0.0-20200716065131-1f7c6da65cb2
	go.uber.org/zap v1.15.0
	golang.org/x/crypto v0.0.0-20201016220609-9e8e0b390897 // indirect
	golang.org/x/sys v0.0.0-20201101102859-da207088b7d1 // indirect
	golang.org/x/tools v0.0.0-20201102212025-f46e4245211d // indirect
	gomodules.xyz/jsonpatch/v2 v2.1.0
	google.golang.org/api v0.31.0
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20201019141844-1ed22bb0c154
	google.golang.org/grpc v1.33.1
	google.golang.org/protobuf v1.25.0
	gorm.io/driver/mysql v1.0.3
	gorm.io/driver/sqlite v1.1.4
	gorm.io/gorm v1.20.7
	k8s.io/api v0.19.3
	k8s.io/apiextensions-apiserver v0.18.6 // indirect
	k8s.io/apimachinery v0.19.3
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	k8s.io/klog/v2 v2.1.0 // indirect
	k8s.io/kube-openapi v0.0.0-20200923155610-8b5066479488 // indirect
	k8s.io/utils v0.0.0-20201015054608-420da100c033 // indirect
	knative.dev/pkg v0.0.0-20200831162708-14fb2347fb77

)

// Knative deps (release-0.16)
replace (
	contrib.go.opencensus.io/exporter/stackdriver => contrib.go.opencensus.io/exporter/stackdriver v0.12.9-0.20191108183826-59d068f8d8ff
	github.com/Azure/azure-sdk-for-go => github.com/Azure/azure-sdk-for-go v38.2.0+incompatible
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.4.0+incompatible
	knative.dev/pkg => knative.dev/pkg v0.0.0-20200831162708-14fb2347fb77
)

// Pin k8s deps to 1.17.6
replace (
	k8s.io/api => k8s.io/api v0.17.6
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.17.6
	k8s.io/apimachinery => k8s.io/apimachinery v0.17.6
	k8s.io/apiserver => k8s.io/apiserver v0.17.6
	k8s.io/client-go => k8s.io/client-go v0.17.6
	k8s.io/code-generator => k8s.io/code-generator v0.17.6
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20200410145947-bcb3869e6f29
)
