# Limitations
<br/>

- Only GitHub and Gitlab webhooks are currently supported.
- Status reporting on Gitlab merge requests does not fully function due to Gitlab architecture.
- Your git server host URL needs to contain github or gitlab in its name.
- Only `push` and `pull_request` events are currently supported for GitHub, these are the events defined on the webhook, tag creation shows as a push event and will also trigger pipelines.
- Only `push`, `tag push` and `merge request` events are currently supported for GitLab, these are the events defined on the webhook.
- The `TriggerTemplate` needs to be available in the install namespace with the name `<pipeline-name>-template` (details further below).
- The two `TriggerBindings` need to available in the install namespace with the names `<pipeline-name>-push-binding` and `<pipeline-name>-pullrequest-binding` (details further below).
- Limited configurable parameters are added to the trigger in the `EventListener` through the UI, statics could be added in your `TriggerBinding` (details further below).


## Tekton Triggers Information

#### Trigger Template & Trigger Bindings

As the UI does not currently offer the ability to select a `TriggerTemplate` or `TriggerBinding`, the current backend code expects to find trigger template and bindings with fixed names prefixed with the pipeline name.

- `<pipeline-name>-template`
- `<pipeline-name>-push-binding`
- `<pipeline-name>-pullrequest-binding`

The reason for requesting two bindings is due to the event payload being different.  The bindings would need to pull different keys from the event payload to run a pipeline for both pull requests and push events.

#### Event Listener Parameters

When a webhook is created through the dashboard UI, a number of parameters are made available to the `TriggerTemplate` through the `EventListener`.  The parameters added to the trigger in the `EventListener` are:

It is important to note the names of the parameters, should you wish to use the extension with your own `TriggerTemplates` and make use of these values.

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
