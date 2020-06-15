module generators

go 1.14

require (
	github.com/google/go-cmp v0.4.1
	github.com/tektoncd/pipeline v0.13.2
	k8s.io/api v0.17.6
	k8s.io/apimachinery v0.17.6
)

replace (
	k8s.io/api => k8s.io/api v0.0.0-20200214081623-ecbd4af0fc33
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20200214081019-7490b3ed6e92
	k8s.io/client-go => k8s.io/client-go v0.0.0-20200214082307-e38a84523341
	k8s.io/code-generator => k8s.io/code-generator v0.0.0-20200214080538-dc8f3adce97c
)
