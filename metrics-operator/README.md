# Tekton Metrics Operator

## Installation
```
kustomize build config | ko apply --local --base-import-paths -f -
```

## Description

This project introduces a new API Group `metrics.tekton.dev`, which has new CRDs
 to enable monitoring of your PipelineRuns and TaskRuns.

### Custom Resource Definitions

#### TaskMonitor

As the name suggests, this CRD is responsible to define and expose metrics for a
 given Task.

```yaml
apiVersion: metrics.tekton.dev/v1alpha1
kind: TaskMonitor
metadata:
  name: hello
spec:
  taskName: hello
  metrics: # for more details on this, see Metrics Definition section
  - name: status
    type: gauge
    by:
    - condition: "Succeeded"
```

#### TaskRunMonitor

Similar to the TaskMonitor, however allows to group a set of TaskRuns
  arbitrarily using selectors

```yaml
apiVersion: metrics.tekton.dev/v1alpha1
kind: TaskRunMonitor
metadata:
  name: repository
spec:
  selector:
    matchExpressions:
    - {key: repository, operator: Exists}
  metrics: # for more details on this, see Metrics Definition section
  - name: status
    type: gauge
    by:
    - condition: "Succeeded"
```

#### PipelineMonitor

This CRD expose the defined metrics for a given Pipeline

```yaml
apiVersion: metrics.tekton.dev/v1alpha1
kind: PipelineMonitor
metadata:
  name: hello
spec:
  pipelineName: hello
  metrics: # for more details on this, see Metrics Definition section
  - name: status
    type: gauge
    by:
    - condition: "Succeeded"
```

#### PipelineRunMonitor

Similar to PipelineMonitor, however this CRD allows to group a set of
 PipelineRuns arbitrarily using selectors

```yaml
apiVersion: metrics.tekton.dev/v1alpha1
kind: PipelineRunMonitor
metadata:
  name: hello
spec:
  selector:
    matchExpressions:
    - {key: repository, operator: Exists}
  metrics: # for more details on this, see Metrics Definition section
  - name: status
    type: gauge
    by:
    - condition: "Succeeded"
```

### Metrics Definition

All Monitor-like CRDs have a list of metric definition as part of the
 specification. This abtraction allow us to configure metrics for any set of
 resources in a efficient way.

Currently, there are three types supported: counter, gauge and histogram.

#### Counter

As the name suggests, this is a simple count of task or pipeline runs executed.
This metric is only updated after the run finishes. You can configure the
dimensions you want to count using the `by` field.

Counter metrics always start as 0 and only go up. This way you can use functions 
 like `rate` and `increase` to get the amount over a period.

For example, let's say you want to know the error rate for each environment and
service name.

```yaml
- name: status
  type: counter
  by:
  - condition: Succeeded
  - param: environment
  - label: your.label/service-name
```

This will expose a new metric `metric_operator_controller_hello_status_total`
with the given label segementation.

The counter metric name convention follows `metric_operator_controller_{{MonitorName}}_{{MetricName}}_total`

#### Gauge

Gauge metrics can go up and down, and given this nature this metric is updated
even during the execution. You can also configure the dimensions you want using 
the `by` field.

You can also configure custom `match` to filter what you want to gauge.

```yaml
name: done
type: gauge
match:
  key:
    condition: Succeeded
  operator: In
  values: [success, failed]
by:
  - param: environment
```

This will expose a new metric `metrics_operator_controller_hello_done`

The gauge metric name convention follows
`metric_operator_controller_{{MonitorName}}_{{MetricName}}`. Note that this is
the only metric type that doesn't have suffix in its name conversion.

#### Histogram

Histogram metrics expose a set of metrics that allow you to analyze the data
further, extracting statistics like percentiles and ranks. This metric is only
reported after the task or pipeline run is done.

You can configure what you want to measure with the `duration` field.
This allows you to get the full task or pipeline duration, but also get a
specific step or set of steps. You can also use that to get schedule delays.

```yaml
name: completion_time
type: histogram
duration:
  from: .status.startTime
  to: .status.completionTime
by:
- label: priority
```

The histogram metric name convention follows
`metric_operator_controller_{{MonitorName}}_{{MetricName}}_seconds`.
Prometheus will add the suffixes `_bucket`, `_sum` and `_count` on top of it.
