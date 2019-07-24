# Triggering Multiple Pipelines

## Introduction

It is possible to trigger multiple pipelines by creating multiple webhooks against the same repository in the Tekton dashboard UI.  Under the covers, only a single webhook is actually created on the git repository itself.

The Tekton dashboard UI will list this 'multiple pipeline' configuration as mutiple webhooks, but if you were to look in GitHub itself you would see only one hook.

![Multiple pipelines](./images/twoPipelines.png?raw=true "Multiple pipelines shown as two webhooks in the Tekton UI")

NOTE: The GitHub webhook is created for you - you do not manually create it.

![Multiple pipelines - Single GitHub webhook](./images/singleGHHook.png?raw=true "A single webhook shown on the GitHub repository, in GitHub")

## Pull Request Status Updates

If you have configured multiple pipelines against a repository, a single `monitor-result-task` TaskRun is created that monitors all the PipelineRuns created as a result of the webhook triggering.  The overall status is reported as `success` **only** if all the PipelineRuns succeed.

The comment uploaded onto the pull request will detail the individual status of **all** the PipelineRuns that were created.  Below, you can see that one PipelineRun failed and one succeeded, thus the overall status is set to failed.

![Status reporting for multiple pipelines](./images/multiplePRs.png?raw=true "Status reporting for multiple pipelines")

For more details on the monitoring see [here](Monitoring.md)
