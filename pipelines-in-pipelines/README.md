# Pipelines In Pipelines

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/tektoncd/experimental/blob/master/LICENSE)

This is an experimental project to provide support for `Pipelines` in `Pipelines` to improve the composability and 
reusability of [Tekton Pipelines](https://github.com/tektoncd/pipeline). The functionality is provided by a controller 
that implements the `Custom Task` interface. 

Its use cases include enabling defining and executing a set of `Tasks` as a complete unit of execution and decoupling `Pipeline` 
branches failures such that a `Task` failure in one branch does not stop execution of unrelated `Pipeline` branches.

- [Pipelines In Pipelines](#pipelines-in-pipelines)
  - [Install](#install)
  - [Usage](#usage)
    - [Configuring a `Pipeline` in a `Run`](#configuring-a-pipeline-in-a-run)
    - [Configuring a `Pipeline` in a `Pipeline`](#configuring-a-pipeline-in-a-pipeline)
    - [Monitoring Execution Status](#monitoring-execution-status)
  - [Uninstall](#uninstall)
  - [Contributions](#contributions)

## Install

Install and configure [`ko`](https://github.com/google/ko).

```
ko apply -f config/
```

This will build and install the `Pipelines-In-Pipelines Controller` on your cluster, in the namespace `tekton-pip-run`.

```commandline
$ k get pods -n tekton-pip-run

NAME                              READY   STATUS    RESTARTS   AGE
pip-controller-654bdc4cc8-7bvvn   1/1     Running   0          3m4s
```

Alternatively, install it from the nightly release using:

```commandline
kubectl apply --filename https://storage.cloud.google.com/tekton-releases-nightly/pipelines-in-pipelines/latest/release.yaml
```

## Usage

### Configuring a `Pipeline` in a `Run`

To specify a `Pipeline` in a `Pipeline`, we need to define a [`Run`](https://github.com/tektoncd/pipeline/blob/master/docs/runs.md)
type with `apiVersion: tekton.dev/v1beta1`, `kind: Pipeline` and pass in the name of the `Pipeline` to be run.

See the [examples](examples) folder for the `Pipelines` in `Pipelines` `Custom Tasks` to run or use as samples.

The `Pipeline` in `Pipeline` `Custom Task` is defined in a `Run`, which supports the following fields:

- [`apiVersion`][kubernetes-overview] - Specifies the API version, `tekton.dev/v1alpha1`
- [`kind`][kubernetes-overview] - Identifies this resource object as a `Run` object
- [`metadata`][kubernetes-overview] - Specifies the metadata that uniquely identifies the `Run`, such as a `name`
- [`spec`][kubernetes-overview] - Specifies the configuration for the `Run`
- [`ref`][kubernetes-overview] - Specifies the `Pipeline` in `Pipeline` `Custom Task`
    - [`apiVersion`][kubernetes-overview] - Specifies the API version, `tekton.dev/v1beta1`
    - [`kind`][kubernetes-overview] - Identifies this resource object as a `Pipeline` object
    - [`name`][kubernetes-overview] - Identifies the `Pipeline` object to be executed

The [example](examples/run-with-pipeline.yaml) below shows a basic `Run`:

```yaml
apiVersion: tekton.dev/v1beta1
kind: Pipeline
metadata:
  name: hello-world
spec:
  tasks:
    - name: echo-hello-world
      taskSpec:
        steps:
          - name: echo
            image: ubuntu
            script: |
              #!/usr/bin/env bash
              echo "Hello World!"
---
apiVersion: tekton.dev/v1alpha1
kind: Run
metadata:
  generateName: piprun-
spec:
  ref:
    apiVersion: tekton.dev/v1beta1
    kind: Pipeline
    name: hello-world
```

### Configuring a `Pipeline` in a `Pipeline`

The `Pipelines` in `Pipelines` `Custom Tasks` can be specified within a `PipelineRun` as shown in this [example](examples/pipelinerun-with-pipeline-in-pipeline.yaml):

```yaml
apiVersion: tekton.dev/v1beta1
kind: Pipeline
metadata:
  name: good-morning-good-afternoon
spec:
  tasks:
    - name: echo-good-morning
      taskSpec:
        steps:
          - name: echo
            image: ubuntu
            script: |
              #!/usr/bin/env bash
              echo "Good Morning!"
    - name: echo-good-afternoon
      taskSpec:
        steps:
          - name: echo
            image: ubuntu
            script: |
              #!/usr/bin/env bash
              echo "Good Afternoon!"
---
apiVersion: tekton.dev/v1beta1
kind: PipelineRun
metadata:
  generateName: pr-
spec:
  serviceAccountName: 'default'
  pipelineSpec:
    tasks:
      - name: hello
        taskSpec:
          steps:
            - name: echo
              image: ubuntu
              script: |
                #!/usr/bin/env bash
                echo "Hello World!"
      - name: greeting
        taskRef:
          apiVersion: tekton.dev/v1beta1
          kind: Pipeline
          name: good-morning-good-afternoon
        runAfter:
          - hello
      - name: bye
        taskSpec:
          steps:
            - name: echo
              image: ubuntu
              script: |
                #!/usr/bin/env bash
                echo "Bye World!"
        runAfter:
          - greeting
```

### Monitoring Execution Status

When the `Run` is executed, it creates a `PipelineRun` to execute the `Pipeline` in the `Pipeline`. Taking the 
[example](examples/pipelinerun-with-pipeline-in-pipeline.yaml) above is executed, two `PipelineRuns` and four 
`TaskRuns` would be created:

```commandline
$ tkn pr list

NAME                      STARTED         DURATION     STATUS
pr-8qcz7-greeting-nfchg   2 minutes ago   7 seconds    Succeeded
pr-8qcz7                  2 minutes ago   52 seconds   Succeeded
```

```commandline
$ tkn tr list

pr-8qcz7-bye-k5s6h                                  2 minutes ago   5 seconds   Succeeded
pr-8qcz7-greeting-nfchg-echo-good-afternoon-65sjp   2 minutes ago   7 seconds   Succeeded
pr-8qcz7-greeting-nfchg-echo-good-morning-srrs8     2 minutes ago   6 seconds   Succeeded
pr-8qcz7-hello-l7z55                                2 minutes ago   6 seconds   Succeeded
```

```commandline
$ tkn pr logs pr-8qcz7

[hello : echo] Hello World!

[bye : echo] Bye World!
```

```commandline
$ tkn pr logs pr-8qcz7-greeting-nfchg 

[echo-good-morning : echo] Good Morning!

[echo-good-afternoon : echo] Good Afternoon!
```

As the `Run` executes, it accumulates information about the overall execution status of the corresponding `PipelineRun`. 
Specifically, the `ConditionSucceeded` status, message and reason of the `Run` would be updated to match that of its `PipelineRun`.

```yaml
$ kubectl describe pipelineruns.tekton.dev pr-8qcz7

Name:         pr-8qcz7
Namespace:    default
Labels:       tekton.dev/pipeline=pr-8qcz7
Annotations:  <none>
API Version:  tekton.dev/v1beta1
Kind:         PipelineRun
Metadata:
  Creation Timestamp:  2021-02-22T03:23:14Z
  Generate Name:       pr-
  # […]  
Spec:
  # […]  
Status:
  Completion Time:  2021-02-22T03:24:06Z
  Conditions:
    Last Transition Time:  2021-02-22T03:24:06Z
    Message:               Tasks Completed: 3 (Failed: 0, Cancelled 0), Skipped: 0
    Reason:                Succeeded
    Status:                True
    Type:                  Succeeded
  Pipeline Spec:
    # […]  
  Runs:
    pr-8qcz7-greeting-nfchg:
      Pipeline Task Name:  greeting
      Status:
        Completion Time:  2021-02-22T03:24:01Z
        Conditions:
          Last Transition Time:  2021-02-22T03:24:01Z
          Message:               Tasks Completed: 2 (Failed: 0, Cancelled 0), Skipped: 0
          Reason:                Succeeded
          Status:                True
          Type:                  Succeeded
        Extra Fields:            <nil>
        Observed Generation:     1
        Start Time:              2021-02-22T03:23:20Z
  Start Time:                    2021-02-22T03:23:14Z
  Task Runs:
    pr-8qcz7-bye-k5s6h:
      Pipeline Task Name:  bye
      Status:
        Completion Time:  2021-02-22T03:24:06Z
        Conditions:
          Last Transition Time:  2021-02-22T03:24:06Z
          Message:               All Steps have completed executing
          Reason:                Succeeded
          Status:                True
          Type:                  Succeeded
        Pod Name:                pr-8qcz7-bye-k5s6h-pod-zsl46
        Start Time:              2021-02-22T03:24:01Z
        Steps:
          Container:  step-echo
          Image ID:   docker-pullable://ubuntu@sha256:0123456789
          Name:       echo
          Terminated:
            Container ID:  docker://0123456789
            Exit Code:     0
            Finished At:   2021-02-22T03:24:06Z
            Reason:        Completed
            Started At:    2021-02-22T03:24:06Z
        Task Spec:
          Steps:
            Image:  ubuntu
            Name:   echo
            Resources:
            Script:  #!/usr/bin/env bash
              echo "Bye World!"
  
    pr-8qcz7-hello-l7z55:
      Pipeline Task Name:  hello
      Status:
        Completion Time:  2021-02-22T03:23:20Z
        Conditions:
          Last Transition Time:  2021-02-22T03:23:20Z
          Message:               All Steps have completed executing
          Reason:                Succeeded
          Status:                True
          Type:                  Succeeded
        Pod Name:                pr-8qcz7-hello-l7z55-pod-qv5f6
        Start Time:              2021-02-22T03:23:14Z
        Steps:
          Container:  step-echo
          Image ID:   docker-pullable://ubuntu@sha256:0123456789
          Name:       echo
          Terminated:
            Container ID:  docker://0123456789
            Exit Code:     0
            Finished At:   2021-02-22T03:23:19Z
            Reason:        Completed
            Started At:    2021-02-22T03:23:19Z
        Task Spec:
          Steps:
            Image:  ubuntu
            Name:   echo
            Resources:
            Script:  #!/usr/bin/env bash
              echo "Hello World!"

Events:
  Type    Reason     Age                From         Message
  ----    ------     ----               ----         -------
  Normal  Started    15m (x2 over 15m)  PipelineRun
  Normal  Running    15m (x2 over 15m)  PipelineRun  Tasks Completed: 0 (Failed: 0, Cancelled 0), Incomplete: 3, Skipped: 0
  Normal  Running    15m                PipelineRun  Tasks Completed: 1 (Failed: 0, Cancelled 0), Incomplete: 2, Skipped: 0
  Normal  Running    15m                PipelineRun  Tasks Completed: 2 (Failed: 0, Cancelled 0), Incomplete: 1, Skipped: 0
  Normal  Succeeded  15m                PipelineRun  Tasks Completed: 3 (Failed: 0, Cancelled 0), Skipped: 0
```

```yaml
$ kubectl describe pipelineruns.tekton.dev pr-8qcz7-greeting-nfchg

Name:         pr-8qcz7-greeting-nfchg
Namespace:    default
Labels:       tekton.dev/pipeline=good-morning-good-afternoon
  tekton.dev/pipelineRun=pr-8qcz7
  tekton.dev/pipelineTask=greeting
  tekton.dev/run=pr-8qcz7-greeting-nfchg
Annotations:  <none>
API Version:  tekton.dev/v1beta1
Kind:         PipelineRun
Metadata:
  Creation Timestamp:  2021-02-22T03:23:20Z
  Generation:          1
  Owner References:
    API Version:           tekton.dev/v1beta1
    Block Owner Deletion:  true
    Controller:            true
    Kind:                  PipelineRun
    Name:                  pr-8qcz7
    UID:                   ea1d7420-42e5-4382
    # […] 
Spec:
  Pipeline Ref:
    API Version:         tekton.dev
    Name:                good-morning-good-afternoon
  Service Account Name:  default
  Timeout:               1h0m0s
Status:
  Completion Time:  2021-02-22T03:23:27Z
  Conditions:
    Last Transition Time:  2021-02-22T03:23:27Z
    Message:               Tasks Completed: 2 (Failed: 0, Cancelled 0), Skipped: 0
    Reason:                Succeeded
    Status:                True
    Type:                  Succeeded
  Pipeline Spec:
    # […] 
  
  Start Time:  2021-02-22T03:23:20Z
  Task Runs:
    pr-8qcz7-greeting-nfchg-echo-good-afternoon-65sjp:
      Pipeline Task Name:  echo-good-afternoon
      Status:
        Completion Time:  2021-02-22T03:23:27Z
        Conditions:
          Last Transition Time:  2021-02-22T03:23:27Z
          Message:               All Steps have completed executing
          Reason:                Succeeded
          Status:                True
          Type:                  Succeeded
        Pod Name:                pr-8qcz7-greeting-nfchg-echo-good-afternoon-65sjp-pod-pggk7
        Start Time:              2021-02-22T03:23:20Z
        Steps:
          Container:  step-echo
          Image ID:   docker-pullable://ubuntu@sha256:0123456789
          Name:       echo
          Terminated:
            Container ID:  docker://0123456789
            Exit Code:     0
            Finished At:   2021-02-22T03:23:26Z
            Reason:        Completed
            Started At:    2021-02-22T03:23:26Z
        Task Spec:
          Steps:
            Image:  ubuntu
            Name:   echo
            Resources:
            Script:  #!/usr/bin/env bash
            echo "Good Afternoon!"
  
    pr-8qcz7-greeting-nfchg-echo-good-morning-srrs8:
      Pipeline Task Name:  echo-good-morning
      Status:
        Completion Time:  2021-02-22T03:23:26Z
        Conditions:
          Last Transition Time:  2021-02-22T03:23:26Z
          Message:               All Steps have completed executing
          Reason:                Succeeded
          Status:                True
          Type:                  Succeeded
        Pod Name:                pr-8qcz7-greeting-nfchg-echo-good-morning-srrs8-pod-rmskq
        Start Time:              2021-02-22T03:23:20Z
        Steps:
          Container:  step-echo
          Image ID:   docker-pullable://ubuntu@sha256:0123456789
          Name:       echo
          Terminated:
            Container ID:  docker://0123456789
            Exit Code:     0
            Finished At:   2021-02-22T03:23:25Z
            Reason:        Completed
            Started At:    2021-02-22T03:23:25Z
        Task Spec:
          Steps:
            Image:  ubuntu
            Name:   echo
            Resources:
            Script:  #!/usr/bin/env bash
            echo "Good Morning!"

Events:
  Type    Reason     Age   From         Message
  ----    ------     ----  ----         -------
  Normal  Started    18m   PipelineRun
  Normal  Running    18m   PipelineRun  Tasks Completed: 0 (Failed: 0, Cancelled 0), Incomplete: 2, Skipped: 0
  Normal  Running    18m   PipelineRun  Tasks Completed: 1 (Failed: 0, Cancelled 0), Incomplete: 1, Skipped: 0
  Normal  Succeeded  18m   PipelineRun  Tasks Completed: 2 (Failed: 0, Cancelled 0), Skipped: 0
```

## Uninstall

```
ko delete -f config/
```

This will delete the `Pipelines-In-Pipelines Controller` and related resources on your cluster.

## Contributions

Read an overview of our processes in [Tekton Community](https://github.com/tektoncd/community).

[kubernetes-overview]:
https://kubernetes.io/docs/concepts/overview/working-with-objects/kubernetes-objects/#required-fields