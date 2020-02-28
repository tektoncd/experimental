# Architecture Information

## Contents

1. [Initial Setup Overview](#initial-setup-overview)
2. [Webhook Creation Architecture](#webhook-creation-architecture)
3. [Webhook Runtime Architecture](#webhook-runtime-architecture)



## Initial Setup Overview

![User Setup Diagram](./images/setup.png?raw=true "Diagram showing initial user setup")

The diagram above shows the initial configuration required prior to using the webhooks extension.  The process consists of:

- Installing the necessary parts of Tekton

- Installing the `TriggerTemplate` for your `Pipeline` into the same namespace as the Tekton installation. **Your `TriggerTemplate` must currently be named `<pipeline-name>-template`**.

- Installing the `TriggerBindings` for your pipeline into the same namespace as the Tekton installation. **You need to install two `TriggerBindings` per `Pipeline`**, one for pull request events and one for push events.  It is believed that this is likely to be changed in a future release. **Your `TriggerBindings` must currently be named `<pipeline-name>-push-binding` and `<pipeline-name>-pullrequest-binding`**.

- Creating secrets for accessing GitHub/Gitlab and Docker and patching them onto the service account under which you want your `PipelineRun` to execute.  These secrets need creating in the **target namespace** where you want your `Pipeline` to run.

- Installing the `Pipeline` into the **target namespace** where you want the `Pipeline` to run.

Note: On installation of the webhooks extension a number of `Tasks` and `Deployments` are installed that are used by the webhooks extension - these are omitted from the diagram above as they are installed automatically as a consequence of installing the webhooks extension.
<br/>
<br/>

## Webhook Creation Architecture

![Webhook Creation Architecture Diagram](./images/creation-architecture.png?raw=true "Diagram showing webhook creation architecture of the webhooks extension")

The diagram above shows the events that take place when webhooks are created via the UI. The process consists of:

1) Creating or updating the `EventListener` from a dashboard webhook request.
For each webhook, two triggers are created. These triggers are for push and
pull request webhook payloads from GitHub/Gitlab, where each payload is structured
differently (highly likely you will need different bindings for the payload). A
third trigger is created for webhooks on a distinct GitHub/Gitlab repository (no such
webhook exists yet). This trigger is for the monitor taskrun that is created
when a pull request event occurs on the repository.  For each trigger, an extra
`TriggerBinding` is created to contain settings used during webhook creation.

2) Creation of a ingress/route which exposes the `EventListener` to the world outside of the cluster.

3) Creation of the actual webhook in GitHub/Gitlab (if one does not already exist).

<br/>
<br/>

## Webhook Runtime Architecture
<br/>

![Architecture Diagram](./images/architecture.png?raw=true "Diagram showing overall runtime architecture of the webhooks extension")

The diagram above shows what occurs at runtime when webhooks are triggered.  The process consists of:

1) All webhooks communicate with a single ingress/route as an access point to the cluster.

2) The ingress/route is backed by the `EventListener` service.  The `EventListener` iterates over all of the triggers defined in the `EventListener` custom resource (labeled 3) and sends a request for each trigger to the interceptor service (labeled 4).

3) The interceptor section for each trigger within the `EventListener` contains conditions under which that trigger should operate.  For example, the event is from git repository X and the event is a pull_request.

4) The interceptor service's response to each request determines whether or not the trigger is valid for the incoming webhook event.  The interceptor checks:

    - Valid secret token in request header - secret token defined at webhook creation matches the secret token on the incoming webhook.
    
    - Repository URL matches - so we only activate a trigger for a selected repository.
    
    - Webhook event matches - so we only activate a trigger for a selected event type, a push or pull request event.

5) The Tekton Triggers code creates the necessary `PipelineResources`, `PipelineRuns` etc... as defined in the `TriggerTemplate` - substituting parameters as defined in the user supplied `TriggerBinding` or from the `TriggerBinding` created automatically during webhook creation.

In the case that the event type is a pull request, a monitor taskrun will be created to monitor the `PipelineRuns` and report status onto the pull request in GitHub/Gitlab.
