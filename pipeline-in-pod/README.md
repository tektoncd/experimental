# Custom Task: Pipeline in a Pod

This is an experimental solution to [TEP-0044](https://github.com/tektoncd/community/blob/main/teps/0044-data-locality-and-pod-overhead-in-pipelines.md).

## Installation
This task can be built and installed with `ko`.

## Supported Features
This custom task currently supports only running sequential tasks together in a pod.
The next features on the roadmap are params, workspaces, and parallel tasks.
