
## About code

This operator is a set of controllers that can be divided in two categories:

### Monitors Controllers

This controllers react to changes in TaskMonitor, TaskRunMonitor,
PipelineMonitor or PipelineRunMonitor, and updates the internal metric
definition.

### Runs Controllers

This controllers react to changes in TaskRun and PipelineRun and updates the
metrics registered. Given the nature of controllers, its required to be careful
to avoid counting the same run twice or not counting at all.
