# Trigger Parameters

The webhook extension makes a number of parameters automatically available for use in the triggertemplate file and the triggerbinding file(s).  

# TriggerBinding Parameters

These parameters are added to the webhook payload body by the webhooks extension interceptor code.  You can reference these parameters as you would any other parameter from the webhook payload in the trigger binding file(s), by prefixing the parameter name with `body.`.

`webhooks-tekton-git-branch` : this parameter is set to the final path segment of the ref tag in the the webhook payload  

`webhooks-tekton-image-tag` : this parameter is set to the shortened 7 character commit id, or, in the case of a git tag, to the tag name  

Example:

```
apiVersion: tekton.dev/v1alpha1
kind: TriggerBinding
metadata:
  name: simple-pipeline-push-binding
spec:
  params:
  - name: gitrevision
    value: $(body.head_commit.id)
  - name: gitrepositoryurl
    value: $(body.repository.clone_url)
  - name: docker-tag
    value: $(body.repository.name):$(body.webhooks-tekton-image-tag)
  - name: event-type
    value: $(header.X-Github-Event)
  - name: webhooks-tekton-git-branch
    value: $(body.webhooks-tekton-git-branch)
```


# TriggerTemplate Parameters

These parameters are, for the most part, settings that were configured when creating the webhook through the GUI.

Parameters Available:

```
  - webhooks-tekton-git-org
  - webhooks-tekton-git-repo
  - webhooks-tekton-git-server
  - webhooks-tekton-release-name
  - webhooks-tekton-target-namespace
  - webhooks-tekton-service-account
  - webhooks-tekton-docker-registry
```

To use these parameters in the triggertemplate, you simply prefix them with the parameter with `params.` (e.g `params.webhooks-tekton-git-org`).  See example triggertemplate below - note that additional params that are used and not listed above will be obtained from the triggerbinding file:

```
apiVersion: tekton.dev/v1alpha1
kind: TriggerTemplate
metadata:
  name: simple-pipeline-template
spec:
  resourcetemplates:
  - apiVersion: tekton.dev/v1alpha1
    kind: PipelineResource
    metadata:
      name: git-source-$(uid)
      namespace: $(params.webhooks-tekton-target-namespace)
    spec:
      type: git
      params:
      - name: revision
        value: $(params.gitrevision)
      - name: url
        value: $(params.gitrepositoryurl)
  - apiVersion: tekton.dev/v1alpha1
    kind: PipelineResource
    metadata:
      name: docker-image-$(uid)
      namespace: $(params.webhooks-tekton-target-namespace)
    spec:
      type: image
      params:
      - name: url
        value: $(params.webhooks-tekton-docker-registry)/$(params.docker-tag)
  - apiVersion: tekton.dev/v1alpha1
    kind: PipelineRun
    metadata:
      generateName: simple-pipeline-run-
      namespace: $(params.webhooks-tekton-target-namespace)
      labels:
        webhooks.tekton.dev/gitServer: $(params.webhooks-tekton-git-server)
        webhooks.tekton.dev/gitOrg: $(params.webhooks-tekton-git-org)
        webhooks.tekton.dev/gitRepo: $(params.webhooks-tekton-git-repo)
        webhooks.tekton.dev/gitBranch: $(params.webhooks-tekton-git-branch)
    spec:
      serviceAccount: $(params.webhooks-tekton-service-account)
      pipelineRef:
        name: simple-pipeline
      params:
      - name: event-type
        value: $(params.event-type)
      resources:
      - name: git-source
        resourceRef:
          name: git-source-$(uid)
      - name: docker-image
        resourceRef: 
          name: docker-image-$(uid)
```