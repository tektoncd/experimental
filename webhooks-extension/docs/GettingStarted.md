# Getting started with the Tekton Dashboard Webhooks Extension

## Introduction

The [Tekton Dashboard](https://github.com/tektoncd/dashboard) is a general purpose, web-based UI for [Tekton Pipelines](https://github.com/tektoncd/pipeline). The Dashboard [Webhooks Extension](https://github.com/tektoncd/experimental/tree/master/webhooks-extension) allows users to set up GitHub webhooks that will trigger Tekton PipelineRuns and associated TaskRuns. This extension is intended to support Continuous Integration and Continuous Delivery (CI/CD) workflows. Git-driven workflow and automation is a common pattern that we expect most of our readers will be comfortable and familiar with.

This article aims to help you get webhooks up and running with Tekton. We talk about 'GitHub webhooks' in this article because as of August 2019, that is what the webhooks extension currently supports. Support for GitLab is on our roadmap but not yet implemented.

## Installation

See the [webhooks extension readme](https://github.com/tektoncd/experimental/blob/master/webhooks-extension/README.md) for information on how to install Tekton Pipelines, the Tekton Dashboard and the webhooks extension.

## What are webhooks?

As GitHub [states](https://developer.github.com/webhooks/),
> Webhooks allow you to build or set up integrations [which] subscribe to certain events on GitHub.com. When one of those events is triggered, we'll send a HTTP POST payload to the webhook's configured URL.

So a 'webhook' is an outbound HTTP POST from GitHub to a given URL. It contains a rich payload in JSON format describing what just happened in Git. This might be, 'A Pull Request was opened', 'A commit was merged to a branch', or many other events.

It's important to understand that webhooks are outbound HTTP requests from GitHub to a given URL. This URL must be DNS-resolvable and IP-reachable from GitHub. If you are using `github.com` then the target URL must be on the external Internet. It is not generally possible to send a webhook from public `github.com` to a target (normally an Ingress endpoint on a Kubernetes cluster) behind a firewall. You should be using GitHub Enterprise today (or your own GitLab install in the future) if you wish to use webhooks to trigger Tekton Pipelines in a Kubernetes cluster situated behind a firewall.

## What does the Tekton Dashboard's Webhooks Extension do?

The webhooks extension helps you to create a relationship between a Git repository and a Tekton pipeline. This pipeline will be triggered when certain changes happen in the associated repository. Currently these changes are,
- A commit is merged into a branch, and
- A Pull Request is created or modified

See ['Notes On Using The Webhooks Extension'](https://github.com/tektoncd/experimental/tree/master/webhooks-extension#notes-on-using-the-webhooks-extension) for more details on how a webhook payload is turned into a `PipelineRun` and the necessary set of accompanying `PipelineResources`.

## Setting up a webhook from scratch

Several steps must be completed in order to set up a working webhook. Pipeline definitions must be imported, and credentials set up to permit those pipelines to function. Credentials are typically required for GitHub, to create webhooks and clone repositories, and for a Docker image registry to push images to.

Tekton PipelineRuns gain access to Git and Docker registry credentials via their associated service account.
- Git credentials must be stored in a secret with an annotation, `tekton.dev/git-0: [url to Git repository]`
- Docker registry credentials must be stored in a secret with an annotation, `tekton.dev/docker-0: [url to docker registry]`
- credentials are located via the service account used by a given Tekton TaskRun or PipelineRun
See the [Tekton documentation](https://github.com/tektoncd/pipeline/blob/master/docs/auth.md#basic-authentication-git) for more information.

Before you start creating Tekton resources, decide which Kubernetes namespace(s) you wish to execute pipelines in, and which service accounts will be used to execute them. Ensure that each service account has sufficient permissions under Kubernetes [Role-Based Access Control (RBAC)](https://kubernetes.io/docs/reference/access-authn-authz/rbac/) to execute your intended pipelines. If you're just testing, the `tekton-webhooks-extension` service account in the `tekton-pipelines` namespace can be used to run a wide range of pipelines by virtue of the `tekton-webhooks-extension-minimal` role and binding.

Having set up the right namespaces and service accounts you can now proceed to create the necessary secrets. The Tekton dashboard has a 'Secrets' menu that can help with the creation of credentials for use within Tekton pipelines.

### Create credentials: Git

You will need to create a secret for Git if any of your GitHub repositories require authentication in order to clone them. This is true for GitHub private repositories, and for many GitHub Enterprise installations. While Tekton supports SSH and Basic authentication for Git, only the latter is supported by the webhooks extension today. Basic authentication can be used to provide either username/password, or an access token. Access tokens are our recommended approach.

Create an access token via GitHub. As of August 2019, for a given GitHub repository, visit [your name icon in the top right corner] > Settings > Developer settings > Personal access tokens > Generate new token and create a token with `repo` permissions. Add `admin:repo_hook` permissions if you are going to use the same access token to create webhooks.

Store the access token in a Kubernetes secret via the Tekton Dashboard's 'Secrets' menu. This panel will help you add the right annotation to the secret, and patch it onto the specified service account. Annotations should typically be of the form, `tekton.dev/git-0: https://github.com`. If you are using an access token, store it in the 'password' field. Be sure to specify the service account that will be used by the associated Tekton pipeline.

![Creating a Git secret](./images/createGitSecret.png?raw=true "Creating a Git secret")

### Create credentials: Docker

Credentials are normally required to push images to a Docker registry. Some Docker registries support access tokens. Others such as Docker Hub only support username/password. If you are pushing images to Docker Hub the annotation should be of the form, `tekton.dev/docker-0: https://index.docker.io/v1/`. Again, select the service account of the associated Tekton pipeline so that it is correctly patched with the new Docker credentials.

### Import pipeline definitions

A Tekton Pipeline is composed of one or more Tasks. We expect that most users will store their Pipeline and Task definitions in Git. You may wish to `git clone` and `kubectl apply` these definitions manually, or more likely, as part of an automated installation and setup. For your convenience, the dashboard provides a menu option, 'Import Tekton resources'.

![Import pipeline definitions](./images/importPipelines.png?raw=true "Importing pipeline definitions")

This screen shot shows a user importing pipeline definitions from a Git repository. This panel will drive the Pipeline `pipeline0`


## Creating a new webhook

The Tekton dashboard webhooks extension creates an association between a Git repository and a Tekton pipeline. Changes to a branch or pull request in the associated Git repository will trigger a new PipelineRun with its accompanying PipelineResource objects. This section covers the various setup and configuration options in more detail. The image below shows the current form of the main 'Create Webhook' panel in the dashboard.

![Creating a new webhook](./images/createWebhook.png?raw=true "Creating a new webhook")

### Name

A descriptive name for this webhook. Names must currently be unique; we aim to relax this restriction.

### Repository URL

The URL of the Git repository in which the webhook should be created. For example, `https://github.com/myorg/myrepo.git`. The '.git' suffix can be kept or omitted; it does not matter.

### Access Token

A GitHub access token that will be used to create the webhook. You may have created an access token in the section  'Create credentials: Git' above. If you chose to have one access token for both checking out code and creating webhooks then you can reuse its value here. Click [here](https://help.github.com/en/articles/creating-a-personal-access-token-for-the-command-line) to learn how to generate an access token.

1. If you did not do so earlier, create an access token via GitHub as described [above](#create-credentials-git) ensuring that the token has `admin:repo_hook` permissions.
2. Access tokens for webhooks are stored differently to credentials for Tekton Pipelines. You must store an access token via the 'Access Token' menu even if you created an access token-based Git secret earlier.

### Namespace

Select the Kubernetes namespace in which the triggered Pipeline should run.

### Pipeline

Select the Pipeline in the target Namespace from which a PipelineRun (and accompanying PipelineResources) will be created, as described in the section 'What does the Tekton Dashboard's Webhooks Extension do?', above.

### Service Account

This field specifies the service account that will be used by the PipelineRun. It should be the same service account that you set up RBAC permissions for in the section 'Setting up a webhook from scratch' and that you patched credentials onto in the sections, 'Create credentials: Git' and 'Create credentials: Docker', above.

### Docker Registry

Finally, select the Docker registry that any built images should be pushed to. This is the Docker registry that the 'Docker' credentials earlier are associated with.
Accepted Formats:
- http://index.docker.io/foo
- index.docker.io/foo
- foo
- https://index.docker.io/foo
- http://my.registry/foo
- https://my.registry/foo
- http://my.registry
- https://my.registry
- my.registry/foo

## Putting it all together: test it's working

Once a webhook is set up, a `git push` or creation of a pull request to the monitored repository should trigger the creation of the correct PipelineRun. This PipelineRun will show up in the Tekton dashboard as usual. As of August 2019, a successful webhook will trigger pods to be creted as follows:

- Firstly, webhook receipt will drive a pod whose name will be of the format `tekton-xwxm5-rp6s6-gldvf-deployment-6674f7fdbb-pdgmw` created.

- Next this pod will trigger the webhooks extension 'sink': a pod of the form `webhooks-extension-sink-dqxkm-deployment-5f64979c5b-8dk4k` will execute.

- The 'sink' will then create the expected PipelineRun. The PipelineRun will spawn pods for each of its Tasks, so for example we might see a pod `buildah-hook-1565009498-build-simple-6qc72-pod-1f0a5e` created to run the `build-simple` Task.

- If the webhook's event type is a pull request, an additional pod will be seen for the monitoring task. This pod will monitor the PipelineRun and update the pull request with status.  The default monitoring task pod will be named similar to `pr-monitor-15657005244rsz8-pod-f5b42a`. For more on monitoring see [here](Monitoring.md)

- Finally the 'sink' and '(webhook) Name' pods will be shut down by [Knative](https://knative.dev/docs/).

You can use `kubectl logs [pod-name] --all-containers` to check the output of each pod in turn, and of course the Tekton dashboard for the pods managed by a PipelineRun. In the case of any problems, check that all of the above steps were correctly performed:
- Create a service account and RoleBinding for the PipelineRuns to use
- Create the correct Git and Docker credentials and patch the right service account
- Ensure that your GitHub can route correctly to the webhooks-extension-sink: use `kubectl get kservice` to check its value, and the GitHub web pages to see that the webhook was correctly created, and that it successfully delivered its result to the [kservice](https://github.com/knative/serving/blob/master/docs/spec/overview.md#service).

## Troubleshooting

### 404 or 503 error in GitHub (red "x" next to your webhook)

Sometimes the sink is not ready in time to receive a webhook event, and GitHub will report a 404 or 503 error. If this happens, you can redeliver the event from the GitHub webpage under the "Settings" > "Hooks" section to fix the problem.
