# Tekton Kubernetes Helm Charts

This functionality is in beta and is subject to change.

## Helm charts repository

**TODO** this is not yet available, maybe document how to install from sources

The Tekton helm charts repository is available here: `https://charts.tekton.dev`.

Add the Tekton helm charts repo:

```bash
helm repo add tekton https://charts.tekton.dev
```

## Helm charts

The following charts are available, please look in the chart directories for the documentation of each chart.

| Tekton chart | Chart link |
|---|---|
| Tekton Pipelines | [chart documentation](./pipeline/README.md) |
| Tekton Dashboard | [chart documentation](./dashboard/README.md) |
| Tekton Triggers | [chart documentation](./triggers/README.md) |
| Tekton Operator | TODO |

## Charts versioning

Charts versions use the same major and minor version as the Tekton target component they bootstrap.

Patch is kept to increment on bug fixes or chart improvements.

In any case, the `appVersion` in the chart description is set to the exact version of the full Tekton target component.

## Kubernetes Versions

The kubernetes versions compatible with the charts are driven by the version of Tekton to be deployed.
Please look at the Tekton documentation to know the kubernetes versions supported.

Openshift compatibility should also be supported.

## Helm versions

The charts should be compatilbe with both helm v2 and v3.

Note that some parameters apply to only one version of helm, this will be stated in the chart documentation.
