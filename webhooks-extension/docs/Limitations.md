# Limitations
<br/>

- Only GitHub webhooks are currently supported.
- Webhooks in GitHub are sometimes left behind after deletion (details further below).
- Only `push` and `pull_request` events are currently supported, these are the events defined on the webhook.
- A limited number of pipelines will currently work with this implementation.

## Webhook Not Deleted

Due to a bug in the knative codebase, the deletion of the webhook does not alway occur in GitHub.  All artefacts required for the webhook to run a pipeline are successfully removed from the cluster but the webhook in GitHub may need to be manually deleted.  Until the webhook is manually deleted, it will attempt to send event details to a non-existent service.  This problem will cease to exist once [issue 247](https://github.com/tektoncd/experimental/issues/247) is addressed and the webhooks extension moves away from its dependency on knative.

## Pipeline Limitations

The Webhooks Extension component does not currently work with all pipelines, it very specifically creates the following when the webhook is triggered:

#### Git PipelineResource

A PipelineResource of type `git` is created with:

  - `revision` set to the short commit id from the webhook payload.
  - `url` set to the repository url from the webhook payload.

#### Image PipelineResource

A PipelineResource of type `image` is created with:

  - `url` set to `${REGISTRY}/${REPOSITORY-NAME}:${SHORT-COMMITID}` where, `REGISTRY` is the value set when creating the webhook, other values are taken from the webhook payload.

#### A PipelineRun

A PipelineRun for your chosen pipeline, in the namespace specified when your webhook was created, the values assigned to parameters on the pipelinerun are taken from values provided when configuring the webhook or from the webhook payload itself.

It is important to note the names of the parameters and resources, should you wish to use the extension with your own pipelines and make use of these values.

PipelineRun params and resources made available:

```
  params:
    - name: image-tag
      value: ${SHORT-COMMITID}
    - name: image-name
      value: ${REGISTRY}/${REPOSITORY-NAME}
    - name: release-name
      value: ${REPOSITORY-NAME}
    - name: repository-name
      value: ${REPOSITORY-NAME}
    - name: target-namespace
      value: ${PIPELINERUN-NAMESPACE}
    - name: docker-registry
      value: ${REGISTRY}

    resources:
    - name: docker-image
      resourceRef:
        name: foo-docker-image-1563812630
    - name: git-source
      resourceRef:
        name: bar-git-source-1563812630

    serviceAccount: ${SERVICE-ACCOUNT}
```