# Cloudevents Controller for Tekton

...

### Motivation

Send Cloudevents based on Tekton Resource Lifecycle.

## Install

Install and configure `ko`.

```
ko apply -f config/
```

This will build and install the controller on your cluster, in the namespace
`tekton-cloudevents`.

## Uninstall

```
$ kubectl delete namespace tekton-cloudevents
namespace "tekton-cloudevents" deleted
```

This will stop the controller and delete its namespace.
