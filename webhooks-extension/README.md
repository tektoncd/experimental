# Webhooks Extension
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/kubernetes/experimental/blob/master/LICENSE)

The Webhooks Extension for Tekton provides allows users to set up Git webhooks that will trigger Tekton PipelineRuns and TaskRuns. An initial implementation will use Knative Eventing but we're closely following the eventing discussion in  [Tekton Pipeline](https://github.com/tektoncd/pipeline) to minimize necessary componentry.

In addition to Tekton/Knative Eventing glue, it includes an extension to the Tekton dashboard.

## Dependencies

This project requires Knative [eventing](https://knative.dev/docs/eventing/), [eventing sources](https://knative.dev/docs/eventing/sources/), and [serving](https://knative.dev/docs/serving/). Install these components [here](https://knative.dev/docs/install/).

## Install

Verify that you have installed the [dependencies](#dependencies).

Until such a time an official release is hosted you will need to use the development process to install this extension.

For convenience, a script has been provided. The script will build the image, push to your registry of choice and then kubectl apply the relevant yaml into the specified namespace. Please note this namespace needs to be the namespace into which you have installed the tekton dashboard.

To initiate this installation, run `development_install.sh`.

_Note: Your git provider must be able to reach the address of this extension's sink. The sink is deployed with a Knative service, so you may need to configure Knative serving. We recommend [setting up a custom domain](https://knative.dev/v0.3-docs/serving/using-a-custom-domain/) with the extension `.nip.io`._

## API Definitions

- [Extension API definitions](cmd/extension/README.md)
  - The extension API endpoints can be accessed through the dashboard.
- [Sink API definitions](cmd/sink/README.md)

### Example creating a webhook

You should be able to use the extension via the Tekton dashboard UI - however, until the UI is coded or if you would prefer to interact with REST endpoint directly, you can create your webhook using curl:

```bash
data='{
  "name": "go-hello-world",
  "namespace": "green",
  "gitrepositoryurl": "https://github.com/ncskier/go-hello-world",
  "accesstoken": "github-secret",
  "pipeline": "simple-pipeline"
}'
curl -d "${data}" -H "Content-Type: application/json" -X POST http://localhost:8080/webhooks-extension/webhooks
```

When curling through the dashboard, use the same endpoints; for example, assuming the dashboard is at `localhost:9097`:

```bash
curl -d "${data}" -H "Content-Type: application/json" -X POST http://localhost:9097/webhooks-extension/webhooks
```

Reference the [Knative eventing GitHub source sample](https://knative.dev/docs/eventing/samples/github-source/) to properly create the `accesstoken` secret. This is the secret that is used to create GitHub webhooks.

## Limitations

- Only GitHub webhooks are currently supported.
- Only `push` and `pull_request` events are supported at the moment. The webhook is currently only created with both `push` and `pull_request` events.
- All knative event sources are created in the namespace into which the dashboard and this extension are installed.
- Currently the docker registry to which built images are pushed is hard coded from the registry you specified at install time, there is work underway to change this restriction.
- Only one webhook can be created for each git repository, so each repository will only be able to trigger a PipelineRun from one webhook.
- Only the [simple-pipeline](https://github.com/pipeline-hotel/example-pipelines/blob/master/config/pipeline.yaml) Pipeline definition is currently supported.

## Architecture information

Each webhook that the user creates will store its configuration information as a configmap in the install namespace. The information is used later by the sink to create PipelineRuns for webhook events.

## Want to get involved

Visit the [Tekton Community](https://github.com/tektoncd/community) project for an overview of our processes.
