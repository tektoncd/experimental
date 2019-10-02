# Limitations
<br/>

- Only GitHub webhooks are currently supported.
- Webhooks in GitHub are sometimes left behind after deletion (details further below).
- Only `push` and `pull_request` events are currently supported, these are the events defined on the webhook.
- A limited number of pipelines will currently work with this implementation.

## Webhook not deleted

Due to a bug in the Knative codebase, the deletion of the webhook does not alway occur in GitHub.  All artifacts required for the webhook to run a pipeline are successfully removed from the cluster but, the webhook in GitHub may need to be manually deleted.  Until the webhook is manually deleted, it will attempt to send event details to a non-existent service.  This problem will cease to exist once [issue 247](https://github.com/tektoncd/experimental/issues/247) is addressed and the webhooks extension moves away from its dependency on Knative.

## Deleted webhooks can still be rendered until a refresh occurs

The Webhooks Extension component does not currently work with all `Pipelines`, it very specifically creates the following when the webhook is triggered:
Due to a bug in *our* codebase, a scenario exists whereby deleted webhooks can appear on the webhooks display table. This scenario is known to occur under the following circumstance.

- Create a webhook named `-`
- Create a webhook named `--`
- Create a webhook with an appropriate name e.g. `mywebhook`
- Attempt to delete all three webhooks

An error is displayed mentioning that problems occurred deleting webhooks (the ones named - and -), but `mywebhook` has actually been deleted. It is only until you refresh the page that this webhook will no longer be displayed.

## Pipeline limitations

The Webhooks Extension component does not currently work with all Tekton Pipelines, it very specifically creates the following when the webhook is triggered:

#### Git PipelineResource

A PipelineResource of type `git` is created with:

  - `revision` set to the short commit ID from the webhook payload.
  - `url` set to the repository URL from the webhook payload.

#### Image PipelineResource

A PipelineResource of type `image` is created with:

  - `url` set to `${REGISTRY}/${REPOSITORY-NAME}:${SHORT-COMMITID}` where, `REGISTRY` is the value set when creating the webhook, other values are taken from the webhook payload.

#### PipelineRuns Parameters/Resources

For a PipelineRun for your chosen Pipeline, in the namespace specified when your webhook was created, the values assigned to parameters on the PipelineRun are taken from values provided when configuring the webhook or from the webhook payload itself.

It is important to note the names of the parameters and resources, should you wish to use the extension with your own Pipelines and make use of these values.

PipelineRun parameters and resources made available:

```
params:
- name: image-tag
  description: The short commit ID
- name: image-name
  description: Image name formatted as: <REGISTRY>/<REPOSITORY-NAME>
- name: repository-name
  description: Repository name formatted as: <REPOSITORY-NAME>
- name: release-name
  description: Repository name formatted as: <REPOSITORY-NAME>
- name: target-namespace
  description: The PipelineRun target namespace
- name: event-payload
  description: The JSON event payload/body
- name: event-headers
  description: The JSON headers
- name: branch
  description: The repository branch

resources:
- name: docker-image
  description: The Name of the ImageResource
- name: git-source
  description: The Name of the GitResource
```

In particular, the `event-headers` and `event-payload` parameters can be especially useful when creating `Conditions` for `Pipelines`. For example, [this](https://github.com/pipeline-hotel/example-pipelines/blob/master/config/pipeline.yaml) `Pipeline` uses [this](https://github.com/pipeline-hotel/example-pipelines/blob/master/config/deployment-condition.yaml) `Condition` to skip `Tasks` if the event type is a `pull_request`.
