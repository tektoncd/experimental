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
This custom task currently supports only running tasks together in a pod with params, a pipeline-level timeout and workspaces.
The next feature on the roadmap is OCI bundles or remote tasks.

### Results support
This custom task supports outputting task results, but does not support passing results of one task into the
parameters of another task.

### Workspaces support
This custom task supports workspaces backed by emptyDir; they may be optional or required.
Workspace volumes are mounted only onto the steps that need them.
