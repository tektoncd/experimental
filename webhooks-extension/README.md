# Webhooks Extension
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/kubernetes/experimental/blob/master/LICENSE)

The Webhooks Extension for Tekton allows users to set up GitHub webhooks that will trigger Tekton `PipelineRuns` and associated `TaskRuns`. This initial implementation utilises Knative Eventing but we're closely following the eventing discussion in  [Tekton Pipeline](https://github.com/tektoncd/pipeline) to minimize necessary componentry.

In addition to Tekton/Knative Eventing glue, it includes an extension to the Tekton Dashboard.

## Prerequisites
- Install [Tekton](https://github.com/tektoncd/pipeline/blob/master/docs/install.md) _(version 0.3.0 is recommended)_ and the [Tekton Dashboard](https://github.com/tektoncd/dashboard)

- Install Istio - for a quickstart install, for example of version 1.1.8 run `./scripts/install_istio.sh 1.1.8` _or_ follow https://knative.dev/docs/install/installing-istio/ for a more customised install _(Istio version 1.1.8 is recommended)_

- Install Knative Eventing, Eventing Sources & Serving - for a quickstart install, for example of version 0.6.0 run `./scripts/install_knative.sh v0.6.0`, for more detailed instructions see the [Knative docs](https://knative.dev/docs/install/index.html) _(Knative version 0.6.0 is recommended)_

*Running on Docker for Desktop?*

- You must have the `edge` version enabled and supply your local cluster with sufficient resources _(6 CPUs, 10GiB Memory & 2.5GiB swap is known good)_

## Install Quickstart

Until published images are available ([Issue 28](https://github.com/tektoncd/experimental/issues/28)), the webhooks extension must be built and deployed manually following the below steps:

1. Ensure you have set your [GOPATH](https://github.com/golang/go/wiki/SettingGOPATH) correctly

2. Setup for `Ko` to be able to push the extension image:

    `docker login`

    `export KO_DOCKER_REPO=docker.io/[your_dockerhub_id]`

3. Build and deploy the webhooks extension:

    `./scripts/install_webhooks_extension.sh [namespace_to_install_into]`

5. Restart the dashboard pod to register the extension:

    _Step to be removed with [Issue 215](https://github.com/tektoncd/dashboard/issues/215) completion_:

    `kubectl delete pod -l app=tekton-dashboard`

## Install and Development

If you are looking to develop/contribute to this repository and for more involved scripts please see the [development installation guide](https://github.com/tektoncd/experimental/blob/master/webhooks-extension/test/README.md#scripting)

## Running tests

```bash
docker build -f cmd/extension/Dockerfile_test .
```

## API Definitions

- [Extension API definitions](cmd/extension/README.md)
  - The extension API endpoints can be accessed through the dashboard.
- [Sink API definitions](cmd/sink/README.md)

### Example creating a webhook

We would recommend using the extension via the Tekton Dashboard UI - however, if you would prefer to interact with REST endpoint directly, you can create your webhook using curl:

```bash
data='{
  "name": "go-hello-world",
  "namespace": "green",
  "gitrepositoryurl": "https://github.com/ncskier/go-hello-world",
  "accesstoken": "github-secret",
  "pipeline": "simple-pipeline",
  "dockerregistry": "mydockerregistry"
}'
curl -d "${data}" -H "Content-Type: application/json" -X POST http://localhost:8080/webhooks
```

When curling through the dashboard, use the same endpoints; for example, assuming the dashboard is at `localhost:9097`:

```bash
curl -d "${data}" -H "Content-Type: application/json" -X POST http://localhost:9097/webhooks
```

Reference the [Knative eventing GitHub source sample](https://knative.dev/docs/eventing/samples/github-source/) to properly create the `accesstoken` secret. This is the secret that is used to create GitHub webhooks.

## Limitations

- Only GitHub webhooks are currently supported.
- Only `push` and `pull_request` events are currently supported, these are the events defined on the webhook.
- All knative event sources are created in the namespace into which the dashboard and this extension are installed.
- Only one webhook can be created for each Git repository, so each repository will only be able to trigger a `PipelineRun` from one webhook.

- The pipeline definitions that are currently supported are:

  - [simple-pipeline](https://github.com/pipeline-hotel/example-pipelines/blob/master/config/pipeline.yaml)
  - [simple-helm-pipeline](https://github.com/pipeline-hotel/example-pipelines/blob/master/config/helm-pipeline.yaml) (requires a secret to talk to a secure Tiller)
  - [simple-helm-pipeline-insecure](https://github.com/pipeline-hotel/example-pipelines/blob/master/config/helm-insecure-pipeline.yaml.yaml)

## Architecture information

Each webhook that the user creates will store its configuration information in a configmap in the install namespace. This information is used by the sink to create `PipelineRuns` for webhook events.

## Want to get involved

Visit the [Tekton Community](https://github.com/tektoncd/community) project for an overview of our processes.
