# Installing Official Releases
<br/>

Run the [`kubectl apply`](https://kubernetes.io/docs/reference/generated/kubectl/kubectl-commands#apply) command to perform installation.  By default the installation will be into a namespace called "tekton-pipelines".  

If the Tekton Dashboard has been installed into a namespace other than "tekton-pipelines", then you will need to ensure you install the Webhooks Extension component into the same namespace.  Instructions for installing into an alternative namespace are provided below the standard install instructions.
<br/>
<br/>

  * **To install latest release image:**

    _On Red Hat OpenShift:_

    Use the following command, replacing `YOUR_SUBDOMAIN_VALUE` with your configured openshift master default subdomain.  

    ```bash
    curl -L https://github.com/tektoncd/dashboard/releases/latest/download/openshift-tekton-webhooks-extension-release.yaml \
    | sed -e 's/{openshift_master_default_subdomain}/YOUR_SUBDOMAIN_VALUE/g' \
    | oc apply --filename -
    ```

    Example:
    ```bash
    curl -L https://github.com/tektoncd/dashboard/releases/latest/download/openshift-tekton-webhooks-extension-release.yaml \
    | sed -e 's/{openshift_master_default_subdomain}/apps.myhost.mycompany.net/g' \
    | oc apply -f -
    ```


    _On other Kubernetes environments:_

    Use the following command, replacing `YOUR_IP_ADDRESS` with your docker-desktop system's IP address.  Note that in
    production you should clone the repository and update the WEBHOOK_CALLBACK_URL setting to avoid using nip.io for host resolution.

    ```bash
    curl -L https://github.com/tektoncd/dashboard/releases/latest/download/webhooks-extension_release.yaml \
    | sed -e 's/IPADDRESS/YOUR_IP_ADDRESS/g' \
    | kubectl apply -f - 
    ```  
<br/>

  * **To install latest release image into a specific namespace:**

    Use the following command to install into an alternative namespace, replacing `TARGET_NAMESPACE` with the required namespace.

    _On Red Hat OpenShift:_
    
    Use the following command, replacing `YOUR_SUBDOMAIN_VALUE` with your configured openshift master default subdomain.  

    ```bash
    curl -L https://github.com/tektoncd/dashboard/releases/latest/download/openshift-tekton-webhooks-extension-release.yaml \
    | sed 's/tekton-pipelines/TARGET_NAMESPACE/g' \
    | sed -e 's/{openshift_master_default_subdomain}/YOUR_SUBDOMAIN_VALUE/g' \
    | oc apply -f -
    ```

    _On other Kubernetes environments:_

    Use the following command, replacing `YOUR_IP_ADDRESS` with your docker-desktop system's IP address.  Note that in
    production you should clone the repository and update the WEBHOOK_CALLBACK_URL setting to avoid using nip.io for host resolution.

    ```bash
    curl -L https://github.com/tektoncd/dashboard/releases/latest/download/webhooks-extension_release.yaml \
    | sed 's/tekton-pipelines/TARGET_NAMESPACE/g' \
    | sed -e 's/IPADDRESS/YOUR_IP_ADDRESS/g' \
    | kubectl apply -f -
    ```  
<br/>

  * **To install a specific version:**

    If looking to install an official release of v0.2.1 or earlier, please navigate to the docs for that release by selecting the branch name matching the version you want to install in the dropdown at the top of this page.  Different prereqs and actions are required for installations of v0.2.1 and earlier.  
    
    In general, once you have taken all the prereq actions for the release level you want to install, you need to use a similar URL as in the previous commands, but that specifies the version you want to install for example, for version 0.2.0:

    _On Red Hat OpenShift:_

    `https://github.com/tektoncd/dashboard/releases/previous/v0.2.0/openshift-webhooks-extension.yaml`

    Note: As of v0.2.1 the filename became openshift-tekton-webhooks-extension-release.yaml

    _On other Kubernetes environments:_

    `https://github.com/tektoncd/dashboard/releases/previous/v0.2.0/webhooks-extension_release.yaml`

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

