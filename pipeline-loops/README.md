# Custom Task: Pipeline Loops

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/kubernetes/experimental/blob/master/LICENSE)

The Pipeline Loop Extension for Tekton allows users to run a `Pipeline` in a loop with varying parameter values.
This functionality is provided by a controller that implements the [Tekton Custom Task interface](https://github.com/tektoncd/pipeline/blob/master/docs/runs.md).

This Extension is using by [kubeflow/kfp-tekton](https://github.com/kubeflow/kfp-tekton) and the users who are using kfp-tekton right now.

# Goal
`Pipeline Loops` is trying to provide pipeline level loops to handle `withItems` loop in `Pipeline` (Tekton).

# Installation

## Option One: Install using KO

- Install and configure [KO](https://github.com/google/ko)
- Install pipelineloop controller
  `ko apply -f config/`
  
## Option Two: Install by building your own Docker images

- Modify `Makefile` by changing `registry-name` to your Docker hub name

- Run `make images` to build the docker image of yours.

- Modify `config/500-webhook.yaml` and `config/500-controller.yaml` Change the image name to your docker image, e.g.:
```
- name: webhook
  image: fenglixa/pipelineloop-webhook:v0.0.1
```
```
- name: tekton-pipelineloop-controller
  image: fenglixa/pipelineloop-controller:v0.0.1
```

- Install pipelineloop controller `kubectl apply -f config/`


# Verification
- check controller and the webhook. `kubectl get po -n tekton-pipelines`
   ```
    ...
    tekton-pipelineloop-controller-db4c7dddb-vrlsd                        1/1     Running     0          6h24m
    tekton-pipelineloop-webhook-7bb98ddc98-qqkv6                          1/1     Running     0          6h17m
   ```
- Try the cases of loop pipelines:
  1. Run pipeline loop with loop parameter as array value:
  - `kubectl apply -f examples/pipelinespec-with-run-arrary-value.yaml`
  2. Run pipeline loop with loop parameter as string value:
  - `kubectl apply -f examples/pipelinespec-with-run-string-value.yaml`
  3. Run pipeline loop with loop parameter as 'From', 'To', 'Step' defined, for example:
  - `kubectl apply -f examples/pipelinespec-with-run-iterate-numeric.yaml`
  4. Run pipeline loop with condition together. The loop will be continue util the condition in the loop is satisfied. The example could be refer to below case:
  - `kubectl apply -f examples/pipelinespec-with-run-condition.yaml`
  5. Run pipeline loop with loop parameter as dict value, then multiple loop parameters could be supported:
  - `kubectl apply -f examples/pipelinespec-with-run-dict-value.yaml`

# End to end example
- Install Tekton version >= v0.19
- Edit feature-flags configmap, ensure "data.enable-custom-tasks" is "true":
`kubectl edit cm feature-flags -n tekton-pipelines`

- Run the E2E example: `kubectl apply -f examples/loop-example-basic.yaml`
