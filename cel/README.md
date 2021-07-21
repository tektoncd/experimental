
# Common Expression Language

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

This is an experimental project that provides support for Common Expression Language (CEL) in Tekton Pipelines.
The functionality is provided by a controller that implements the Custom Task interface. Its use cases include 
evaluating complex expressions to be used in [`WhenExpressions`](https://github.com/tektoncd/pipeline/blob/master/docs/pipelines.md#guard-task-execution-using-whenexpressions)
in subsequent `Tasks` to guard their execution. 

- [Install](#install)
- [Usage](#usage)
- [Uninstall](#uninstall)

## Install

Install and configure [`ko`](https://github.com/google/ko).

```
ko apply -f config/
```

This will build and install the `CEL Controller` on your cluster, in the namespace `tekton-cel-run`. 

```commandline
$ k get pods -n tekton-cel-run 

NAME                              READY   STATUS    RESTARTS   AGE
cel-controller-654bdc4cc8-7bvvn   1/1     Running   0          3m4s
```

Alternatively, install it from the nightly release using:

```commandline
kubectl apply --filename https://storage.cloud.google.com/tekton-releases-nightly/cel/latest/release.yaml
```

## Usage

To evaluate a CEL expressions using `Custom Tasks`, we need to define a [`Run`](https://github.com/tektoncd/pipeline/blob/master/docs/runs.md)
type with `apiVersion: cel.tekton.dev/v1alpha1` and `kind: CEL`. The `Run` takes the CEL expressions to be evaluated
as `Parameters`. If executed successfully, the `Run` will produce the evaluation results as `Results` with names corresponding
with the `Parameters` names. See the [examples](examples) folder for `CEL` `Custom Tasks` to run or use as samples. 

### Configuring a `CEL` `Custom Task`

The `CEL` `Custom Task` is defined in a `Run`, which supports the following fields:

- [`apiVersion`][kubernetes-overview] - Specifies the API version, `tekton.dev/v1alpha1`
- [`kind`][kubernetes-overview] - Identifies this resource object as a `Run` object
- [`metadata`][kubernetes-overview] - Specifies the metadata that uniquely identifies the `Run`, such as a `name`
- [`spec`][kubernetes-overview] - Specifies the configuration for the `Run`
- [`ref`][kubernetes-overview] - Specifies the `CEL` `Custom Task`
  - [`apiVersion`][kubernetes-overview] - Specifies the API version, `cel.tekton.dev/v1alpha1`
  - [`kind`][kubernetes-overview] - Identifies this resource object as a `CEL` object
- [`params`](#specifying-cel-expressions) - Specifies the CEL expressions to be evaluated as parameters

The example below shows a basic `Run`:

```yaml
apiVersion: tekton.dev/v1alpha1
kind: Run
metadata:
  generateName: celrun-
spec:
  ref:
    apiVersion: cel.tekton.dev/v1alpha1
    kind: CEL
  params:
  - name: expression
    value: "type(1)"
```

### Configuring a `CEL` `Custom Task` in a `Pipeline`

The `CEL` `Custom Task` can be specified within a `Pipeline`, as such:

```yaml
apiVersion: tekton.dev/v1beta1
kind: Pipeline
metadata:
  generateName: pipeline-
spec:
  tasks:
    - name: get-red
      taskRef:
        apiVersion: cel.tekton.dev/v1alpha1
        kind: CEL
      params:
        - name: red
          value: "{'blue': '0x000080', 'red': '0xFF0000'}['red']"
```
### Configuring a `CEL` `Custom Task` in a `PipelineRun`

The `CEL` `Custom Task` can be specified within a `PipelineRun`, as such:

```yaml
apiVersion: tekton.dev/v1beta1
kind: PipelineRun
metadata:
  generateName: pipelinerun-
spec:
  pipelineSpec:
    tasks:
      - name: get-blue
        taskRef:
          apiVersion: cel.tekton.dev/v1alpha1
          kind: CEL
        params:
          - name: blue
            value: "{'blue': '0x000080', 'red': '0xFF0000'}['blue']"
```

### Specifying CEL expressions

The CEL expressions to be evaluated by the `Run` are specified using parameters. The parameters can be specified
in the `Run` directly or be passed through from a `Pipeline` or `PipelineRun`, as such:

```yaml
apiVersion: tekton.dev/v1beta1
kind: PipelineRun
metadata:
  generateName: pipelinerun-
spec:
  pipelineSpec:
    params:
    - name: is-red-expr
      type: string
    tasks:
      - name: is-red
        taskRef:
          apiVersion: cel.tekton.dev/v1alpha1
          kind: CEL
        params:
          - name: is-red-expr
            value: "$(params.is-red-expr)"
  params:
    - name: is-red-expr
      value: "{'blue': '0x000080', 'red': '0xFF0000'}['red'] == '0xFF0000'"
```

For more information about specifying `Parameters`, read [specifying parameters](https://github.com/tektoncd/pipeline/blob/master/docs/pipelines.md#specifying-parameters).

### Monitoring execution status

As the `Run` executes, its `status` field accumulates information about the execution status of the `Run` in general.

If the evaluation is successful, it will also contain the `Results` of the evaluation under `status.results` with the
corresponding names of the CEL expressions as provided in the `Parameters`.

```yaml
Name:         celrun-is-red-8lbwv
Namespace:    default
API Version:  tekton.dev/v1alpha1
Kind:         Run
Metadata:
  Creation Timestamp:  2021-01-20T17:51:52Z
  Generate Name:       celrun-is-red-
# […]
Spec:
  Params:
    Name:   red
    Value:  {'blue': '0x000080', 'red': '0xFF0000'}['red']
    Name:   is-red
    Value:  {'blue': '0x000080', 'red': '0xFF0000'}['red'] == '0xFF0000'
  Ref:
    API Version:         cel.tekton.dev/v1alpha1
    Kind:                CEL
  Service Account Name:  default
Status:
  Completion Time:  2021-01-20T17:51:52Z
  Conditions:
    Last Transition Time:  2021-01-20T17:51:52Z
    Message:               CEL expressions were evaluated successfully
    Reason:                EvaluationSuccess
    Status:                True
    Type:                  Succeeded
  Extra Fields:            <nil>
  Observed Generation:     1
  Results:
    Name:      red
    Value:     0xFF0000
    Name:      is-red
    Value:     true
  Start Time:  2021-01-20T17:51:52Z
Events:
  Type    Reason         Age   From            Message
  ----    ------         ----  ----            -------
  Normal  RunReconciled  13s   cel-controller  Run reconciled: "default/celrun-is-red-8lbwv"
```

If no CEL expressions are provided, any CEL expression is invalid or there's any other error, the `CEL` `Custom Task`
will fail and the details will be included in `status.conditions` as such:

```yaml
Name:         celrun-is-red-4ttr8
Namespace:    default
API Version:  tekton.dev/v1alpha1
Kind:         Run
Metadata:
  Creation Timestamp:  2021-01-20T17:58:53Z
  Generate Name:       celrun-is-red-
# […]
Spec:
  Ref:
    API Version:         cel.tekton.dev/v1alpha1
    Kind:                CEL
  Service Account Name:  default
Status:
  Completion Time:  2021-01-20T17:58:53Z
  Conditions:
    Last Transition Time:  2021-01-20T17:58:53Z
    Message:               Run can't be run because it has an invalid spec - missing field(s) params
    Reason:                RunValidationFailed
    Status:                False
    Type:                  Succeeded
  Extra Fields:            <nil>
  Observed Generation:     1
  Start Time:              2021-01-20T17:58:53Z
Events:                    <none>
```

For more information about monitoring `Run` in general, read [monitoring execution status](https://github.com/tektoncd/pipeline/blob/master/docs/runs.md#monitoring-execution-status).

### Using the evaluation results

A successful `Run` contains the `Results` of evaluating the CEL expressions under `status.results`, with the name of
each evaluation `Result` matching the name of the corresponding CEL expression as provided in the `Parameters`.
Users can reference the `Results` in subsequent `Tasks` using variable substitution, as such:

```yaml
apiVersion: tekton.dev/v1beta1
kind: PipelineRun
metadata:
  generateName: pipelinerun-
spec:
  pipelineSpec:
    params:
      - name: is-red-expr
        type: string
    tasks:
      - name: is-red
        taskRef:
          apiVersion: cel.tekton.dev/v1alpha1
          kind: CEL
        params:
          - name: is-red-expr
            value: "$(params.is-red-expr)"
      - name: echo-is-red
        when:
          - input: "$(tasks.is-red.results.is-red-expr)"
            operator: in
            values: ["true"]
        taskSpec:
          steps:
            - name: echo
              image: ubuntu
              script: echo RED!
  params:
    - name: is-red-expr
      value: "{'blue': '0x000080', 'red': '0xFF0000'}['red'] == '0xFF0000'"
```

For more information about using `Results`, read [using results](https://github.com/tektoncd/pipeline/blob/master/docs/pipelines.md#using-results).


## Uninstall 

```
ko delete -f config/
```

This will delete the `CEL Controller` and related resources on your cluster.

## Contributions

Read an overview of our processes in [Tekton Community](https://github.com/tektoncd/community).

[kubernetes-overview]:
https://kubernetes.io/docs/concepts/overview/working-with-objects/kubernetes-objects/#required-fields
