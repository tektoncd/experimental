# Install on Red Hat OpenShift

Assuming you've completed the [prereq installation and setup](./InstallPrereqs.md),

1. Configure your scc:

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