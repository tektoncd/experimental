module github.com/tektoncd/experimental/tekdoc

go 1.13

require (
	github.com/evanphx/json-patch v4.5.0+incompatible // indirect
	github.com/gogo/protobuf v1.3.1 // indirect
	github.com/google/go-cmp v0.3.1 // indirect
	github.com/google/go-containerregistry v0.0.0-20191216221554-74b082017bc4 // indirect
	github.com/hashicorp/golang-lru v0.5.3 // indirect
	github.com/json-iterator/go v1.1.8 // indirect
	github.com/mattbaird/jsonpatch v0.0.0-20171005235357-81af80346b1a // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/tektoncd/pipeline v0.9.2
	go.uber.org/zap v1.13.0 // indirect
	golang.org/x/xerrors v0.0.0-20191204190536-9bdfabe68543 // indirect
	k8s.io/api v0.17.0 // indirect
	k8s.io/client-go v11.0.0+incompatible // indirect
	k8s.io/kube-openapi v0.0.0-20191107075043-30be4d16710a // indirect
	k8s.io/utils v0.0.0-20191217112158-dcd0c905194b // indirect
	knative.dev/pkg v0.0.0-20191216211902-b26ddf762bc9 // indirect
)

// Pin k8s deps to 1.12.9

replace (
	k8s.io/api => k8s.io/api v0.0.0-20191004102255-dacd7df5a50b
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20191004105443-a7d558db75c6
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20191004074956-01f8b7d1121a
	k8s.io/apiserver => k8s.io/apiserver v0.0.0-20191004103531-b568748c9b85
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.0.0-20191004110054-fe9b9282443f
	k8s.io/client-go => k8s.io/client-go v0.0.0-20191004102537-eb5b9a8cfde7
	k8s.io/code-generator => k8s.io/code-generator v0.0.0-20181117043124-c2090bec4d9b
	k8s.io/gengo => k8s.io/gengo v0.0.0-20190327210449-e17681d19d3a
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.0.0-20191004103911-2797d0dcf14b
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.0.0-20191016015407-72acd948ffff
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.0.0-20191016015246-999188f3eff6
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.0.0-20191016015341-7be46aeada42
	k8s.io/kubelet => k8s.io/kubelet v0.0.0-20191016015314-e7fc4f69fc2c
	k8s.io/kubernetes => k8s.io/kubernetes v1.13.12
	k8s.io/metrics => k8s.io/metrics v0.0.0-20191004105814-56635b1b5a0c
)
