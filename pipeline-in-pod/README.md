# Custom Task: Pipeline in a Pod

This is an experimental solution to [TEP-0044](https://github.com/tektoncd/community/blob/main/teps/0044-data-locality-and-pod-overhead-in-pipelines.md).

## Installation

### Install from nightly release

```
kubectl apply --filename https://storage.googleapis.com/tekton-releases-nightly/pipeline-in-pod/latest/release.yaml
```

### Build and install from source

```
ko apply -f config
```

## Supported Features
This custom task currently supports only running sequential tasks together in a pod with params, a pipeline-level timeout and workspaces.
In this implementation, workspace volumes are accessible to all tasks, but this can be changed.
The next features on the roadmap are OCI bundles and parallel tasks.
