# Nightly builds
<br/>
<br/>

The Tekton Webhooks Extension has a hosted image of the latest builds located at gcr.io/tekton-nightly/extension.  

Note: If you want to install into an alternative namespace you would need to modify all namespace entries in the applied yaml files.
<br/>
<br/>

* **To install the latest nightly image:**

  1. Clone this repository

      ```bash
      git clone https://github.com/tektoncd/experimental.git
      ```
  
  2. Change into the webhooks-extension directory

      ```bash
      cd webhooks-extension
      ```

  3. Set the WEBHOOK_CALLBACK_URL in the relevant yaml file

      _On Red Hat OpenShift:_

      - Open overlays/openshift-all/deployment-patch.yaml.
      - Find WEBHOOK_CALLBACK_URL.
      - Edit the value - this should simply be a case of replacing {openshift_master_default_subdomain} with your actual value.

      _On other Kubernetes environments:_

      - Open overlays/plainkube-all/deployment-patch.yaml.
      - Find WEBHOOK_CALLBACK_URL.
      - Edit the value - this could simply be a case of replacing IPADDRESS with your actual value.  

  3. Apply the yaml

      _On Red Hat OpenShift:_

      ```bash
      kubectl apply -k overlays/openshift-latest
      ```

      _On other Kubernetes environments:_

      ```bash
      kubectl apply -k overlays/latest
      ```  
<br/>
<br/>

You will be able to access the webhooks section of the dashboard once the pods are all up and running.
<br/>
<br/>

  * **To monitor the pods:**
  
    Run the [`kubectl get`](https://kubernetes.io/docs/reference/generated/kubectl/kubectl-commands#get) pods command to monitor the Tekton Dashboard Webhooks Extension component until all of the components show a `STATUS` of `Running`:

    ```bash
    kubectl get pods --namespace tekton-pipelines --watch
    ```
    _Tip: Use CTRL + C to exit watch mode._
<br/>

  * **To access the webhooks extension:**

    Access the Webhooks Extension through the Dashboard UI that you should already have a Route for, for example at:
    
    - http://tekton-dashboard.[cluster_master_node_DNS_name]/#/extensions/webhooks-extension

    _or if using the kube proxy_

    - http://localhost:8001/api/v1/namespaces/tekton-pipelines/services/tekton-dashboard:http/proxy/#/extensions/webhooks-extension
<br/>

You are now ready to use the Tekton Dashboard Webhooks Extension - see our [Getting Started](https://github.com/tektoncd/experimental/blob/master/webhooks-extension/docs/GettingStarted.md) guide.

  ![Create webhook page in dashboard](./images/createWebhook.png?raw=true "Create webhook page in dashboard")