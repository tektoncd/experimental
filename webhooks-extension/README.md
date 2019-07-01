# Webhooks Extension

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/kubernetes/experimental/blob/master/LICENSE)

The Webhooks Extension for Tekton allows users to set up GitHub webhooks that will trigger Tekton `PipelineRuns` and associated `TaskRuns`.

This initial implementation utilises Knative Eventing but there is discussion and work in [Tekton Pipelines](https://github.com/tektoncd/pipeline) to minimize necessary componentry in future.

In addition to Tekton/Knative Eventing glue, it includes an extension to the Tekton Dashboard.

## Prerequisites


- Install [Tekton Pipelines](https://github.com/tektoncd/pipeline/blob/master/docs/install.md) and the [Tekton Dashboard](https://github.com/tektoncd/dashboard)

- Install Istio - for a quickstart install, for example of version 1.1.8 run `./scripts/install_istio.sh 1.1.8` _or_ follow https://knative.dev/docs/install/installing-istio/ for a more customised install _(Istio version 1.1.8 is recommended)_

- Install Knative Eventing, Eventing Sources & Serving - for a quickstart install, for example of version 0.6.0 run `./scripts/install_knative.sh v0.6.0`, for more detailed instructions see the [Knative docs](https://knative.dev/docs/install/index.html) _(Knative version 0.6.0 is recommended)_

*If running on Docker for Desktop*

- Knative requires a Kubernetes cluster running version v.1.11 or greater. Currently this requires the edge version of Docker for Desktop. Your cluster must also be supplied with sufficient resources _(6 CPUs, 10GiB Memory & 2.5GiB swap is known good)_.

## Install Quickstart

### Domain setup for Knative Serving

Set your own domain and selectors following the [configuring Knative Serving docs](https://github.com/knative/serving/blob/master/install/CONFIG.md) which outline setting up routes in the `config-domain` ConfigMap in the `knative-serving` namespace

On Docker Desktop, you can retrieve your IP and patch it to the ConfigMap by running:

`ip=$(ifconfig | grep netmask | sed -n 2p | cut -d ' ' -f2)`

```
kubectl patch configmap config-domain --namespace knative-serving --type='json' \
  --patch '[{"op": "add", "path": "/data/'"${ip}.nip.io"'", "value": ""}]'
```

### Install the Webhooks Extension 

The Tekton Webhooks Extension has hosted images located at `gcr.io/tekton-nightly/extension:latest` and `gcr.io/tekton-nightly/extension:latest`, to install the latest extension and sink using these images:

`kubectl apply -f config/release/gcr-tekton-webhooks-extension.yaml`

### Access the Extension through the Dashboard UI 

Restart the dashboard to register the extension:

`kubectl delete pod -l app=tekton-dashboard`

Access the Dashboard through its ClusterIP Service by running `kubectl proxy`. Assuming tekton-pipelines is the install namespace for the dashboard, you can access the web UI at localhost:8001/api/v1/namespaces/tekton-pipelines/services/tekton-dashboard:http/proxy/ 

Navigate to `Webhooks`, listed in the navigation under `Extensions`

## Uninstall

`kubectl delete -f config/release/gcr-tekton-webhooks-extension.yaml`

## Limitations

- Only GitHub webhooks are currently supported.
- Only `push` and `pull_request` events are currently supported, these are the events defined on the webhook.
- Only one webhook can be created for each Git repository, so each repository will only be able to trigger a `PipelineRun` from one webhook.

## Want to get involved

Visit the [Tekton Community](https://github.com/tektoncd/community) project for an overview of our processes.

## For developers

If you are looking to develop or contribute to this repository please see the [development docs](https://github.com/tektoncd/experimental/blob/master/webhooks-extension/DEVELOPMENT.md)

For more involved development scripts please see the [development installation guide](https://github.com/tektoncd/experimental/blob/master/webhooks-extension/test/README.md#scripting)
