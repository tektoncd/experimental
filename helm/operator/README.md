

# Tekton Operator Helm Chart

TODO

## Requirements

* [Helm](https://helm.sh/) v2 or v3
* Kubernetes >= 1.15 (it's driven by the version of Tekton Pipelines installed)
* Depending on the configuration you will need admin access to be able to install the CRDs

## Description

TODO

## Installing

- Add the Tekton helm charts repo

**TODO** this is not yet available, maybe document how to install from sources

```bash
helm repo add tekton https://charts.tekton.dev
```

- Install (or upgrade)

```bash
# This will install Tekton Operator in the tekton namespace (with a my-operator release name)

# Helm v2
helm upgrade --install my-operator --namespace tekton tekton/operator
# Helm v3
helm upgrade --install my-operator --namespace tekton tekton/operator --set customResourceDefinitions.create=false
```

- Install (or upgrade) without CRDs (assuming CRDs have already been deployed by an admin)

```bash
# This will install Tekton Operator in the tekton namespace (with a my-operator release name)

# Helm v2
helm upgrade --install my-operator --namespace tekton tekton/operator --set customResourceDefinitions.create=false
# Helm v3
helm upgrade --install my-operator --namespace tekton tekton/operator --set customResourceDefinitions.create=false --skip-crds
```

- Install (or upgrade) without creating RBAC resources (assuming RBAC resources have been created by an admin)

```bash
# This will install Tekton Operator in the tekton namespace (with a my-operator release name)

# Helm v2
helm upgrade --install my-operator --namespace tekton tekton/operator --set rbac.create=false --set rbac.serviceAccountName=svcAccountName
# Helm v3
helm upgrade --install my-operator --namespace tekton tekton/operator --set customResourceDefinitions.create=false --set rbac.create=false --set rbac.serviceAccountName=svcAccountName
```

Look [below](#chart-values) for the list of all available options and their corresponding description.

## Uninstalling

To uninstall the chart, simply delete the release.

```bash
# This will uninstall Tekton Operator in the tekton namespace (assuming a my-operator release name)

# Helm v2
helm delete --purge my-operator
# Helm v3
helm uninstall my-operator --namespace tekton
```

## Version

Current chart version is `0.0.0`

## Chart Values


| Key | Type | Description | Default |
|-----|------|-------------|---------|
| `controller.affinity` | object | Operator affinity rules | `{}` |
| `controller.annotations` | object | Operator pod annotations | See [values.yaml](./values.yaml) |
| `controller.image.pullPolicy` | string | Operator docker image pull policy | `"IfNotPresent"` |
| `controller.image.repository` | string | Operator docker image repository | `"gcr.io/tekton-releases/github.com/tektoncd/operator/cmd/manager"` |
| `controller.image.tag` | string | Operator docker image tag | `"v0.6.0"` |
| `controller.nodeSelector` | object | Operator node selector | `{}` |
| `controller.readOnly` | bool | Drives running the Operator in read only mode | `false` |
| `controller.resources` | object | Operator resource limits and requests | `{}` |
| `controller.securityContext` | object | Operator pods security context | `{}` |
| `controller.tolerations` | list | Operator tolerations | `[]` |
| `customResourceDefinitions.create` | bool | Create CRDs | `true` |
| `fullnameOverride` | string | Fully override resource generated names | `""` |
| `nameOverride` | string | Partially override resource generated names | `""` |
| `rbac.create` | bool | Create RBAC resources | `true` |
| `rbac.serviceAccountName` | string | Name of the service account to use when rbac.create is false | `nil` |
| `version` | string | Tekton triggers version used to add labels on deployments, pods and services | `"v0.0.0"` |


You can look directly at the [values.yaml](./values.yaml) file to look at the options and their default values.

## Try it out

TODO

---

Except as otherwise noted, the content of this page is licensed under the
[Creative Commons Attribution 4.0 License](https://creativecommons.org/licenses/by/4.0/),
and code samples are licensed under the
[Apache 2.0 License](https://www.apache.org/licenses/LICENSE-2.0).
