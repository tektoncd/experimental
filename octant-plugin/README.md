# Tekton Octant Plugin

This provides a plugin to the Octant dashboard, which displays additional useful
information about Tekton custom resources.

# Screenshots

## Task
![Image of Task additions](./task.png)

* Lists input resources and parameters, and output resources
* TODO: links to recent TaskRuns

## TaskRun
![Image of TaskRun additions](./taskrun.png)

* Links Pod status (with logs!), and Task definition
* Displays queued time and duration
* TODO: Graphviz visualization of status

## Pipeline
![Image of Pipeline additions](./pipeline.png)

* Lists input resources and parameters
* Links Task definition
* TODO: Graphviz of Pipeline configuration
* TODO: links to recent PipelineRuns

## PipelineRun
![Image of PipelineRun additions](./pipelinerun.png)

* Links Pipeline definition
* Displays queued time and duration
* TODO: Graphviz visualization of status


# Testing

`go build -o ~/.config/octant/plugins/tekton-plugin ./`

Then restart `octant` which will open a new tab.
