# CloudEvents Controller for Tekton

The CloudEvents controller for Tekton provides CloudEvents for the lifecycle of
the following Tekton Resources:

- [`PipelineRuns`](https://tekton.dev/docs/pipelines/pipelineruns/)

[CloudEvents](https://cloudevents.io/) is a CNCF project that provides:

> "A specification for describing event data in a common way"

The CloudEvents controller supports sending events according to two different
specifications:

- [Tekton CloudEvents](https://tekton.dev/docs/pipelines/events/#events-via-cloudevents):
  this spec has been defined by the Tekton project in the absence of a common
  standard for events in CI/CD, and may be deprecated in future.
- [CDEvents](https://cdevents.dev): this spec is a CDF incubated project. Its
  v0.1 version is work in progress, and so is CDEvents support in this
  controller.

## Motivation

CloudEvents support in Tekton today is implement in the specific resource
controllers. Having a central controller offloads this work from the controllers
and allows to implement new features that can be shared across resources.

## Install

Install and configure `ko`.

```shell
ko apply -f config/
```

This will build and install the controller on your cluster, in the namespace
`tekton-cloudevents`.

Alternatively, install the latest nightly build:

```shell
kubectl apply -f https://storage.cloud.google.com/tekton-releases-nightly/cloudevents/latest/release.yaml
```

## Configuration

The controller defines several configuration flags in the `config-defaults`
config map:

| Key Name | Description | Default Value |
|----------|-------------|---------------|
| `default-cloud-events-sink` | The URL of the sink where events are delivered. When empty no events are sent | "" |
| `default-cloud-events-format` | Which format of CloudEvents to send, `legacy` or `cdevents` | `cdevents` |

Logging format and verbosity can be configured in the `config-logging` config map.

## Uninstall

```shell
$ kubectl delete namespace tekton-cloudevents
namespace "tekton-cloudevents" deleted
```

This will stop the controller and delete its namespace.
