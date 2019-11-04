# Limitations
<br/>

- Only GitHub webhooks are currently supported.
- Webhooks in GitHub are sometimes left behind after deletion (details further below).
- Only `push` and `pull_request` events are currently supported, these are the events defined on the webhook.
- The trigger template needs to be available in the install namespace with the name `<pipeline-name>-template` (details further below).
- The two trigger bindings need to available in the install namespace with the names `<pipeline-name>-push-binding` and `<pipeline-name>-pullrequest-binding` (details further below).
- Limited configurable parameters are added to the trigger in the eventlistener through the UI, statics could be added in your trigger binding (details further below).

## Deleted webhooks can still be rendered until a refresh occurs

The Webhooks Extension component does not currently work with all `Pipelines`, it very specifically creates the following when the webhook is triggered:
Due to a bug in *our* codebase, a scenario exists whereby deleted webhooks can appear on the webhooks display table. This scenario is known to occur under the following circumstance.

- Create a webhook named `-`
- Create a webhook named `--`
- Create a webhook with an appropriate name e.g. `mywebhook`
- Attempt to delete all three webhooks

An error is displayed mentioning that problems occurred deleting webhooks (the ones named - and -), but `mywebhook` has actually been deleted. It is only until you refresh the page that this webhook will no longer be displayed.

## Tekton Triggers Information

#### Trigger Template & Trigger Bindings

As the UI does not currently offer the ability to select a trigger template or trigger binding, the current backend code expects to find trigger template and binding with fixed names prefixed with the pipeline name.

- `<pipeline-name>-template`
- `<pipeline-name>-push-binding`
- `<pipeline-name>-pullrequest-binding`

The reason for requesting two bindings is due to the event payload being different.  The bindings would need to pull different keys from the event payload to run a pipeline for both pull requests and push events.

#### Event Listener Parameters

When a webhook is created through the dashboard UI, a number of parameters are made available to the trigger template through the event listener.  The parameters added to the trigger in the event listener are:

It is important to note the names of the parameters, should you wish to use the extension with your own trigger templates and make use of these values.

```
params:
- name: release-name
  description: The git repository name
- name: target-namespace
  description: A namespace, generally referenced in metadata sections to define the namespace in which to create a resource
- name: service-account
  description: A service account name, generally referenced to ensure PipelineRuns are executed under a specific service account
- name: docker-registry
  description: A docker registry, generally referenced where systems push images to a configured registry 
```

The event headers and event-payload parameters can be especially useful when creating `Conditions` for `Pipelines`. For example, [this](https://github.com/pipeline-hotel/example-pipelines/blob/master/triggers-resources/config/simple-pipeline/simple-pipeline.yaml) `Pipeline` uses [this](https://github.com/pipeline-hotel/example-pipelines/blob/master/triggers-resources/config/simple-pipeline/deployment-condition.yaml) `Condition` to skip `Tasks` if the event type is a `pull_request`.  You can see how the relevant property is passed from the event via the bindings files [here](https://github.com/pipeline-hotel/example-pipelines/blob/master/triggers-resources/config/simple-pipeline/simple-pipeline-push-binding.yaml) for push events and [here](https://github.com/pipeline-hotel/example-pipelines/blob/master/triggers-resources/config/simple-pipeline/simple-pipeline-pullrequest-binding.yaml) for pull requests.
