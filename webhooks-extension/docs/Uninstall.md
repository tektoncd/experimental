# Uninstall
<br/>

To uninstall the webhooks extension:
<br/>

1. Clone this repository

    ```bash
    git clone https://github.com/tektoncd/experimental.git
    ```

2. Use the `kubectl delete` command to delete the webhooks extension
      
      _On Red Hat OpenShift:_

      ```bash
      kubectl delete -k overlays/openshift-latest
      ```

      _On other Kubernetes environments:_

      ```bash
      kubectl delete -k overlays/latest
      ```  
<br/>

Uninstall any of the prereqs added during installation:

1. [Uninstall Tekton Dashboard](https://github.com/tektoncd/dashboard)  
2. Uninstall Tekton Triggers

    ```bash
    kubectl delete --filename https://storage.googleapis.com/tekton-releases/triggers/latest/release.yaml
    ```

3. Uninstall Tekton Pipelines

    ```bash
    kubectl delete -f https://storage.googleapis.com/tekton-releases/pipeline/latest/release.yaml
    ```

Note: You may need to use the URL of the file you installed rather than the latest release in the above commands.