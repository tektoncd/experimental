# Webhooks Extension
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/kubernetes/experimental/blob/master/LICENSE)

The Webhooks Extension for Tekton allows users to set up Git webhooks that will trigger Tekton PipelineRuns and TaskRuns. An initial implementation will use Knative Eventing but we're closely following the eventing discussion in  [Tekton Pipeline](https://github.com/tektoncd/pipeline) to minimize necessary componentry.

In addition to Tekton/Knative Eventing glue, it includes an extension to the Tekton Dashboard.

## Runtime Dependencies
- [Tekton](https://github.com/tektoncd/pipeline) & [Tekton Dashboard](https://github.com/tektoncd/dashboard)
- Knative [eventing](https://knative.dev/docs/eventing/), [eventing sources](https://knative.dev/docs/eventing/sources/), & [serving](https://knative.dev/docs/serving/)

## Install
Please see the [development installation guide](https://github.com/tektoncd/experimental/blob/master/webhooks-extension/test/README.md#scripting)

_Note: The main dashboard user interface (UI) finds registered extension UIs when the main dashboard pod starts.  If you have installed this extension after you installed the dashboard you will need to restart the dashboard pod (`kubectl delete pod -l app=tekton-dashboard`). This only needs doing until https://github.com/tektoncd/dashboard/issues/215 is completed such that the dashboard dynamically finds extensions._

## Running tests

```bash
docker build -f cmd/extension/Dockerfile_test .
```

## API Definitions

- [Extension API definitions](cmd/extension/README.md)
  - The extension API endpoints can be accessed through the dashboard.
- [Sink API definitions](cmd/sink/README.md)

### Example creating a webhook

You should be able to use the extension via the Tekton Dashboard UI - however, until the UI is coded or if you would prefer to interact with REST endpoint directly, you can create your webhook using curl:

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

- Only Git Hub webhooks are currently supported.
- Only `push` and `pull_request` events are supported at the moment. The webhook is currently only created with both `push` and `pull_request` events.
- All knative event sources are created in the namespace into which the dashboard and this extension are installed.
- Only one webhook can be created for each git repository, so each repository will only be able to trigger a PipelineRun from one webhook.

- Three pipeline definitions are currently supported.

  - [simple-pipeline](https://github.com/pipeline-hotel/example-pipelines/blob/master/config/pipeline.yaml)
  - [simple-helm-pipeline](https://github.com/pipeline-hotel/example-pipelines/blob/master/config/helm-pipeline.yaml) (requires a secret to talk to a secure Tiller)
  - [simple-helm-pipeline-insecure](https://github.com/pipeline-hotel/example-pipelines/blob/master/config/helm-insecure-pipeline.yaml.yaml)

## Architecture information

Each webhook that the user creates will store its configuration information as a configmap in the install namespace. The information is used later by the sink to create PipelineRuns for webhook events.

## Want to get involved

Visit the [Tekton Community](https://github.com/tektoncd/community) project for an overview of our processes.
