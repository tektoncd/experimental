# Notes for Red Hat OpenShift Installations

### Using buildah in your pipelines

If you plan to use `buildah` in your Pipelines, you will need to set an additional permission on any service account that will be used to run a pipeline by using the following command:

      ```
      oc adm policy add-scc-to-user privileged -z [service_account_name] -n [namespace]
      ```


### Pushing to the OpenShift registry using webhooks

Let's assume you wish to create a webhook such that created PipelineRuns will use the provided service account `tekton-webhooks-extension`.

Run the following command first:

`oc adm policy add-role-to-user edit -z tekton-webhooks-extension`

You should specify the following registry location if your namespace is `kabanero`:

`image-registry.openshift-image-registry.svc:5000/kabanero` (for OpenShift 4.2x)

or

`docker-registry.default.svc:5000/kabanero` (for OpehShift 3.11)

If using a self-signed certificate for the internal RedHat Docker registry, you will need to use a `buildah` task that skips self-signed certificate verifications too, for example by using the Tekton catalog's `buildah` task and setting TLS_VERIFY to default to `false`


### Defined SCC

This project has been tested with the following scc (from `oc get scc`):

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