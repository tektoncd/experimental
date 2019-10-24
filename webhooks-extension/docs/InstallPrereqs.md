# Prerequisites

1. [Cluster requirements](#cluster-requirements)
2. [Install prereqs](#install-prereqs)
3. [Knative setup](#domain-setup-for-knative-serving)

## Cluster requirements

**All Kubernetes Clusters:**

- Knative requires a Kubernetes cluster running version v.1.11 or greater.
- Cluster must also be supplied with sufficient resources, for a single node cluster _(6 CPUs, 10GiB Memory & 2.5GiB swap)_.

**Docker Desktop Only:**

- Known to work with Kubernetes v1.11 and Kubernetes v1.14 (intermediate versions should work too, we just haven't tested it)

## Install prereqs

1. Install [Tekton Pipelines](https://github.com/tektoncd/pipeline/blob/master/docs/install.md) 

2. Install [Tekton Dashboard](https://github.com/tektoncd/dashboard)

3. Install Istio

   - **For a quickstart Istio install:**
 
       - Clone this repository: `git clone https://github.com/tektoncd/experimental.git`
       - Run the Istio installation script: `./scripts/install_istio.sh <version>`

         **Note:** requires [Helm](https://helm.sh/docs/using_helm/#installing-helm) 2.x to be installed and Istio `<version>` from 1.1.7 to 1.1.16 are known working configurations.

   - **For a custom quickstart install:**

       - Install as per instructions in the knative docs [https://knative.dev/docs/install/installing-istio/](https://knative.dev/docs/install/installing-istio/) _(Istio version 1.1.16 is recommended)_

   - **For official Istio install steps:**

       - Install as per instructions in the Istio documentation [https://istio.io/docs/setup/kubernetes/install/](https://istio.io/docs/setup/kubernetes/install/)

4. Install Knative Eventing, Eventing Sources & Serving

    The initial webhooks-extension implementation utilises Knative Eventing but there is discussion and work in [Tekton Pipelines](https://github.com/tektoncd/pipeline) to minimize necessary componentry in future.

   - **For a quickstart install:**
   
       - Clone this repository: `git clone https://github.com/tektoncd/experimental.git`
       - Run the Knative installation script: `./scripts/install_knative.sh <version>` 
         
         **Note:** only Knative `v0.6.0` is known to work and tested: you should specify this as `<version>`

   - **For official Knative install steps:**
   
       - Install as per instructions in the Knative documentation [https://knative.dev/v0.6-docs/](https://knative.dev/v0.6-docs/)

5.  Configure knative serving (see below)  


## Domain setup for Knative serving

After installing the prereqs, set your own domain and selectors following the [configuring Knative Serving docs](https://github.com/knative/serving/blob/master/install/CONFIG.md) which outlines setting up routes in the `config-domain` ConfigMap in the `knative-serving` namespace.

**Example using the cluster master node's DNS name**

  Patch the ConfigMap, set the dns_name environment variable substituting `CLUSTER_MASTER_NODE_DNS_NAME` with the relevant DNS name:

    export dns_name=CLUSTER_MASTER_NODE_DNS_NAME

  and then run the following command to patch the config map

    kubectl patch configmap config-domain --namespace knative-serving --type='json' \
      --patch '[{"op": "add", "path": "/data/'"${dns_name}"'", "value": ""}]'


**Example using IP**

  Retrieve your IP:

    ip=$(ifconfig | grep netmask | sed -n 2p | cut -d ' ' -f2)

  And patch it to the ConfigMap:

    kubectl patch configmap config-domain --namespace knative-serving --type='json' \
      --patch '[{"op": "add", "path": "/data/'"${ip}.nip.io"'", "value": ""}]'


This setup is important, as it will be used on the webhook's payload URL - remember that your source code repository must be able to reach your cluster, or your webhooks will never be received (mentioning "Service Timeout" errors).

If using GitHub enterprise you may simply be able to use your system IP address, if using github.com, your settings will need to be such that github.com can route to the endpoint through any firewall.