# Pull Request Status Updates 

Note:- The terms and images used in this documentation relate to Github.  

If the webhook is triggered due to a pull request being created or updated with code, a monitor task will be run to track and report on the status of the configured `PipelineRun` that is started by the `EventListener`.

1.  Each webhook created in the webhooks extension console relates to three triggers registered with the webhooks extension's `EventListener`.

2.  The three triggers are conceptually as follows:  

    - run the relevant `Pipeline` for push events on this repo
    - run the relevant `Pipeline` for pull_request events on this repo
    - run a monitor `Task` for pull_request events on this repo

3.  A `PipelineRun` and `TaskRun` are therefore created by the `EventListener` merging together `TriggerTemplates` with `TriggerBindings`.

4.  The monitor-result-task updates the pull request, putting its status into pending.

![Pending status on pull request](./images/pendingStatus.png?raw=true "Pending status shown on a GitHub pull request")

5.  The monitor-result-task periodically checks the `PipelineRun` for completion and changes the pull request's status accordingly to one of success, failure or error.

![Success status on pull request](./images/successStatus.png?raw=true "Success status shown on a GitHub pull request")

![Failure status on pull request](./images/failStatus.png?raw=true "Failure status shown on a GitHub pull request")

![Error status on pull request](./images/errorStatus.png?raw=true "Error status shown on a GitHub pull request")

6.  A comment is added to the pull request showing the result of the `PipelineRun`. The reported status operates as a hyperlink to the specific `PipelineRun` in the Tekton Dashboard, allowing you to quickly navigate to any relevant log files.  Note that `Unknown` as a status denotes that the `PipelineRun` had not completed before the monitor `TaskRun` reached its maximum polling duration (30 mins).  

![PipelineRun status reporting](./images/comment.png?raw=true "PipelineRun status report as comment on GitHub pull request")


## Notes

1. If you want to change the polling duration or customise the messages or task, further details can be found [here](CustomizingTheMonitor.md).

2. For details about running multiple pipelines from a single webhook, and how the monitor behaves see [here](MultiplePipelines.md).