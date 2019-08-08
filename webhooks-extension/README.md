# Webhooks Extension

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/kubernetes/experimental/blob/master/LICENSE)

The Webhooks Extension for Tekton allows users to set up GitHub webhooks that will trigger Tekton `PipelineRuns` and associated `TaskRuns`.

This initial implementation utilises Knative Eventing but there is discussion and work in [Tekton Pipelines](https://github.com/tektoncd/pipeline) to minimize necessary componentry in future.

In addition to Tekton/Knative Eventing glue, it includes an extension to the Tekton Dashboard.

See our [Getting Started](https://github.com/tektoncd/experimental/blob/master/webhooks-extension/docs/GettingStarted.md) guide for more on what this extension does, and how to use it. 


## Prerequisites

- Install [Tekton Pipelines](https://github.com/tektoncd/pipeline/blob/master/docs/install.md) and the [Tekton Dashboard](https://github.com/tektoncd/dashboard)

- Install Istio - for a quickstart install, for example of version 1.1.7 run `./scripts/install_istio.sh 1.1.7` _or_ follow https://knative.dev/docs/install/installing-istio/ for a more customised install _(Istio version 1.1.7 is recommended) - note that this script requires Helm to be installed_

- Install Knative Eventing, Eventing Sources & Serving - for a quickstart install, for example of version 0.6.0 run `./scripts/install_knative.sh v0.6.0`, for more detailed instructions see the [Knative docs](https://knative.dev/docs/install/index.html) _(Knative version 0.6.0 is strongly recommended)_

*If running on Docker for Desktop*

- Knative requires a Kubernetes cluster running version v.1.11 or greater. Currently this requires the edge version of Docker for Desktop. Your cluster must also be supplied with sufficient resources _(6 CPUs, 10GiB Memory & 2.5GiB swap is a known good configuration)_.


## Install Quickstart

### Domain setup for Knative Serving

Set your own domain and selectors following the [configuring Knative Serving docs](https://github.com/knative/serving/blob/master/install/CONFIG.md) which outlines setting up routes in the `config-domain` ConfigMap in the `knative-serving` namespace

On Docker Desktop, you can retrieve your IP and patch it to the ConfigMap by running:

`ip=$(ifconfig | grep netmask | sed -n 2p | cut -d ' ' -f2)`

```
kubectl patch configmap config-domain --namespace knative-serving --type='json' \
  --patch '[{"op": "add", "path": "/data/'"${ip}.nip.io"'", "value": ""}]'
```

### Install the Webhooks Extension 

### Installing the latest release

1. Run the
   [`kubectl apply`](https://kubernetes.io/docs/reference/generated/kubectl/kubectl-commands#apply)
   command to install the [Tekton Webhooks Extension](https://github.com/tektoncd/experimental/webhooks-extension)
   and its dependencies:
    
  ```bash
  kubectl apply --filename https://github.com/tektoncd/dashboard/releases/download/v0.1.0/webhooks-extension_release.yaml
  ```

   _(Previous versions will be available at `previous/$VERSION_NUMBER`, e.g.
   https://storage.googleapis.com/tekton-releases/previous/v0.1.0/webhooks-extension_release.yaml.)_

1. Run the
   [`kubectl get`](https://kubernetes.io/docs/reference/generated/kubectl/kubectl-commands#get)
   command to monitor the Tekton Dashboard Webhooks Extension component until all of the
   components show a `STATUS` of `Running`:

   ```bash
   kubectl get pods --namespace tekton-pipelines
   ```

   Tip: Instead of running the `kubectl get` command multiple times, you can
   append the `--watch` flag to view the component's status updates in real
   time. Use CTRL + C to exit watch mode.

You are now ready to use the Tekton Dashboard Webhooks Extension - see our [Getting Started](https://github.com/tektoncd/experimental/blob/master/webhooks-extension/docs/GettingStarted.md) guide.

## Nightly builds

The Tekton Webhooks Extension has a hosted image of the latest builds located at `gcr.io/tekton-nightly/extension:latest` and `gcr.io/tekton-nightly/extension:latest`, to install the latest extension and sink using these images:

```bash
kubectl apply -f config/release/gcr-tekton-webhooks-extension.yaml
```

### Access the Extension through the Dashboard UI 

Restart the dashboard to register the extension:

`kubectl delete pod -l app=tekton-dashboard`

Access the Dashboard through its ClusterIP Service by running `kubectl proxy`. Assuming tekton-pipelines is the install namespace for the dashboard, you can access the web UI at localhost:8001/api/v1/namespaces/tekton-pipelines/services/tekton-dashboard:http/proxy/ 

Navigate to `Webhooks`, listed in the navigation under `Extensions`


## Notes On Using The Webhooks Extension

Further to the limitations listed further below, it is worth noting that the webhooks extension does not currently work with all pipelines, it very specifically creates the following when the webhook is triggered:

#### Git PipelineResource

A PipelineResource of type git is created with:

  - `revision` set to the short commit id from the webhook payload.
  - `url` set to the repository url from the webhook payload.

#### Image PipelineResource

A PipelineResource of type image is created with:

  - `url` set to ${REGISTRY}/${REPOSITORY-NAME}:${SHORT-COMMITID} where, REGISTRY is the value set when creating the webhook, other values are taken from the webhook payload.

#### A PipelineRun

A PipelineRun for your chosen pipeline, in the namespace specified when your webhook was created, the values assigned to parameters on the pipelinerun are taken from values provided when configuring the webhook or from the webhook payload itself.

It is important to note the names of the parameters and resources, should you wish to use the extension with your own pipelines and make use of these values.

PipelineRun params and resources made available:

```
  params:
    - name: image-tag
      value: ${SHORT-COMMITID}
    - name: image-name
      value: ${REGISTRY}/${REPOSITORY-NAME}
    - name: release-name
      value: ${REPOSITORY-NAME}
    - name: repository-name
      value: ${REPOSITORY-NAME}
    - name: target-namespace
      value: ${PIPELINERUN-NAMESPACE}
    - name: docker-registry
      value: ${REGISTRY}

    resources:
    - name: docker-image
      resourceRef:
        name: foo-docker-image-1563812630
    - name: git-source
      resourceRef:
        name: bar-git-source-1563812630

    serviceAccount: ${SERVICE-ACCOUNT}
```


## Limitations

- Only GitHub webhooks are currently supported.
- Only `push` and `pull_request` events are currently supported, these are the events defined on the webhook.
- Only one webhook can be created for each Git repository, so each repository will only be able to trigger a `PipelineRun` from one webhook.


## Uninstall

`kubectl delete -f config/release/gcr-tekton-webhooks-extension.yaml`


## Install on OpenShift

Assuming you've installed knative and Istio already, configure your scc:

```
oc adm policy add-scc-to-user anyuid -z build-controller -n knative-build
oc adm policy add-scc-to-user anyuid -z controller -n knative-serving
oc adm policy add-scc-to-user anyuid -z autoscaler -n knative-serving
oc adm policy add-cluster-role-to-user cluster-admin -z build-controller -n knative-build
oc adm policy add-cluster-role-to-user cluster-admin -z controller -n knative-serving
```

Tip: if you plan to use `buildah` in your Pipelines, you will need to set an additional permission - for example with:

`oc adm policy add-scc-to-user privileged -z tekton-pipelines -n tekton-pipelines`

Install the extension:

`git clone https://github.com/tektoncd/experimental.git && kubectl apply -f experimental/webhooks-extension/config/release/gcr-tekton-webhooks-extension.yaml`

Check you can access the Webhooks Extension through the Dashboard UI that you should already have a Route for, for example at http://tekton-dashboard.${openshift_master_default_subdomain}/#/extensions/webhooks-extension.

Enable wildcard routes on your cluster:

```
oc scale -n default dc/router --replicas=0
oc set env -n default dc/router ROUTER_ALLOW_WILDCARD_ROUTES=true
oc scale -n default dc/router --replicas=1
```

Add a Route:

```
oc expose service istio-ingressgateway \
  -n istio-system \
  --name="webhooks-route" \
  --wildcard-policy="Subdomain" \
  --port="http2" \
  --hostname=wildcard.tekton-pipelines.${openshift_master_default_subdomain}
```

`$openshift_master_default_subdomain` in this example is `mycluster.foo.com`. This gives you the following Route:

```
NAME                                    HOST/PORT                                                         PATH      SERVICES               PORT      TERMINATION          WILDCARD
webhooks-route                          wildcard.tekton-pipelines.mycluster.foo.com                       istio-ingressgateway             http2                          Subdomain
```

Configure your ConfigMap `config-domain` in the `knative-serving` namespace: a working set up can be achieved with:

```
kubectl patch configmap config-domain --namespace knative-serving --type='json' \
  --patch '[{"op": "add", "path": "/data/'"${openshift_master_default_subdomain}"'", "value": ""}]'
```

You can now proceed to create webhooks using the Webhooks Extension UI. Remember that your source code repository must be able to reach your cluster, or your webhooks will never be received (mentioning "Service Timeout" errors.

This has been tested with the following scc (from `oc get scc`):

```
NAME               PRIV      CAPS      SELINUX     RUNASUSER          FSGROUP     SUPGROUP    PRIORITY   READONLYROOTFS   VOLUMES
anyuid             false     []        MustRunAs   RunAsAny           RunAsAny    RunAsAny    10         false            [configMap downwardAPI emptyDir persistentVolumeClaim projected secret]
hostaccess         false     []        MustRunAs   MustRunAsRange     MustRunAs   RunAsAny    <none>     false            [configMap downwardAPI emptyDir hostPath persistentVolumeClaim projected secret]
hostmount-anyuid   false     []        MustRunAs   RunAsAny           RunAsAny    RunAsAny    <none>     false            [configMap downwardAPI emptyDir hostPath nfs persistentVolumeClaim projected secret]
hostnetwork        false     []        MustRunAs   MustRunAsRange     MustRunAs   MustRunAs   <none>     false            [configMap downwardAPI emptyDir persistentVolumeClaim projected secret]
node-exporter      false     []        RunAsAny    RunAsAny           RunAsAny    RunAsAny    <none>     false            [*]
nonroot            false     []        MustRunAs   MustRunAsNonRoot   RunAsAny    RunAsAny    <none>     false            [configMap downwardAPI emptyDir persistentVolumeClaim projected secret]
privileged         true      [*]       RunAsAny    RunAsAny           RunAsAny    RunAsAny    <none>     false            [*]
restricted         false     []        MustRunAs   MustRunAsRange     MustRunAs   RunAsAny    <none>     false            [configMap downwardAPI emptyDir persistentVolumeClaim projected secret]
```


## Want to get involved?

Visit the [Tekton Community](https://github.com/tektoncd/community) project for an overview of our processes.


## For developers

If you are looking to develop or contribute to this repository please see the [development docs](https://github.com/tektoncd/experimental/blob/master/webhooks-extension/DEVELOPMENT.md)

For more involved development scripts please see the [development installation guide](https://github.com/tektoncd/experimental/blob/master/webhooks-extension/test/README.md#scripting)
