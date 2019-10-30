# Installing on Red Hat OpenShift

There are two procedures for installing the Tekton Dashboard and Webhooks Extension on OpenShift. The recommended approach is to use the operators provided as part of OpenShift 4.2 - but a couple of command-line based steps are currently required. Also note that this uses a (as of the 24th of October) unreleased version of the Serverless Operator (version 1.1.0). If it's in the operator catalog by the time you're using OpenShift 4.2, go ahead and skip the step involving cloning it from source.

## Install on Red Hat OpenShift 4.2

1. Install the ServiceMesh 1.0.1 operator
2. Configure the ServiceMesh using the example configuration files in the provided `example-openshift-configuration` folder:

```
oc new-project istio-system
oc apply -f example-openshift-configuration/maistra.yaml
oc apply -f example-openshift-configuration/member-roll.yaml
```   

3. Install the Tekton Pipelines operator into a namespace other than where you will install the Tekton Dashboard and Webhooks Extension (e.g. a namespace called `kabanero`)
4. Install the Knative Eventing 0.8 operator
5. Install the Knative Eventing Contrib GitHubSource CRD:

`oc apply -f https://github.com/knative/eventing-contrib/releases/download/v0.8.0/github.yaml`

6. Install the Serverless 1.1.0 Operator

7. Install the Knative Serving object:

`oc apply -f example-openshift-configuration/Serving.yaml`

8. Install the Tekton Dashboard into a namespace other than `openshift-pipelines`, for example: `kabanero`:

```
curl -L https://github.com/tektoncd/dashboard/releases/download/v0.2.1/openshift-tekton-dashboard.yaml \
  | sed 's/namespace: tekton-pipelines/namespace: kabanero/' \
  | sed 's/value: tekton-pipelines/value: kabanero/' \
  | oc apply --validate=false --filename -
```

```
curl -L https://github.com/tektoncd/dashboard/releases/download/v0.2.1/openshift-webhooks-extension.yaml \
  | sed 's/namespace: tekton-pipelines/namespace: kabanero/' \
  | sed 's/default: tekton-pipelines/default: kabanero/' \
  | oc apply --filename -
```

### Pushing to the OpenShift registry using webhooks

Let's assume you wish to create a webhook such that created PipelineRuns will use the provided service account `tekton-webhooks-extension`.

Run the following command first:

`oc adm policy add-role-to-user edit -z tekton-webhooks-extension`

You should specify the following registry location if your namespace is `kabanero`:

`image-registry.openshift-image-registry.svc:5000/kabanero`

If using a self-signed certificate for the internal RedHat Docker registry, you will need to use a `buildah` task that skips self-signed certificate verifications too, for example by using the Tekton catalog's `buildah` task and setting TLS_VERIFY to default to `false`

## Install on Red Hat OpenShift 3.11

Assuming you've completed the [prereq installation and setup](./InstallPrereqs.md),

1. Configure your `scc`:

      ```
      oc adm policy add-scc-to-user anyuid -z build-controller -n knative-build
      oc adm policy add-scc-to-user anyuid -z controller -n knative-serving
      oc adm policy add-scc-to-user anyuid -z autoscaler -n knative-serving
      oc adm policy add-cluster-role-to-user cluster-admin -z build-controller -n knative-build
      oc adm policy add-cluster-role-to-user cluster-admin -z controller -n knative-serving
      ```

2. If you plan to use `buildah` in your Pipelines, you will need to set an additional permission on any service account that will be used to run a pipeline by using the following command:

      ```
      oc adm policy add-scc-to-user privileged -z [service_account_name] -n [namespace]
      ```

3. Enable wildcard routes on your cluster:

      ```
      oc scale -n default dc/router --replicas=0
      oc set env -n default dc/router ROUTER_ALLOW_WILDCARD_ROUTES=true
      oc scale -n default dc/router --replicas=1
      ```

4. Define a Route for the webhooks:

      ```
      oc expose service istio-ingressgateway \
        -n istio-system \
        --name="webhooks-route" \
        --wildcard-policy="Subdomain" \
        --port="http2" \
        --hostname=wildcard.tekton-pipelines.${openshift_master_default_subdomain}
      ```

    **Example:**

    In this example, we can see a Route that was created with `$openshift_master_default_subdomain` set to `mycluster.foo.com`.

    ```
    oc expose service istio-ingressgateway \
      -n istio-system \
      --name="webhooks-route" \
      --wildcard-policy="Subdomain" \
      --port="http2" \
      --hostname=wildcard.tekton-pipelines.mycluster.foo.com
    ```
    
    We can get the route by running `oc get routes -n istio-system`:

    ```
    NAME                                    HOST/PORT                                                         PATH      SERVICES               PORT      TERMINATION          WILDCARD
    webhooks-route                          wildcard.tekton-pipelines.mycluster.foo.com                       istio-ingressgateway             http2                          Subdomain
    ```

5. Install the webhooks extension:

      - Install the [release build](./InstallReleaseBuild.md)
      - Install the [nightly build](./InstallNightlyBuild.md)

6. Check you can access the Webhooks Extension through the Dashboard UI that you should already have a Route for, for example at http://tekton-dashboard.${openshift_master_default_subdomain}/#/extensions/webhooks-extension.

    ![Create webhook page in dashboard](./images/createWebhook.png?raw=true "Create webhook page in dashboard")

7. Begin creating webhooks


## Notes:

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