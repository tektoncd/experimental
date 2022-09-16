This experimental controller allows defining concurrency controls for PipelineRuns.

An example concurrency control is as follows:

```yaml
apiVersion: tekton.dev/v1alpha1
kind: ConcurrencyControl
metadata:
  name: cc
  namespace: my-namespace
spec:
  params:
  - name: pull-request-id
  key: $(params.pull-request-id)
  strategy: Cancel
  selector:
    matchLabels:
      tekton.dev/pipeline: ci-pipeline
```

The ConcurrencyControl parameters must match the parameters of the PipelineRun.
The parameters in the concurrency key will be substituted with the parameter values from the PipelineRun.
Any PipelineRuns in the same namespace as the ConcurrencyControl matching the label selector and with the
same value for the concurrency key (after parameter substitution) are considered part of the same concurrency group.
The only currently supported strategy is "Cancel", meaning that when a new PipelineRun is created, any PipelineRuns in the
same concurrency group will be canceled.

If a PipelineRun is part of multiple concurrency groups (i.e. its labels match the label selectors of multiple concurrency controls),
PipelineRuns in all of its concurrency groups will be canceled when it is created.

To avoid allowing the new PipelineRun to start before the old PipelineRun has been canceled, PipelineRuns should be created as pending.
TODO(Lee): explore admission webhook to create all PRs as pending.