# Pull Request Status Updates 

If the webhook is triggered due to a pull request being created (or updated with code), a monitor task will be run to track and report on the status of the configured PipelineRun that is started by the webhooks-extension.  The sequence of events are as follows:

1.  Creation of a PipelineRun by the webhooks-extension's `sink` handler occurs in response to the configured webhook.

2.  When the PipelineRun is created, a TaskRun is also created for the `monitor-result-task` Task.

3.  The monitor-result-task updates the pull request, putting its status into pending.

![Pending status on pull request](./images/pendingStatus.png?raw=true "Pending status shown on a GitHub pull request")

4.  The monitor-result-task periodically checks the PipelineRun for completion and changes the pull request's status accordingly to one of success, failure or error.

![Success status on pull request](./images/successStatus.png?raw=true "Success status shown on a GitHub pull request")

![Failure status on pull request](./images/failStatus.png?raw=true "Failure status shown on a GitHub pull request")

![Error status on pull request](./images/errorStatus.png?raw=true "Error status shown on a GitHub pull request")

5.  A comment is added to the pull request showing the result of the PipelineRun. The reported status operates as a hyperlink to the specific PipelineRun in the Tekton Dashboard, allowing you to quickly navigate to any relevant log files.  Note that `??????` as a status denotes that the PipelineRun had not completed before the monitor Task reached its maximum polling duration (30 mins).  

![PipelineRun status reporting](./images/comment.png?raw=true "PipelineRun status report as comment on GitHub pull request")


## Notes

1. If you want to change the polling duration or customise the messages or task, further details can be found [here](CustomizingTheMonitor.md).

2. For details about running multiple pipelines from a single webhook, and how the monitor behaves see [here](MultiplePipelines.md).