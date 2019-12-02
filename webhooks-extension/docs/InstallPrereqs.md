# Prerequisites

1. [Cluster requirements](#cluster-requirements)
2. [Install prereqs](#install-prereqs)

## Cluster requirements

**All Kubernetes Clusters:**

- Knative requires a Kubernetes cluster running version v.1.11 or greater.
- Cluster must also be supplied with sufficient resources, for a single node cluster _(6 CPUs, 10GiB Memory & 2.5GiB swap)_.

**Docker Desktop Only:**

- Known to work with Kubernetes v1.11 and Kubernetes v1.14 (intermediate versions should work too, we just haven't tested it)

## Install prereqs

1. Install [Tekton Pipelines](https://github.com/tektoncd/pipeline/blob/master/docs/install.md) version 0.7  

2. Install [Tekton Dashboard](https://github.com/tektoncd/dashboard)

    Creation of the first webhook might exceed 30s while pods start, therefore you should increase your gateway timeout.  

    _On RedHat OpenShift:_ 

    Increase the gateway timeout on the tekton-dashboard route using the following command:

        ```
        oc annotate route tekton-dashboard --overwrite haproxy.router.openshift.io/timeout=2m
        ```

3. Install [Tekton Triggers](https://github.com/tektoncd/triggers/blob/master/docs/install.md#installing-tekton-triggers-1) version 0.1  

4. Install a LoadBalancer if one is not present on your cluster.  For Docker Desktop you could consider using nginx as per the following instructions:

```bash
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/master/deploy/static/mandatory.yaml
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/master/deploy/static/provider/cloud-generic.yaml
```
