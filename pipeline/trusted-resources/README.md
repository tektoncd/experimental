# Trusted Task

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/tektoncd/experimental/blob/master/LICENSE)

This is an experimental project to provide a seperate admission webhook for remote resources verification.

generate deepcopy code
```bash
go mod download k8s.io/code-generator
go install k8s.io/code-generator/cmd/deepcopy-gen
$HOME/go/bin/deepcopy-gen   -O zz_generated.deepcopy   --go-header-file ./hack/boilerplate/boilerplate.go.txt  -i ./pkg/trustedtask

# or use go generate
# cd pkg/trustedtask
# go generate
```
