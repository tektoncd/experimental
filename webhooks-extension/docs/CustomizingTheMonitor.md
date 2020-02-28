# Customizing the Monitor Task

## Contents

1. [Changing The Polling Duration](#changing-the-polling-duration)
2. [Overriding The Status Message](#overriding-the-status-message)
3. [Custom Monitor Tasks](#custom-monitor-tasks)

## Introduction

If the webhook is triggered due to a pull request being created (or updated with code), a monitor task will be run to track and report on the status of the configured `PipelineRun` that is started by the webhooks-extension.  Some customization of the default monitor task is possible along with overriding the task altogether.


## Changing The Polling Duration

The polling duration of the `monitor-result-task` Task that comes installed with the webhooks-extension, polls the `PipelineRun` every 10 seconds for up to 30 minutes (1800 seconds).

To extend the duration allowed for `PipelineRun` completion, you would currently need to edit the Task, either using `kubectl edit task monitor-result-task -n tekton-pipelines` or by editing a copy of the yaml and reapplying using `kubectl apply -f <path_to_yaml_file> -n tekton-pipelines`.

If you have not installed into the tekton-pipelines namespace, you would need to change the value in the command you use to edit/apply the yaml.


## Overriding The Status Message

To change the "Success", "Failed", "Unknown" and "Missing" status messages shown in the 'Tekton Status' comment added to the pull request, you can use the REST endpoint for creating your webhooks, rather than the UI.  Note however that the REST endpoints are only advised to be used in development, see [here](../DevelopmentAPIs.md).

The body you POST to the REST endpoint has four properties, `onsuccesscomment`, `onfailurecomment`, `ontimeoutcomment`, `onmissingcomment` that define the value used in the Status column.  For example, creating the webhook using the JSON body:

```
{
  "name": "germanmessage",
  "namespace": "tekton-pipelines",
  "gitrepositoryurl": "https://github.com/ORG/REPO",
  "accesstoken": "GITHUBSECRET",
  "pipeline": "simple-pipeline",
  "dockerregistry": "FOO",
  "onsuccesscomment": "Erfolg",  <--------------------------------------------
  "onfailurecomment": "Fehler",  <--------------------------------------------
  "ontimeoutcomment": "Frozen",  <--------------------------------------------
  "onmissingcomment": "Fehlt"    <--------------------------------------------
}
```

If the `PipelineRun` then fails we get "Fehler" as the status.

![German failure comment](./images/germanComment.png?raw=true "German failure comment on GitHub pull request")


## Custom Monitor Tasks

Using the REST endpoint directly, it is also possible to override the `Task` that is created after the `PipelineRun`.  The `Task` must be specified on the pulltask property in the JSON body, for example:

```
{
  "name": "germanmessage",
  "namespace": "tekton-pipelines",
  "gitrepositoryurl": "https://github.com/ORG/REPO",
  "accesstoken": "GITHUBSECRET",
  "pipeline": "simple-pipeline",
  "dockerregistry": "FOO",
  "onsuccesscomment": "Erfolg",
  "onfailurecomment": "Fehler",
  "ontimeoutcomment": "Frozen",
  "onmissingcomment": "Fehlt",
  "pulltask": "my-custom-task"   <--------------------------------------------
}
```

In this situation, after the webhook is triggered and the `PipelineRun` created, a `TaskRun` for the `my-custom-task` will be created instead of the default `monitor-result-task` `Task`.  The same set of inputs, output and parameters will be placed on the `TaskRun` as would have been placed into the `TaskRun` for the  `monitor-result-task` `Task`.

```
  inputs:
    params:
    - name: commentsuccess
      value: Erfolg
    - name: commentfailure
      value: Fehler
    - name: commenttimeout
      value: Frozen
    - name: commentmissing
      value: Fehlt
    - name: secret
      value: GITHUBSECRET
```

The params above are populated from the data specified at webhook creation.

```
    - name: pipelineruns
      value: germanmessage-1565788339-rjmzv:tekton-pipelines:simple-pipeline
```

The `pipelineruns` parameter is currently a comma separated list of pipelinerunname:namespace:pipeline, so where two `PipelineRuns` were created due to the webhook triggering, this might be the string "germanmessage-1565788339-rjmzv:tekton-pipelines:simple-pipeline,englishmessage-1414141452-uwntd:default:other-pipeline".

``` 
    name: dashboard-url
      value: http://localhost:9097/
```

The `dashboard-url` defaults to `http://localhost:9097/` unless a value can be found from a call to the `endpoints` REST endpoint.

```
    resources:
    - name: pull-request
      resourceRef:
        name: pull-request-n2dfs
  outputs:
    resources:
    - name: pull-request
      resourceRef:
        name: pull-request-n2dfs
```

A `PipelineResource` of type pullRequest is created and added to the `TaskRun` as both an input and output resource.