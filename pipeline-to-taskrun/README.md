# Pipeline to TaskRun

This project is a controller that enables an experimental [custom task](https://github.com/tektoncd/pipeline/blob/main/docs/runs.md)
that will allow you to execute a Pipeline (with [limited features](#supported-pipeline-features)) via a TaskRun, enabling you to
run a Pipeline in a pod ([TEP-0044](https://github.com/tektoncd/community/blob/main/teps/0044-decouple-task-composition-from-scheduling.md)).

This can be useful when you want to combine the functionality of several Tasks, but you don't want to have to deal with
the additional overhead of running multiple pods and/or creating volumes that can be used to share data between them.

A common use case for this is wanting to, with just one pod and without needing to create or worry about volumes,
the pattern of:
1. Clone from version control (e.g. cloning with [git-clone](https://github.com/tektoncd/catalog/tree/master/task/git-clone/0.4))
2. Do something with the data (e.g. run tests with [golang-test](https://github.com/tektoncd/catalog/tree/master/task/golang-test/0.1))
3. Uplod the results somewhere (e.g. upload to a GCS bucket with [gcs-upload](https://github.com/tektoncd/catalog/tree/master/task/gcs-upload/0.1))

* [Usage](#usage)
  * [Invoke via a `Run`](#invoke-via-a-run)
  * [Invoke from a `Pipeline`](#invoke-from-a-pipeline)
* [Supported Pipeline Features](#supported-pipeline-features)
* [Install](#install)
  * [From nightly release](#from-nightly-release)
  * [Build and install](#build-and-install)
* [Examples](#examples)
* [Tests](#tests)

## Usage

### Invoke via a `Run`

To execute a Pipeline via a TaskRun using this custom task, create a [`Run`](https://github.com/tektoncd/pipeline/blob/master/docs/runs.md)
with:

* `apiVersion: tekton.dev/v1alpha1`
* `kind: PipelineToTaskRun`
* `name: PipelineName` - Where `PipelineName` is the name of the Pipeline you'd like to run
* Any required runtime information (e.g. params), complete list below

The full list of supported fields:

- [`apiVersion`][kubernetes-overview] - Specifies the API version, `tekton.dev/v1alpha1`
- [`kind`][kubernetes-overview] - Identifies this resource object as a `Run` object
- [`metadata`][kubernetes-overview] - Specifies the metadata that uniquely identifies the `Run`, such as a `name`
- [`spec`][kubernetes-overview] - Specifies the configuration for the `Run`
    - [`ref`][kubernetes-overview] - Specifies the Pipeline to TaskRun `Custom Task`
        - [`apiVersion`][kubernetes-overview] - Specifies the API version, `tekton.dev/v1alpha1`
        - [`kind`][kubernetes-overview] - Identifies this resource object as a `PipelineToTaskRun` object
        - [`name`][kubernetes-overview] - Identifies the `Pipeline` object to be executed
    - Optional:
      - [`params`](https://github.com/tektoncd/pipeline/blob/main/docs/pipelineruns.md#specifying-parameters) - Specifies values for
        [pipeline level params](https://github.com/tektoncd/pipeline/blob/main/docs/pipelines.md#specifying-workspaces)
      - [`serviceAccountName`](https://github.com/tektoncd/pipeline/blob/main/docs/pipelineruns.md#specifying-custom-serviceaccount-credentials) - Specifies a `ServiceAccount` to use for the TaskRun.
      - [`workspaces`](https://github.com/tektoncd/pipeline/blob/main/docs/pipelineruns.md#specifying-workspaces) - Specifies the physical volumes to use for
        [the workspaces required by the Pipeline](https://github.com/tektoncd/pipeline/blob/main/docs/pipelines.md#specifying-workspaces)

See [pipeline-taskrun-run.yaml](examples/pipeline-taskrun-run.yaml) for a complete example
([the examples section shows you how to try it out](#examples).

### Invoke from a `Pipeline`

You can use this custom task to run a sub pipeline (similar to [pipeline in a pipeline](../pipelines-in-pipelines)) but
within one TaskRun. To do this [specify the PipelineToTaskRun custom task](https://github.com/tektoncd/pipeline/blob/main/docs/pipelines.md#specifying-the-target-custom-task):

* `apiVersion: tekton.dev/v1alpha1`
* `kind: PipelineToTaskRun`
* `name: PipelineName` - Where `PipelineName` is the name of the Pipeline you'd like to run

You can provide the following runtime values in the same way as you would for a
[task in a pipeline](https://github.com/tektoncd/pipeline/blob/main/docs/pipelines.md#adding-tasks-to-the-pipeline):
- `params` - Specifies values for [pipeline level params](https://github.com/tektoncd/pipeline/blob/main/docs/pipelines.md#specifying-workspaces)
- [`workspaces`](https://github.com/tektoncd/pipeline/blob/main/docs/pipelines.md#specifying-workspaces) - Specifies the physical volumes to use for
  [the workspaces required by the Pipeline](https://github.com/tektoncd/pipeline/blob/main/docs/pipelines.md#specifying-workspaces)

See [pipeline-taskrun-pipelinerun.yaml](examples/pipeline-taskrun-pipelinerun.yaml) for a complete example
([the examples section shows you how to try it out](#examples).
  
## Supported Pipeline Features

Since this custom task works by executing a Pipeline as a [TaskRun](https://github.com/tektoncd/pipeline/blob/main/docs/taskruns.md)
it can only support a subset of Pipeline features.

Currently supported features:

* Sequential tasks (specified using [`runAfter`](https://github.com/tektoncd/pipeline/blob/main/docs/pipelines.md#using-the-runafter-parameter))
* [String params](https://github.com/tektoncd/pipeline/blob/main/docs/pipelines.md#specifying-parameters)
* [Workspaces](https://github.com/tektoncd/pipeline/blob/main/docs/pipelines.md#specifying-workspaces)
  * Including [optional workspaces](https://github.com/tektoncd/pipeline/blob/main/docs/workspaces.md#optional-workspaces)

### Potential future features

These features may be added in the future:

* [Array params](https://github.com/tektoncd/pipeline/blob/main/docs/tasks.md#specifying-parameters) - since we do a lot
  of variable renaming and replacement, array params were left out of the initial version
* Passing workspace paths and params via pipeline tasks - Since [all uses of params are namespaced](#params)
  and [workspaces are remapped](#workspaces), all uses of these via variable replacement must be updated. This
  has been applied to the Task definitions, but not to the pipeline tasks where they can also be used
  via param values.
* [Pipeline level results](https://github.com/tektoncd/pipeline/blob/main/docs/pipelines.md#emitting-results-from-a-pipeline)
* Exposing Task results as Pipeline level results
* [Passing results between tasks](https://github.com/tektoncd/pipeline/blob/main/docs/pipelines.md#passing-one-tasks-results-into-the-parameters-or-whenexpressions-of-another)
* [Sidecars](https://github.com/tektoncd/pipeline/blob/main/docs/tasks.md#specifying-sidecars)
  (if we support this, all would have to start up simultaneously which may not be the desired behavior)
* Workspace features:
  * [mountPaths](https://github.com/tektoncd/pipeline/blob/main/docs/workspaces.md#using-workspaces-in-tasks)
  * [subPaths](https://github.com/tektoncd/pipeline/blob/main/docs/workspaces.md#using-workspaces-in-pipelines)
  * [readOnly](https://github.com/tektoncd/pipeline/blob/main/docs/workspaces.md#using-workspaces-in-tasks)
  * [isolated](https://github.com/tektoncd/pipeline/blob/main/docs/workspaces.md#isolating-workspaces-to-specific-steps-or-sidecars)
  * In addition further thought will have to be given to support workspaces that combine
    [mountPaths](https://github.com/tektoncd/pipeline/blob/main/docs/workspaces.md#using-workspaces-in-tasks) and
    or [Pipeline task level subpaths](https://github.com/tektoncd/pipeline/blob/main/docs/workspaces.md#using-workspaces-in-pipelines) with
    [volumeclaimtemplates](https://github.com/tektoncd/pipeline/blob/main/docs/workspaces.md#volumeclaimtemplate)
    (see [tektoncd/pipeline#3440](https://github.com/tektoncd/pipeline/issues/3440) - it is not possible to have two
    different workspace declarations in the taskspec which are mapped to one volumeClaimTemplate at runtime)
* Specifying Tasks in a Pipeline via [Bundles](https://github.com/tektoncd/pipeline/blob/main/docs/tekton-bundle-contracts.md)
* These fields would be easy to support one of, but it's not clear how to handle cases where more than one task declares them (since in the taskrun they would apply to the entire task):
    * [step templates](https://github.com/tektoncd/pipeline/blob/main/docs/tasks.md#specifying-a-step-template)
    * [timeout](https://github.com/tektoncd/pipeline/blob/main/docs/pipelines.md#configuring-the-failure-timeout)
    * [retries](https://github.com/tektoncd/pipeline/blob/main/docs/pipelines.md#using-the-retries-parameter)
* [Volumes and volume mounts](https://github.com/tektoncd/pipeline/blob/main/docs/tasks.md#specifying-volumes) -
  still [not clear that we really need these in addition to workspaces](https://github.com/tektoncd/pipeline/issues/2058)
* Contextual variable replacement that assumes a PipelineRun, for example [`context.pipelineRun.name`](https://github.com/tektoncd/pipeline/blob/main/docs/variables.md#variables-available-in-a-pipeline)

### Features unlikely to be supported

These features are not supported by TaskRuns so this custom task is unlikely to support them (unless the design
is changed substantially, [see "What comes next?" in the proposal](https://github.com/tektoncd/community/issues/447)):

* [Parallel tasks](https://github.com/tektoncd/pipeline/blob/main/docs/pipelines.md#configuring-the-task-execution-order)
* [When expressions](https://github.com/tektoncd/pipeline/blob/main/docs/pipelines.md#guard-task-execution-using-whenexpressions)
  (and [Conditions](https://github.com/tektoncd/pipeline/blob/main/docs/pipelines.md#guard-task-execution-using-conditions))
* [Custom tasks](https://github.com/tektoncd/pipeline/blob/main/docs/pipelines.md#using-custom-tasks)
* [Finally tasks](https://github.com/tektoncd/pipeline/blob/main/docs/pipelines.md#adding-finally-to-the-pipeline)
  (maybe if we allow step failure ([TEP-0040](https://github.com/tektoncd/community/blob/main/teps/0040-ignore-step-errors.md))
  we can use that to make finally steps work??)
* PipelineResources - both because of
  [questions around the future of the feature](https://github.com/tektoncd/pipeline/blob/main/docs/resources.md#why-arent-pipelineresources-in-beta)
  and because TaskRuns have no [linking via from](https://github.com/tektoncd/pipeline/blob/main/docs/pipelines.md#using-the-from-parameter)
  
## How does this work?

This custom task will take a Pipeline (using [Supported Pipeline Features](#supported-pipeline-features)) and
run in one TaskRun by combining the steps from each Task into one Task spec embedded in the TaskRun.
But how does this actually work, considering that there can be collisions and duplication between names
of params, steps, workspaces, etc.?

### Params

The custom task will add params of each of the Pipeline's Tasks to the resulting task spec. To deal with collisions,
each param is namespaced by prepending it with the name of the pipeline task it came from.

For example, given the following pipeline Task:

```yaml
    - name: grab-source
      taskRef:
        name: git-clone
      params:
        - name: url
          value: $(params.git-url)
```

Using a Task which declares this params:

```yaml
  params:
    - name: url
      description: git url to clone
      type: string
```

The resulting Task spec in the TaskRun that executes the Pipeline will declare:

```yaml
    params:
      - description: git url to clone
        name: grab-source-url
        type: string
```

Fields in the Task which used variable replacements for these params will be updated to use the new
name, for example this portion of the step's script:

```yaml
        /ko-app/git-init \
        -url "$(params.url)" \
```

Will become:

```yaml
        /ko-app/git-init \
        -url "$(params.grab-source-url)" \
```

### Steps

The custom task will add the step of each of the Pipeline's Tasks to the resulting task spec. To deal with collisions,
each step is namespaced by prepending it with the name of the pipeline task it came from. If the step
has no name, it will be left unnamed.

For example, given the following pipeline Task:

```yaml
    - name: grab-source
      taskRef:
        name: git-clone
      params:
        - name: url
          value: $(params.git-url)
```

Which contains this step:

```yaml
  steps:
    - name: clone
```

The step in the Task spec of the resulting TaskRun that executes the Pipeline will contain this step:

```yaml
  steps:
    - name: grab-source-clone
```

_What if the resulting step name is too long to be a valid container? It will be truncated to the maximum length
of 63 characters._

### Workspaces

Workspaces that are declared in a Pipeline and passed to Tasks must be remapped to make sense in the context
of a TaskRun. This means removing a layer of Workspace mapping:

* **PipelineRun**: In the context of a PipelineRun (the normal mode of Pipeline execution),
  there will be the following layers of mapping:
  * A Task declares Workspaces it needs
  * A Pipeline declares Workspace it needs
  * In each Pipeline Task, the Pipeline will map from its declared Workspaces to the Workspaces the Task needs
  * In the PipelineRun, actual volumes/secrets/etc will be provided for each Workspace the Pipeline declares
* **Task**: In the Context of a TaskRun (which the Pipeline will be mapped to), there will be the following
  layers of mapping:
  * A Task declares Workspaces it needs
  * In the TaskRun, actual volumes/secrets/etc will be provided for each Workspace the Pipeline declares
 
So in order to map a Pipeline's execution into a TaskRun, we need to remove the intermediary layer of the
Workspaces declared by the Pipeline and mapped to each Pipeline Task.

For example, given these two Tasks's workspace declarations:

```yaml
apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: git-clone
  workspaces:
    - name: output
  steps:
    - name: clone
      image: some-git-image
      script: |-
        echo $(workspaces.output.path)
---
apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: gcs-upload
  workspaces:
    - name: credentials
    - name: source
  steps:
    - name: upload
      image: some-gsutil-image
      script: |-
        echo $(workspaces.source.path)
        echo $(workspaces.credentials.path)
```

Say that a Pipeline maps these workspaces like this:

```yaml
spec:
  workspaces:
    - name: where-it-all-happens
    - name: gcs-creds
  tasks:
    - name: grab-source
      taskRef:
        name: git-clone
      workspaces:
        - name: output
          workspace: where-it-all-happens
    - name: upload-results
      taskRef:
        name: gcs-upload
      workspaces:
        - name: source
          workspace: where-it-all-happens
        - name: credentials
          workspace: gcs-creds
```

And finally in the custom task Run, the workspaces are defined like this:

```yaml
    workspaces:
    - name: where-it-all-happens
      volumeClaimTemplate:
        spec:
          accessModes:
            - ReadWriteOnce
          resources:
            requests:
              storage: 1Gi
    - name: gcs-creds
      secret:
        secretName: mikey
```

What we ultimately have is 1 workspace that is mapping to a secret called `mikey` and
2 workspaces mapping to a persistent volume claim template - indicating that ultimately
both of those workspaces are intended to use the same volume.

To ensure that this intent is respected, instead of declaring 3 different workspaces in our
generated TaskRun, we will declare just one workspace which will be bound to the `volumeClaimTemplate`
and we will rewrite the Tasks to use this workspace.

For the above example, the resulting TaskRun will look like this:

```yaml
spec:
  taskSpec:
    workspaces:
      - name: where-it-all-happens
      - name: gcs-creds
    steps:
      - name: clone
        image: some-git-image
        script: |-
          echo $(workspaces.where-it-all-happens.path)
      - name: upload
        image: some-gsutil-image
        script: |-
          echo $(workspaces.where-it-all-happens.path)
          echo $(workspaces.gcs-creds.path)
  workspaces:
    - name: where-it-all-happens
      volumeClaimTemplate:
        spec:
          accessModes:
            - ReadWriteOnce
          resources:
            requests:
              storage: 1Gi
    - name: gcs-creds
      secret:
        secretName: mikey
```

## Install

### From nightly release

This controller is published nightly via automation in [tetkon](./tekton). Install the latest nightly release with:

```
kubectl apply --filename https://storage.cloud.google.com/tekton-releases-nightly/pipeline-to-taskrun/latest/release.yaml
```

### Build and install

1. Install and configure [`ko`](https://github.com/google/ko).

2. Install with `ko`:
```
ko apply -f config/
```

This will build and install the `Pipeline-To-TaskRun Controller` on your cluster, in the namespace `tekton-pipeline-to-taskrun`.

```commandline
$ k get pods -n tekton-pipeline-to-taskrun

NAME                              READY   STATUS    RESTARTS   AGE
pipeline-to-taskrun-controller-654bdc4cc8-7bvvn   1/1     Running   0          3m4s
```

To look at the logs:

```
kubectl -n tekton-pipeline-to-taskrun logs $(kubectl -n tekton-pipeline-to-taskrun get pods -l app=pipeline-to-taskrun-controller -o name)
```

## Examples

The example pipeline [clone-test-upload.yaml](examples/clone-test-upload.yaml) is a Pipeline that will:
1. Download from git
2. Run `go test` and capture the output
3. Upload the output to GCS

### Requirements

* [Enable custom tasks](#enable-custom-tasks-in-tekton-pipelines)
* [Get GCS credentials](#gcs-credentials)

#### Enable custom tasks in Tekton Pipelines

To run the example that invokes the custom task from a Pipeline
([examples/pipeline-taskrun-pipelinerun.yaml](examples/pipeline-taskrun-pipelinerun.yaml)) you must enable custom
tasks in Tekton Pipelines
[by setting `enable-custom-tasks` to true](https://github.com/tektoncd/pipeline/blob/main/docs/install.md#customizing-the-pipelines-controller-behavior).

#### GCS Credentials

In order to run (3) you will need to grab GCS credentials and store them in secret. The Pipeline expects a secret to be
provided via the workspaces `gcs-creds`
[at the path `service-account.json`](https://github.com/tektoncd/catalog/tree/main/task/gcs-upload/0.1#parameters) that
corresponds to a service account that has
[bucket write permissions (e.g. storage object admin)](https://cloud.google.com/storage/docs/access-control/iam-permissions).

### Running

```bash
# Install the Tasks from the catalog that we'll be using in our Pipeline
tkn hub install task git-clone
tkn hub install task golang-test
tkn hub install task gcs-upload

# Install the Pipeline that we'll be running
kubectl apply -f examples/clone-test-upload.yaml

# To make sure everything is working, you can create the equivalent PipelineRun
# In this example we're using a secret called `mikey` to upload to the bucket `christies-empty-bucket`
# and it will run the unit tests for tektoncd/chains (as a random example with a quick test suite :D)
tkn pipeline start clone-test-upload \
    -p git-url="https://github.com/tektoncd/chains" \
    -p package="github.com/tektoncd/chains/pkg" \
    -p packages="./pkg/..." \
    -p gcs-location="gs://christies-empty-bucket" \
    -w name=where-it-all-happens,volumeClaimTemplateFile=examples/pvc.yaml \
    -w name=gcs-creds,secret=mikey

# make the pvc we'll be using
kubectl create -f examples/run-pvc.yaml

# run as a Run (using the same config as the `tkn pipeline start` above)
kubectl create -f examples/pipeline-taskrun-run.yaml

# run as a custom task invoked from another pipeline (using the same config as the `tkn pipeline start` above)
kubectl create -f examples/pipeline-taskrun-pipelinerun.yaml
```

## Tests

```
go test ./pkg/...
```

[kubernetes-overview]:
https://kubernetes.io/docs/concepts/overview/working-with-objects/kubernetes-objects/#required-fields
