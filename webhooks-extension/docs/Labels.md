# Labelling PipelineRuns

The webhooks extension's graphical user interface (GUI) provides the ability to display the status of the latest `PipelineRun` for branches of a specified webhook.  This listing is seen by clicking on the webhook name in the webhook table.

For this functionality to work, users must:  
<br/>

1. Add labels to the pipelinerun's metadata defined in the pipeline's triggertemplate.  The labels required are:

```
  webhooks.tekton.dev/gitBranch: $(params.webhooks-tekton-git-branch)
  webhooks.tekton.dev/gitOrg: $(params.webhooks-tekton-git-org)
  webhooks.tekton.dev/gitRepo: $(params.webhooks-tekton-git-repo)
  webhooks.tekton.dev/gitServer: $(params.webhooks-tekton-git-server)
```  
<br/>

Example excerpt:

```
  apiVersion: tekton.dev/v1alpha1
  kind: PipelineRun
  metadata:
    labels:
      webhooks.tekton.dev/gitBranch: $(params.webhooks-tekton-git-branch)
      webhooks.tekton.dev/gitOrg: $(params.webhooks-tekton-git-org)
      webhooks.tekton.dev/gitRepo: $(params.webhooks-tekton-git-repo)
      webhooks.tekton.dev/gitServer: $(params.webhooks-tekton-git-server)
    generateName: simple-pipeline-run-
  spec:
    :
    :
```  
<br/>
<br/>


2. Three of the `params` are automatically available to the `TriggerTemplate` at runtime, however `params.webhooks-tekton-git-server` needs to be extracted from the triggering event's payload. In the pipeline's `TriggerBinding` files (both push and pullrequest) add the following entry to the params to make webhooks-tekton-git-server available as a param to the `TriggerTemplate`:

```
  - name: webhooks-tekton-git-branch
    value: $(body.webhooks-tekton-git-branch)
```  
<br/>

Example excerpt:

```
apiVersion: tekton.dev/v1alpha1
kind: TriggerBinding
metadata:
  name: simple-pipeline-push-binding
spec:
  params:
  - name: webhooks-tekton-git-branch
    value: $(body.webhooks-tekton-git-branch)
```  
<br/>

Once these steps have been taken, new `Pipelineruns` will be configured correctly such that clicking on a webhook will render a listing similar to the image below:

![Latest pipelinerun status for a webhook, displayed by branch with clickable link](./images/webhookBranches.png?raw=true "Latest pipelinerun status for a webhook, displayed by branch with clickable link")

Clicking on the branch name will navigate to a filtered list of `PipelineRuns` for this pipeline running against the specific branch of the repository.