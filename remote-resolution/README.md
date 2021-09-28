# Remote Resolution Experimental Controller

This directory contains a proof-of-concept controller for testing
alternative implementations of [Remote
Resolution](https://github.com/tektoncd/community/blob/main/teps/0060-remote-resource-resolution.md).

## How to Try

1. Ensure [Pipelines](https://github.com/tektoncd/pipeline) and
   [Triggers](https://github.com/tektoncd/triggers) are both installed
   in your Kubernetes cluster.
2. Run `ko apply -f ./config` from the directory this readme is in.

Once that's done a new namespace will be created in your cluster called
`tekton-remote-resolution` and there will be controller + webhook
deployments running inside it.

To test it out try applying a PipelineRun using git resolution:

```bash
kubectl apply -f ./remote-resolution/pr-git.yaml
kubectl get pipelineruns -w
```

If everything's working as expected you should see this PR start in a
Pending state and then resolve, run and succeed.

## Resolution Modes

This proof-of-concept implements two of [the alternatives from
TEP-0060](https://github.com/tektoncd/community/blob/main/teps/0060-remote-resource-resolution.md#alternatives):

- A `ResourceRequest` CRD.
- A `ClusterInterceptor`-based HTTP interface.

Both of these modes support fetching `Pipelines` from `git` or `in-cluster`.

### Switching mode

By default the controller starts in `ResourceRequest` mode. It can be
switched into `ClusterInterceptor` mode by setting an environment
variable.

To change the mode open [./config/controller.yaml](./config/controller.yaml)
and set the `RESOLUTION_MODE` environment variable to `"rr"` for
`ResourceRequests` or `"ci"` for `ClusterInterceptors`.

## Resolver Framework

Also included here is a hokey attempt at a "Resolver Framework". At this
early stage it's just a golang interface that both `git` and
`in-cluster` resolvers implement, which makes it easy to support both
the `ResourceRequest` and `ClusterInterceptor` alternatives without
a large duplication of logic. The eventual intention is that writing a
basic resolver should only require a few dozen lines of Go and should be
independent of the "protocol" that's being used.

The interface is defined in
[./pkg/reconciler/framework/interface.go](./pkg/reconciler/framework/interface.go)
and the three included resolvers demonstrate how it can be implemented
for different use-cases:

[Git Resolver](./pkg/resolvers/gitref/resolver.go).
[ClusterRef Resolver](./pkg/resolvers/clusterref/resolver.go).
[No-Op Resolver](./pkg/resolvers/noopref/resolver.go).
