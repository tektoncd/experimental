

# Tekton Pipelines Helm Chart

The [Tekton Pipelines](https://github.com/tektoncd/pipeline) project provides k8s-style resources for declaring CI/CD-style pipelines.

This helm chart is a lightweight way to deploy, configure and run Tekton Pipelines on a k8s cluster.

## Requirements

* [Helm](https://helm.sh/) v2 or v3
* Kubernetes >= 1.15 (it's driven by the version of Tekton Pipelines installed)
* Depending on the configuration you will need admin access to be able to install the CRDs

## Description

This chart deploys the Tekton Pipelines controller and optionnaly the associated webhook (it's strongly recommended to deploy both). It should run on k8s as well as OpenShift.

It includes various options to expose metrics and/or profiling endpoints, create rbac resources, run in high availabilty mode, control pods placement and resources, etc...

All options are documented in the [Chart Values](#chart-values) section.

Various configuration examples are document in the [Try it out](#try-it-out) section.

An additional guide is available in the [Production grade configuration](#production-grade-configuration) section to help deploying Tekton Pipelines in a highlly available and secure mode.

## Installing

- Add the Tekton helm charts repo

**TODO** this is not yet available, maybe document how to install from sources

```bash
helm repo add tekton https://charts.tekton.dev
```

- Install (or upgrade)

```bash
# This will install Tekton Pipelines in the tekton namespace (with a my-pipeline release name)

# Helm v2
helm upgrade --install my-pipeline --namespace tekton tekton/pipeline
# Helm v3
helm upgrade --install my-pipeline --namespace tekton tekton/pipeline --set customResourceDefinitions.create=false
```

- Install (or upgrade) without CRDs (assuming CRDs have already been deployed by an admin)

```bash
# This will install Tekton Pipelines in the tekton namespace (with a my-pipeline release name)

# Helm v2
helm upgrade --install my-pipeline --namespace tekton tekton/pipeline --set customResourceDefinitions.create=false
# Helm v3
helm upgrade --install my-pipeline --namespace tekton tekton/pipeline --set customResourceDefinitions.create=false --skip-crds
```

- Install (or upgrade) without creating RBAC resources (assuming RBAC resources have been created by an admin)

```bash
# This will install Tekton Pipelines in the tekton namespace (with a my-pipeline release name)

# Helm v2
helm upgrade --install my-pipeline --namespace tekton tekton/pipeline --set rbac.create=false --set rbac.serviceAccountName=svcAccountName
# Helm v3
helm upgrade --install my-pipeline --namespace tekton tekton/pipeline --set customResourceDefinitions.create=false --set rbac.create=false --set rbac.serviceAccountName=svcAccountName
```

Look [below](#chart-values) for the list of all available options and their corresponding description.

## Uninstalling

To uninstall the chart, simply delete the release.

```bash
# This will uninstall Tekton Pipelines in the tekton namespace (assuming a my-pipeline release name)

# Helm v2
helm delete --purge my-pipeline
# Helm v3
helm uninstall my-pipeline --namespace tekton
```

## Version

Current chart version is `0.0.1`

## Chart Values


| Key | Type | Description | Default |
|-----|------|-------------|---------|
| `controller.affinity` | object | Controller affinity rules | `{}` |
| `controller.annotations` | object | Controller pod annotations | See [values.yaml](./values.yaml) |
| `controller.args` | list | Controller arguments | See [values.yaml](./values.yaml) |
| `controller.config.artifactBucket` | object | Controller configuration for artifact bucket (see https://github.com/tektoncd/pipeline/blob/master/docs/install.md) | See [values.yaml](./values.yaml) |
| `controller.config.artifactPvc` | object | Controller configuration for artifact pvc (see https://github.com/tektoncd/pipeline/blob/master/docs/install.md) | See [values.yaml](./values.yaml) |
| `controller.config.defaults` | object | Controller configuration for default values (see https://github.com/tektoncd/pipeline/blob/master/docs/install.md) | See [values.yaml](./values.yaml) |
| `controller.config.featureFlags` | object | Controller configuration for feature flags | See [values.yaml](./values.yaml) |
| `controller.config.leaderElection` | object | Controller configuration for leader election | See [values.yaml](./values.yaml) |
| `controller.config.logging` | object | Controller configuration for logging (see https://github.com/tektoncd/pipeline/blob/master/docs/install.md) | See [values.yaml](./values.yaml) |
| `controller.config.observability` | object | Controller configuration for observability (see https://github.com/tektoncd/pipeline/blob/master/docs/install.md) | See [values.yaml](./values.yaml) |
| `controller.image.pullPolicy` | string | Controller docker image pull policy | `"IfNotPresent"` |
| `controller.image.repository` | string | Controller docker image repository | `"gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/controller"` |
| `controller.image.tag` | string | Controller docker image tag | `"v0.11.1"` |
| `controller.metrics.enabled` | bool | Enable controller metrics service | `true` |
| `controller.metrics.port` | int | Controller metrics service port | `9090` |
| `controller.metrics.portName` | string |  | `"metrics"` |
| `controller.nodeSelector` | object | Controller node selector | `{}` |
| `controller.resources` | object | Controller resource limits and requests | `{}` |
| `controller.securityContext` | object | Controller pods security context | `{}` |
| `controller.service.annotations` | object | Controller service annotations | `{}` |
| `controller.service.type` | string | Controller service type | `"ClusterIP"` |
| `controller.tolerations` | list | Controller tolerations | `[]` |
| `customResourceDefinitions.create` | bool | Create CRDs | `true` |
| `fullnameOverride` | string | Fully override resource generated names | `""` |
| `nameOverride` | string | Partially override resource generated names | `""` |
| `podSecurityPolicy.enabled` | bool | Enable pod security policy | `false` |
| `rbac.create` | bool | Create RBAC resources | `true` |
| `rbac.serviceAccountName` | string | Name of the service account to use when rbac.create is false | `nil` |
| `version` | string | Tekton pipelines version used to add labels on deployments, pods and services | `"v0.11.1"` |
| `webhook.affinity` | object | Webhook affinity rules | `{}` |
| `webhook.annotations` | object | Webhook pod annotations | See [values.yaml](./values.yaml) |
| `webhook.enabled` | bool | Enable webhook | `true` |
| `webhook.image.pullPolicy` | string | Webhook docker image pull policy | `"IfNotPresent"` |
| `webhook.image.repository` | string | Webhook docker image repository | `"gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/webhook"` |
| `webhook.image.tag` | string | Webhook docker image tag | `"v0.11.1"` |
| `webhook.metrics.enabled` | bool | Enable webhook metrics service | `true` |
| `webhook.metrics.port` | int | Webhook metrics service port | `9090` |
| `webhook.metrics.portName` | string | Webhook metrics service port name | `"http-metrics"` |
| `webhook.nodeSelector` | object | Webhook node selector | `{}` |
| `webhook.podDisruptionBudget.ennabled` | bool |  | `false` |
| `webhook.podDisruptionBudget.maxUnavailable` | int | Maximum unavailable webhook pods | `1` |
| `webhook.podDisruptionBudget.minAvailable` | int | Minimum available webhook pods | `1` |
| `webhook.profiling.enabled` | bool | Enable pebhook profiling service | `true` |
| `webhook.profiling.port` | int | Webhook profiling service port | `8008` |
| `webhook.profiling.portName` | string | Webhook profiling service port name | `"http-profiling"` |
| `webhook.replicas` | int | Webhook replicas | `1` |
| `webhook.resources` | object | Webhook resource limits and requests | `{}` |
| `webhook.securityContext` | object | Webhook pods security context | `{}` |
| `webhook.service.annotations` | object | Webhook service annotations | `{}` |
| `webhook.service.type` | string | Webhook service type | `"ClusterIP"` |
| `webhook.tolerations` | list | Webhook tolerations | `[]` |
| `webhook.updateStrategy` | object | Webhook pods update strategy | `{}` |


You can look directly at the [values.yaml](./values.yaml) file to look at the options and their default values.

## Try it out

This chart should deploy correctly with default values.

You will find examples below of how to customize the deployment of a release with various options. The list of examples is by no means exhaustive, it tries to cover the most used cases.

If you feel something is incomplete, missing or incorrect please open an issue and we'll do our best to improve this documentation.

### Disable webhook deployment (not recommended)

This will prevent validation and resource updates if using an old version of the CRDs.

```bash
# This will install Tekton Pipelines in the tekton namespace (with a my-pipeline release name)

# Helm v2
helm upgrade --install my-pipeline --namespace tekton tekton/pipeline --set webhook.enabled=false
# Helm v3
helm upgrade --install my-pipeline --namespace tekton tekton/pipeline --set webhook.enabled=false --set customResourceDefinitions.create=false
```

### Configure artifact bucket

Look at the [Installing Tekton Pipelines](https://github.com/tektoncd/pipeline/blob/master/docs/install.md) doc for more informations about the content of this config map.

Create a yaml file called `config-artifact-bucket.yaml` looking like this (the name doesn't really matters):

```yaml
location: s3://my-artifact-bucket
bucket.service.account.secret.name: my-secret
bucket.service.account.secret.key: boto-config
bucket.service.account.field.name: BOTO_CONFIG
```

Use the previously created file to pass the configuration to helm:

```bash
# This will install Tekton Pipelines in the tekton namespace (with a my-pipeline release name)

# Helm v2
helm upgrade --install my-pipeline --namespace tekton tekton/pipeline --set-file controller.config.artifactBucket=config-artifact-bucket.yaml
# Helm v3
helm upgrade --install my-pipeline --namespace tekton tekton/pipeline --set-file controller.config.artifactBucket=config-artifact-bucket.yaml --set customResourceDefinitions.create=false
```

### Configure artifact pvc

Look at the [Installing Tekton Pipelines](https://github.com/tektoncd/pipeline/blob/master/docs/install.md) doc for more informations about the content of this config map.

Create a yaml file called `config-artifact-pvc.yaml` looking like this (the name doesn't really matters):

```yaml
size: 1Gi
```

Use the previously created file to pass the configuration to helm:

```bash
# This will install Tekton Pipelines in the tekton namespace (with a my-pipeline release name)

# Helm v2
helm upgrade --install my-pipeline --namespace tekton tekton/pipeline --set-file controller.config.artifactPvc=config-artifact-pvc.yaml
# Helm v3
helm upgrade --install my-pipeline --namespace tekton tekton/pipeline --set-file controller.config.artifactPvc=config-artifact-pvc.yaml --set customResourceDefinitions.create=false
```

### Configure defaults

Look at the [Installing Tekton Pipelines](https://github.com/tektoncd/pipeline/blob/master/docs/install.md) doc for more informations about the content of this config map.

Create a yaml file called `config-artifact-defaults.yaml` looking like this (the name doesn't really matters):

```yaml
default-service-account: my-service-account
default-pod-template: |
  nodeSelector:
    kops.k8s.io/instancegroup: my-node-group
```

Use the previously created file to pass the configuration to helm:

```bash
# This will install Tekton Pipelines in the tekton namespace (with a my-pipeline release name)

# Helm v2
helm upgrade --install my-pipeline --namespace tekton tekton/pipeline --set-file controller.config.defaults=config-artifact-defaults.yaml
# Helm v3
helm upgrade --install my-pipeline --namespace tekton tekton/pipeline --set-file controller.config.defaults=config-artifact-defaults.yaml --set customResourceDefinitions.create=false
```

### Other config maps

Same thing applies for other config maps.

Find below the list of supported config maps and their corresponding config key:

| Config map | Config key | Official documentation
|---|---|---|
| artifact bucket | `controller.config.artifactBucket` | [Installing Tekton Pipelines](https://github.com/tektoncd/pipeline/blob/master/docs/install.md) |
| artifact pvc | `controller.config.artifactPvc` | [Installing Tekton Pipelines](https://github.com/tektoncd/pipeline/blob/master/docs/install.md) |
| defaults | `controller.config.defaults` | [Installing Tekton Pipelines](https://github.com/tektoncd/pipeline/blob/master/docs/install.md) |
| feature flags | `controller.config.featureFlags` | [Installing Tekton Pipelines](https://github.com/tektoncd/pipeline/blob/master/docs/install.md) |
| leader election | `controller.config.leaderElection` | [Installing Tekton Pipelines](https://github.com/tektoncd/pipeline/blob/master/docs/install.md) |
| logging | `controller.config.logging` | [Installing Tekton Pipelines](https://github.com/tektoncd/pipeline/blob/master/docs/install.md) |
| observability | `controller.config.observability` | [Installing Tekton Pipelines](https://github.com/tektoncd/pipeline/blob/master/docs/install.md) |

Please look in [values.yaml](./values.yaml) to find the default values for each config map.

### Configure Pod Resources

Controller and Webhook pod resources are configured independently.

Create a yaml file called `pod-resources.yaml` looking like this (the name doesn't really matters):

```yaml
controller:
  resources:
    requests:
      cpu: 0.5
      memory: 128m
    limits:
      cpu: 1
      memory: 256m
webhook:
  resources:
    requests:
      cpu: 0.2
      memory: 100m
    limits:
      cpu: 0.5
      memory: 200m
```

Use the previously created file to pass the values to helm:

```bash
# This will install Tekton Pipelines in the tekton namespace (with a my-pipeline release name)

# Helm v2
helm upgrade --install my-pipeline --namespace tekton tekton/pipeline --values pod-resources.yaml
# Helm v3
helm upgrade --install my-pipeline --namespace tekton tekton/pipeline --values pod-resources.yaml --set customResourceDefinitions.create=false
```

### Configure number of webhook replicas

Only Webhook pod replicas can be configured, the controller doesn't support more than 1 replica.

```bash
# This will install Tekton Pipelines in the tekton namespace (with a my-pipeline release name)

# Helm v2
helm upgrade --install my-pipeline --namespace tekton tekton/pipeline --set webhook.replicas=3
# Helm v3
helm upgrade --install my-pipeline --namespace tekton tekton/pipeline --set webhook.replicas=3 --set customResourceDefinitions.create=false
```

### Enable prometheus scraping

To let prometheus scrape the metrics endpoints, we need to set annotations on the controller and/or webhook services.

This can be done using the `controller.service.annotations` and `webhook.service.annotations` options.

Create a yaml file called `service-annotations.yaml` looking like this (the name doesn't really matters):

```yaml
controller:
  service:
    annotations:
      prometheus.io/scrape: 'true'
      prometheus.io/port: '9090'
webhook:
  service:
    annotations:
      prometheus.io/scrape: 'true'
      prometheus.io/port: '9090'
```

Use the previously created file to pass the values to helm:

```bash
# This will install Tekton Pipelines in the tekton namespace (with a my-pipeline release name)

# Helm v2
helm upgrade --install my-pipeline --namespace tekton tekton/pipeline --values service-annotations.yaml
# Helm v3
helm upgrade --install my-pipeline --namespace tekton tekton/pipeline --values service-annotations.yaml --set customResourceDefinitions.create=false
```

## Production grade configuration

An example configuration is available in [values-production.yaml](./values-production.yaml).

Deploy with:

```bash
# This will install Tekton Pipelines in the tekton namespace (with a my-pipeline release name)

# Helm v2
helm upgrade --install my-pipeline --namespace tekton tekton/pipeline --values values-production.yaml
# Helm v3
helm upgrade --install my-pipeline --namespace tekton tekton/pipeline --values values-production.yaml --set customResourceDefinitions.create=false
```

- Enable pod security policy

```yaml
podSecurityPolicy:
  enabled: true
```

- Configure controller/webhook pod resources

Depending on the size and load of your cluster, requests and/or limits values will need to be adjusted.

```yaml
controller:
  resources:
    requests:
      cpu: 0.5
      memory: 128m
    limits:
      cpu: 0.5
      memory: 128m
webhook:
  resources:
    requests:
      cpu: 0.5
      memory: 128m
    limits:
      cpu: 0.5
      memory: 128m
```

- Prevent cluster autoscaler to evict controller

```yaml
controller:
  annotations:
    cluster-autoscaler.kubernetes.io/safe-to-evict: 'false'
```

- Enable metrics scraping

```yaml
controller:
  service:
    annotations:
      prometheus.io/scrape: 'true'
      prometheus.io/port: '9090'
  metrics:
    enabled: true
    port: 9090
    portName: metrics
webhook:
  service:
    annotations:
      prometheus.io/scrape: 'true'
      prometheus.io/port: '9090'
  metrics:
    enabled: true
    port: 9090
    portName: metrics
```

- Configure webhook replicas and affinity

Depending on your k8s platform and cluster topology, you should ensure that more than one webhook pod is running.

Webhook pods should be distributed across data centers/availability zones.

```yaml
webhook:
  replicas: 3
  affinity:
    podAntiAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
        - labelSelector:
            matchLabels:
              app.kubernetes.io/component: webhook
              app.kubernetes.io/instance: my-pipeline
          topologyKey: failure-domain.beta.kubernetes.io/zone
```

- Configure webhook pod disruption budget and update strategy

To ensure there is always a minimum number of webhook pods running, you should configure a pod disruption budget.

```yaml
webhook:
  podDisruptionBudget:
    ennabled: true
    minAvailable: 1
    maxUnavailable: 1
  updateStrategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1
      maxSurge: 1
```

---

Except as otherwise noted, the content of this page is licensed under the
[Creative Commons Attribution 4.0 License](https://creativecommons.org/licenses/by/4.0/),
and code samples are licensed under the
[Apache 2.0 License](https://www.apache.org/licenses/LICENSE-2.0).
