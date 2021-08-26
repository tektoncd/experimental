
# Common Expression Language

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

This is an experimental project that provides support for Common Expression Language (CEL) in Tekton Pipelines.
The functionality is provided by a controller that implements the Custom Task interface. Its use cases include 
evaluating complex expressions to be used in [`when` expressions][when-expressions] in subsequent `Tasks` to guard 
their execution. 

**Note**: As an experimental project, the syntax and functionality may change at any time. We hope to promote this to a
top level feature with stability guarantees, but we'll gather feedback and re-examine the design before then. 

- [Install](#install)
- [Usage](#usage)
  - [Configuring a `CELEval`](#configuring-a-celeval)
  - [Configuring a `CELEval` in a `Pipeline`](#configuring-a-celeval-in-a-pipeline)
  - [Configuring a `CELEval` in a `PipelineRun`](#configuring-a-celeval-in-a-pipelinerun)
  - [Specifying CEL expressions](#specifying-cel-expressions)
  - [Specifying CEL environment variables](#specifying-cel-environment-variables)
  - [Monitoring execution status](#monitoring-execution-status)
  - [Using the evaluation results](#using-the-evaluation-results)
- [Uninstall](#uninstall)
- [Contributions](#contributions)

## Install

Install and configure [`ko`][ko].

```
ko apply -f config/
```

This will build and install the `CELEval Controller` on your cluster, in the namespace `tekton-pipelines`. 

```commandline
$ k get pods -n tekton-pipelines

NAME                                  READY   STATUS    RESTARTS   AGE
celeval-controller-654bdc4cc8-7bvvn   1/1     Running   0          3m4s
```

Alternatively, install it from the nightly release using:

```commandline
kubectl apply --filename https://storage.cloud.google.com/tekton-releases-nightly/celeval/latest/release.yaml
```

## Usage

To evaluate a CEL expressions using `Custom Tasks`, we need to define a [`Run`][run] type with 
`apiVersion: custom.tekton.dev/v1alpha1` and `kind: CELEval`. 

The `Run` takes the CEL expressions to be evaluated through `expressions` field. The `Run` optionally takes CEL 
environment variables through the `variables` field.

If executed successfully, the `Run` will produce the evaluation results as `Results` with names corresponding
with the `Expressions`'s names. See the [examples](examples) folder for `CELEvals` to run or use as samples. 

### Configuring a `CELEval`

A `CELEval` definition supports the following fields:

- Required:
  - [`apiVersion`][kubernetes-overview] - Specifies the API version, `custom.tekton.dev/v1alpha1`.
  - [`kind`][kubernetes-overview] - Identifies this resource object as a `CELEval` object.
  - [`metadata`][kubernetes-overview] - Specifies the metadata that uniquely identifies the `CELEval`, such as a `name`.
  - [`spec`][kubernetes-overview] - Specifies the configuration for the `CELEval`.
    - [`expressions`](#specifying-cel-expressions) - Specifies the CEL expressions to be evaluated
    - [`variables`](#specifying-cel-environment-variables) - (optional) Specifies the CEL environment variables 

The example below shows a basic `CELEval`:

```yaml
apiVersion: custom.tekton.dev/v1alpha1
kind: CELEval
metadata:
  name: get-type
spec:
  expressions:
    - name: expression
      value: "type(1)"
```

The example below shows a basic `Run`:

```yaml
apiVersion: tekton.dev/v1alpha1
kind: Run
metadata:
  generateName: get-type-
spec:
  ref:
    apiVersion: custom.tekton.dev/v1alpha1
    kind: CELEval
    name: get-type
```

### Configuring a `CELEval` in a `Pipeline`

A `CELEval` can be specified within a `Pipeline`, as such:

```yaml
apiVersion: custom.tekton.dev/v1alpha1
kind: CELEval
metadata:
  name: get-red
spec:
  expressions:
    - name: expression
      value: "{'blue': '0x000080', 'red': '0xFF0000'}['red']"
---
apiVersion: tekton.dev/v1beta1
kind: Pipeline
metadata:
  generateName: pipeline-get-red-
spec:
  tasks:
    - name: get-red
      taskRef:
        apiVersion: custom.tekton.dev/v1alpha1
        kind: CELEval
        name: get-red
```
### Configuring a `CELEval` in a `PipelineRun`

A `CELEval` can be specified within a `PipelineRun`, as such:

```yaml
apiVersion: custom.tekton.dev/v1alpha1
kind: CELEval
metadata:
  name: get-blue
spec:
  expressions:
    - name: expression
      value: "{'blue': '0x000080', 'red': '0xFF0000'}['blue']"
---
apiVersion: tekton.dev/v1beta1
kind: PipelineRun
metadata:
  generateName: pipelinerun-get-blue-
spec:
  pipelineSpec:
    tasks:
      - name: get-blue
        taskRef:
          apiVersion: custom.tekton.dev/v1alpha1
          kind: CELEval
          name: get-blue 
```

### Specifying CEL expressions

The CEL expressions to be evaluated by the `CELEval` are specified in `expressions` field, which are made up of `name` 
and `value` pairs which are `Strings`.

```yaml
apiVersion: custom.tekton.dev/v1alpha1
kind: CELEval
metadata:
  name: get-type
spec:
  expressions:
    - name: type
      value: "type(1)"
---
apiVersion: tekton.dev/v1beta1
kind: PipelineRun
metadata:
  generateName: pipeline-run-get-type-
spec:
  serviceAccountName: 'default'
  pipelineSpec:
    tasks:
      - name: get-type
        taskRef:
          apiVersion: custom.tekton.dev/v1alpha1
          kind: CELEval
          name: get-type
      - name: echo-get-type
        when:
          - input: "$(tasks.get-type.results.type)"
            operator: in
            values: ["int"]
        taskSpec:
          steps:
            - name: echo
              image: ubuntu
              script: echo ISINT!
```

For more information about `Parameters`, read [specifying `Parameters`][specifying-parameters].

### Specifying CEL environment variables

For each execution of a `CELEval`, we create an evaluation environment. 
As described in [CEL Evaluation Documentation][cel-evaluation], context variables can be bound to the environment. 

The CEL variables to be declared in the environment before evaluation by `CELEval` are specified in `variables` field, 
which are made up of `name` and `value` pairs which are `Strings`.

```yaml
apiVersion: custom.tekton.dev/v1alpha1
kind: CELEval
metadata:
  name: get-sev
spec:
  variables:
    - name: priority
      value: "high"
    - name: alert_enable
      value: "true"
  expressions:
    - name: severity
      value: "priority in ['high', 'normal'] ? 'sev-1' : 'sev-2'"
---
apiVersion: tekton.dev/v1beta1
kind: PipelineRun
metadata:
  generateName: pipeline-run-get-sev-
spec:
  serviceAccountName: 'default'
  pipelineSpec:
    tasks:
      - name: get-sev
        taskRef:
          apiVersion: custom.tekton.dev/v1alpha1
          kind: CELEval
          name: get-sev
      - name: is-severe
        when:
          - input: "$(tasks.get-sev.results.severity)"
            operator: in
            values: ["true"]
        taskSpec:
          steps:
            - name: echo
              image: ubuntu
              script: echo SEVERE!
```

For more information about `Parameters`, read [specifying `Parameters`][specifying-parameters]. For more information
about CEL environment, read [environment setup][cel-docs].

### Monitoring execution status

As the `Run` executes, its `status` field accumulates information about the execution status of the `Run` in general.

If the evaluation is successful, it will also contain the `Results` of the evaluation under `status.results` with the
corresponding names of the CEL expressions as provided in the `expressions`.

```yaml
Name:         colors-xgtb9
Namespace:    default
Labels:       custom.tekton.dev/celeval=colors
Annotations:  <none>
API Version:  tekton.dev/v1alpha1
Kind:         Run
Metadata:
  Creation Timestamp:  2021-08-26T05:03:24Z
  Generate Name:       colors-
# […]
Spec:
  Ref:
    API Version:         custom.tekton.dev/v1alpha1
    Kind:                CELEval
    Name:                colors
  Service Account Name:  default
Status:
  Completion Time:  2021-08-26T05:03:24Z
  Conditions:
    Last Transition Time:  2021-08-26T05:03:24Z
    Message:               CEL expressions were evaluated successfully
    Reason:                EvaluationSuccess
    Status:                True
    Type:                  Succeeded
  Extra Fields:
    Results:
      Name:   red
      Value:  0xFF0000
      Name:   blue
      Value:  0x000080
      Name:   is-red
      Value:  true
      Name:   is-blue
      Value:  false
    Spec:
      Expressions:
        Name:           red
        Value:          {'blue': '0x000080', 'red': '0xFF0000'}['red']
        Name:           blue
        Value:          {'blue': '0x000080', 'red': '0xFF0000'}['blue']
        Name:           is-red
        Value:          {'blue': '0x000080', 'red': '0xFF0000'}['red'] == '0xFF0000'
        Name:           is-blue
        Value:          {'blue': '0x000080', 'red': '0xFF0000'}['blue'] == '0xFF0000'
  Observed Generation:  1
  Results:
    Name:      red
    Value:     0xFF0000
    Name:      blue
    Value:     0x000080
    Name:      is-red
    Value:     true
    Name:      is-blue
    Value:     false
  Start Time:  2021-08-26T05:03:24Z
Events:
  Type    Reason     Age                From      Message
  ----    ------     ----               ----      -------
  Normal  Started    18s (x2 over 18s)  CELEval
  Normal  Succeeded  18s (x2 over 18s)  CELEval   CEL expressions were evaluated successfully
```

If no CEL expressions are provided, any CEL expression is invalid or there's any other error, the `Run` will fail and 
the details will be included in `status.conditions` as such:

```yaml
Name:         colors-f8n9t
Namespace:    default
Labels:       custom.tekton.dev/celeval=colors
Annotations:  <none>
API Version:  tekton.dev/v1alpha1
Kind:         Run
Metadata:
  Creation Timestamp:  2021-08-26T05:06:34Z
  Generate Name:       colors-
# […]
Spec:
  Ref:
    API Version:         custom.tekton.dev/v1alpha1
    Kind:                CELEval
    Name:                colors
  Service Account Name:  default
Status:
  Completion Time:  2021-08-26T05:06:34Z
  Conditions:
    Last Transition Time:  2021-08-26T05:06:34Z
    Message:               Run can't be run because it has an invalid spec - missing field(s): expressions
    Reason:                RunValidationFailed
    Status:                False
    Type:                  Succeeded
  Extra Fields:
    Spec:
      Expressions:      <nil>
  Observed Generation:  1
  Start Time:           2021-08-26T05:06:34Z
Events:
  Type     Reason   Age   From      Message
  ----     ------   ----  ----      -------
  Normal   Started  14s   CELEval
  Warning  Failed   14s   CELEval   Run can't be run because it has an invalid spec - missing field(s): expressions
```

For general information about monitoring a `Run`, read [monitoring execution status][monitoring-run-execution-status].

### Using the evaluation results

A successful `Run` contains the `Results` of evaluating the CEL expressions under `status.results`, with the name of
each evaluation `Result` matching the name of the corresponding CEL expression as provided in `expressions`.

Users can reference the `Results` in subsequent `Tasks` using variable substitution, as such:

```yaml
apiVersion: custom.tekton.dev/v1alpha1
kind: CELEval
metadata:
  name: get-type
spec:
  expressions:
    - name: type
      value: "type(1)"
---
apiVersion: tekton.dev/v1beta1
kind: PipelineRun
metadata:
  generateName: pipeline-run-get-type-
spec:
  serviceAccountName: 'default'
  pipelineSpec:
    tasks:
      - name: get-type
        taskRef:
          apiVersion: custom.tekton.dev/v1alpha1
          kind: CELEval
          name: get-type
      - name: echo-get-type
        when:
          - input: "$(tasks.get-type.results.type)"
            operator: in
            values: ["int"]
        taskSpec:
          steps:
            - name: echo
              image: ubuntu
              script: echo ISINT!
```

For more information about using `Results`, read [using results][using-results].


## Uninstall 

```
ko delete -f config/
```

This will delete the `CELEval Controller` and related resources on your cluster.

## Contributions

Read an overview of our processes in [Tekton Community](https://github.com/tektoncd/community).

[ko]: https://github.com/google/ko
[run]: https://github.com/tektoncd/pipeline/blob/main/docs/runs.md
[kubernetes-overview]: https://kubernetes.io/docs/concepts/overview/working-with-objects/kubernetes-objects/#required-fields
[when-expressions]: https://github.com/tektoncd/pipeline/blob/main/docs/pipelines.md#guard-task-execution-using-when-expressions
[specifying-parameters]: https://github.com/tektoncd/pipeline/blob/master/docs/pipelines.md#specifying-parameters
[monitoring-run-execution-status]: https://github.com/tektoncd/pipeline/blob/main/docs/runs.md#monitoring-execution-status
[using-results]: https://github.com/tektoncd/pipeline/blob/main/docs/pipelines.md#using-results
[cel-docs]: https://github.com/google/cel-go#environment-setup
[cel-evaluation]: https://github.com/google/cel-spec/blob/master/doc/langdef.md#evaluation