# Uninstall
<br/>

To uninstall to webhooks extension:
<br/>

1. Clone this repository

    ```bash
    git clone https://github.com/tektoncd/experimental.git
    ```

2. Use the `kubectl delete` command to delete the webhooks extension

    ```bash
    kubectl delete -f config/latest/gcr-tekton-webhooks-extension.yaml
    ```
<br/>

Uninstall any of the prereqs added during installation:

1. [Uninstall Tekton Dashboard](https://github.com/tektoncd/dashboard)  
2. [Unintsall Knative](https://knative.dev/docs/)  
3. [Uninstall Istio](https://istio.io/docs/setup/kubernetes/getting-started/)  