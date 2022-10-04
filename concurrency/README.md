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