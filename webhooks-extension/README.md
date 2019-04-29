# Webhooks Extension
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/kubernetes/experimental/blob/master/LICENSE)

The Webhooks Extension for Tekton provides allows users to set up Git webhooks that will trigger Tekton PipelineRuns and TaskRuns. An initial implementation will use Knative Eventing but we're closely following the eventing discussion in  [Tekton Pipeline](https://github.com/tektoncd/pipeline) to minimize necessary componentry.

In addition to Tekton/Knative Eventing glue, it includes an extension to the Tekton dashboard.

## How it works?
When a webhookt trigger is receive via cloudevent, this extension looks for pipeline and pipelinerun file in .tekton directoy. Then instantiates pipeline if it doesn't exist. It instantiates or update the git resource with commit id of trigger. And then run pipelinerun using pipelinerun.yaml file.

## Run
### Local Outside the cluster
Set up envionment variable. Github token can be generated from https://github.com/settings/tokens


```
export KUBECONFIG=KUBECONFIGLOCATION
export GITHUB_AUTH_TOKEN=TOKENFROMGITHUB   #

```
Build

```
cd cmd/trigger
go build
./trigger
```
Trigger service will run on 8080 port.

### Inside the cluster

Setup the github environment variable. Github token can be generated from https://github.com/settings/tokens

Build the image

```
docker build . -t repo:tag
```

Deploy the image in cluster as cluster.

## Example trigger

https://github.com/khrm/knative-eventing-extension-example


You can either build and run locally outside the cluster by setting KUBECONFIG and GITHUB_AUTH_TOKEN environment variable. 

Or you can run it inside the cluster after 

## Want to get involved

Visit the [Tekton Community](https://github.com/tektoncd/community) project for an overview of our processes.
