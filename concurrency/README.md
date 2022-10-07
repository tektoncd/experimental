This project aims to address [TEP-0120: Canceling Concurrent PipelineRuns](https://github.com/tektoncd/community/blob/main/teps/0120-canceling-concurrent-pipelineruns.md).
It is an experimental project that may change in breaking ways at any time.

## Installation

### Install from nightly release

```
kubectl apply --filename https://storage.googleapis.com/tekton-releases-nightly/concurrency/latest/release.yaml
```

### Build and install from source

```
ko apply -f config
```

## Usage

To try it out:
```sh
kubectl apply -f examples/concurrencycontrol.yaml
kubectl create -f examples/pipelinerun.yaml
```

This first PipelineRun should begin executing normally.
Now, create another PipelineRun that also matches the concurrency control:
```sh
kubectl create -f examples/pipelinerun.yaml
```

The first PipelineRun should be canceled, and the second one should execute normally.

### Supported concurrency strategies

Supported strategies are "Cancel", "GracefullyCancel", and "GracefullyStop"
(corresponding to canceling, gracefully canceling, and gracefully stopping a PipelineRun, respectively).
The default strategy is "GracefullyCancel".
If multiple ConcurrencyControls with different strategies apply to the same PipelineRun, concurrency controls will fail.