# Trusted Task

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/tektoncd/experimental/blob/master/LICENSE)

This is an experimental project to provide a seperate webhook for remote resources verification.

- [Install](#install)
- [Uninstall](#uninstall)
- [Development](#development)
- [Config Secret Path](#config-secret-path)

## Install

Install and configure [`ko`](https://github.com/google/ko).

Install tekton pipeline. To install from source, checkout to pipeline repo and execute:
```bash
ko apply -f config/
```

Generate cosign key pair
```bash
# cosign generate-key-pair k8s://tekton-pipelines/signing-secrets
cosign generate-key-pair
```

Prepare signed files
```bash
# This is a demo of how to generate signed files.
go run cmd/sign/main.go -pk=cosign.key -ts=examples/example-task.yaml -td=examples
```

Then install the new admission webhook:
```bash
# delete secret if already exists
# kubectl delete secret signing-secrets -n tekton-trusted-resources
kubectl create secret generic signing-secrets \
  --from-file=cosign.key=./cosign.key \
  --from-literal=cosign.password='1234'\
  --from-file=cosign.pub=./cosign.pub \
  -n tekton-trusted-resources

ko apply -f config/
```

Examples:
```bash
# Test API taskrun
ko apply -f examples/1-test-taskrun-signed.yaml

# Test OCI Bundle
# add this secret to controller's service account
kubectl create secret generic ${SECRET_NAME} \
--from-file=.dockerconfigjson=<path/to/.docker/config.json> \
--type=kubernetes.io/dockerconfigjson
--namespace=tekton-pipelines

tkn bundle push docker.io/my-dockerhub-username/testtask:latest -f examples/signed.yaml
cosign sign --key cosign.key docker.io/my-dockerhub-username/testtask:latest
```

## Uninstall

```bash
ko delete -f config/
```


## Development

generate deepcopy code
```bash
go mod download k8s.io/code-generator
go install k8s.io/code-generator/cmd/deepcopy-gen
$HOME/go/bin/deepcopy-gen   -O zz_generated.deepcopy   --go-header-file ./hack/boilerplate/boilerplate.go.txt  -i ./pkg/trustedtask

# or use go generate
# cd pkg/trustedtask
# go generate
```

## Config Secret Path

`signing-secret-path` is used to specify the mounted path to store the cosign pubkey. By default it is "/etc/signing-secrets/cosign.pub".

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: config-trusted-resources
  namespace: tekton-trusted-resources
  labels:
    app.kubernetes.io/component: tekton-trusted-resources
    app.kubernetes.io/instance: default
    app.kubernetes.io/part-of: admissioncontrol
data:
  signing-secret-path: "/etc/signing-secrets/cosign.pub"
```
