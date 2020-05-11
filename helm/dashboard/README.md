

# Tekton Dashboard Helm Chart

The [Tekton Dashboard](https://github.com/tektoncd/dashboard) is a general purpose, web-based UI for [Tekton Pipelines](https://github.com/tektoncd/pipeline).

This helm chart is a lightweight way to deploy, configure and run Tekton Dashboard on a k8s cluster.

## Requirements

* [Helm](https://helm.sh/) v2 or v3
* Kubernetes >= 1.15 (it's driven by the version of Tekton Pipelines installed)
* Depending on the configuration you will need admin access to be able to install the CRDs
* Tekton Pipelines deployed in the target cluster (see [Tekton Pipelines Helm Chart](../pipeline/README.md))

## Description

This chart deploys the Tekton Dashboard. It should run on k8s as well as OpenShift.

It includes various options to create rbac resources, control pods placement and resources, etc...

All options are documented in the [Chart Values](#chart-values) section.

Various configuration examples are document in the [Try it out](#try-it-out) section.

## Installing

- Add the Tekton helm charts repo

**TODO** this is not yet available, maybe document how to install from sources

```bash
helm repo add tekton https://charts.tekton.dev
```

- Install (or upgrade)

```bash
# This will install Tekton Dashboard in the tekton namespace (with a my-dashboard release name)

# Helm v2
helm upgrade --install my-dashboard --namespace tekton tekton/dashboard
# Helm v3
helm upgrade --install my-dashboard --namespace tekton tekton/dashboard --set customResourceDefinitions.create=false
```

- Install (or upgrade) without CRDs (assuming CRDs have already been deployed by an admin)

```bash
# This will install Tekton Dashboard in the tekton namespace (with a my-dashboard release name)

# Helm v2
helm upgrade --install my-dashboard --namespace tekton tekton/dashboard --set customResourceDefinitions.create=false
# Helm v3
helm upgrade --install my-dashboard --namespace tekton tekton/dashboard --set customResourceDefinitions.create=false --skip-crds
```

- Install (or upgrade) without creating RBAC resources (assuming RBAC resources have been created by an admin)

```bash
# This will install Tekton Dashboard in the tekton namespace (with a my-dashboard release name)

# Helm v2
helm upgrade --install my-dashboard --namespace tekton tekton/dashboard --set rbac.create=false --set rbac.serviceAccountName=svcAccountName
# Helm v3
helm upgrade --install my-dashboard --namespace tekton tekton/dashboard --set customResourceDefinitions.create=false --set rbac.create=false --set rbac.serviceAccountName=svcAccountName
```

Look [below](#chart-values) for the list of all available options and their corresponding description.

## Uninstalling

To uninstall the chart, simply delete the release.

```bash
# This will uninstall Tekton Dashboard in the tekton namespace (assuming a my-dashboard release name)

# Helm v2
helm delete --purge my-dashboard
# Helm v3
helm uninstall my-dashboard --namespace tekton
```

## Version

Current chart version is `0.6.4`

## Chart Values


| Key | Type | Description | Default |
|-----|------|-------------|---------|
| `customResourceDefinitions.create` | bool | Create CRDs | `true` |
| `dashboard.affinity` | object | Dashboard affinity rules | `{}` |
| `dashboard.annotations` | object | Dashboard pod annotations | See [values.yaml](./values.yaml) |
| `dashboard.image.pullPolicy` | string | Dashboard docker image pull policy | `"IfNotPresent"` |
| `dashboard.image.repository` | string | Dashboard docker image repository | `"gcr.io/tekton-releases/github.com/tektoncd/dashboard/cmd/dashboard"` |
| `dashboard.image.tag` | string | Dashboard docker image tag | `"v0.6.1"` |
| `dashboard.ingress.annotations` | object | Dashboard ingress annotations | `{}` |
| `dashboard.ingress.enabled` | bool | Enable dashboard ingress | `false` |
| `dashboard.ingress.hosts` | list | Dashboard ingress host names | `[]` |
| `dashboard.nodeSelector` | object | Dashboard node selector | `{}` |
| `dashboard.readOnly` | bool | Drives running the dashboard in read only mode | `false` |
| `dashboard.resources` | object | Dashboard resource limits and requests | `{}` |
| `dashboard.securityContext` | object | Dashboard pods security context | `{}` |
| `dashboard.service.annotations` | object | Dashboard service annotations | `{}` |
| `dashboard.service.port` | int | Dashboard service port | `9097` |
| `dashboard.service.portName` | string |  | `"http"` |
| `dashboard.service.type` | string | Dashboard service type | `"ClusterIP"` |
| `dashboard.tolerations` | list | Dashboard tolerations | `[]` |
| `extensions` | list | Kubernetes resources extensions list | See [values.yaml](./values.yaml) |
| `fullnameOverride` | string | Fully override resource generated names | `""` |
| `nameOverride` | string | Partially override resource generated names | `""` |
| `rbac.create` | bool | Create RBAC resources | `true` |
| `rbac.serviceAccountName` | string | Name of the service account to use when rbac.create is false | `nil` |
| `version` | string | Tekton dashboard version used to add labels on deployments, pods and services | `"v0.6.1"` |


You can look directly at the [values.yaml](./values.yaml) file to look at the options and their default values.

## Try it out

This chart should deploy correctly with default values.

You will find examples below of how to customize the deployment of a release with various options. The list of examples is by no means exhaustive, it tries to cover the most used cases.

If you feel something is incomplete, missing or incorrect please open an issue and we'll do our best to improve this documentation.

### Deploy Tekton Pipelines

Before deploying Tekton Dashboard you should have Tekton Pipelines deployed.

You can use Helm to deploy Tekton Pipelines on you cluster, see [Installing](../pipeline/README.md#installing) instructions.

### Configure pod resources

Create a yaml file called `pod-resources.yaml` looking like this (the name doesn't really matters):

```yaml
dashboard:
  resources:
    requests:
      cpu: 0.5
      memory: 128m
    limits:
      cpu: 1
      memory: 256m
```

Use the previously created file to pass the values to helm:

```bash
# This will install Tekton Dashboard in the tekton namespace (with a my-dashboard release name)

# Helm v2
helm upgrade --install my-dashboard --namespace tekton tekton/dashboard --values pod-resources.yaml
# Helm v3
helm upgrade --install my-dashboard --namespace tekton tekton/dashboard --values pod-resources.yaml --set customResourceDefinitions.create=false
```

### Deploy a read only dashboard

```bash
# This will install Tekton Dashboard in the tekton namespace (with a my-dashboard release name)

# Helm v2
helm upgrade --install my-dashboard --namespace tekton tekton/dashboard --set dashboard.readOnly=true
# Helm v3
helm upgrade --install my-dashboard --namespace tekton tekton/dashboard --set dashboard.readOnly=true --set customResourceDefinitions.create=false
```

---

Except as otherwise noted, the content of this page is licensed under the
[Creative Commons Attribution 4.0 License](https://creativecommons.org/licenses/by/4.0/),
and code samples are licensed under the
[Apache 2.0 License](https://www.apache.org/licenses/LICENSE-2.0).
