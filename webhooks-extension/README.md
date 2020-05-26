# Webhooks Extension

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/kubernetes/experimental/blob/master/LICENSE)

The Webhooks Extension for Tekton allows users to set up GitHub or Gitlab webhooks that will trigger Tekton `PipelineRuns` and associated `TaskRuns`.  This is possible via an extension to the Tekton Dashboard and via REST endpoints.

See our [Getting Started](https://github.com/tektoncd/experimental/blob/master/webhooks-extension/docs/GettingStarted.md) guide for more on what this extension does, and how to use it.

  ![Create webhook page in dashboard](./docs/images/createWebhook.png?raw=true "Create webhook page in dashboard")

### Install Prereqs

[Install and Configure Prereqs](./docs/InstallPrereqs.md)  

### Install Webhook Extension

To install an official release please navigate to the docs for that release. In the branches dropdown at the top of this page, simply select the branch name matching the version you want to install. 

Note that if you're going from Triggers 0.4 to Triggers 0.5 (for example, as part of upgrading the webhooks extension), you must:

1) delete any existing TriggerTemplates with `type:` in them, to avoid

```
for: "STDIN": admission webhook "webhook.triggers.tekton.dev" denied the request: mutation failed: cannot decode incoming new object: json: unknown field "type"
```
errors.

2) `kubectl delete deployment webhooks-extension` (which will be in either the `tekton-pipelines` or `openshift-pipelines` namespace depending on your platform). This is to prevent an immutable field type error.

[Installing Official Release (stable)](./docs/InstallReleaseBuild.md)

[Installing Development Build (nightly)](./docs/InstallNightlyBuild.md)

As a convenience, the **/test/install_dashboard_and_extension.sh** script can be
used to install a specified version of the dashboard and the webhook extension.  

### Usage Guides

[Getting Started](./docs/GettingStarted.md)  
[Parameters Available To Trigger Templates](./docs/Parameters.md)  
[Labelling Pipeline Runs For UI Display](./docs/Labels.md)  
[Multiple Pipelines](./docs/MultiplePipelines.md)  
[Pull Request Status Updates](./docs/Monitoring.md)  
[Security](./docs/Security.md)  
[Additional Notes If Using Red Hat OpenShift](./docs/NotesOnOpenShiftInstallations.md)  
[Additional Notes If Using Amazon EKS](./docs/AmazonEKS.md)  
[Limitations](./docs/Limitations.md)  

### Architecture Guide

[Architecture](./docs/Architecture.md)

### Uninstall

[Uninstall](./docs/Uninstall.md)

## Want to get involved?

Visit the [Tekton Community](https://github.com/tektoncd/community) project for an overview of our processes.

## Information for developers

If you are looking to develop or contribute to this repository please see the [development docs](https://github.com/tektoncd/experimental/blob/master/webhooks-extension/DEVELOPMENT.md)

For more involved development scripts please see the [development installation guide](https://github.com/tektoncd/experimental/blob/master/webhooks-extension/test/README.md#scripting)
