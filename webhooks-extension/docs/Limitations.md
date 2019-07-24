# Limitations
<br/>

- Only GitHub webhooks are currently supported.
- Only `push` and `pull_request` events are currently supported, these are the events defined on the webhook.
- A limited number of pipelines will currently work with this implementation.

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