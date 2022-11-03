# Tekton Workflows

Workflows is an experimental concept for grouping other Tekton primitives(such as Pipelines and Triggers) in order to simplify CI/CD configuration. 
It is an experimental project that may change in breaking ways at any time.

See [TEP-0098: Workflows](https://github.com/tektoncd/community/blob/main/teps/0098-workflows.md) for more information.
This project is discussed in the [Workflows Working Group](https://github.com/tektoncd/community/blob/main/working-groups.md#workflows).

## Installation

You must first install Tekton Pipelines and Tekton Triggers.

### Install from nightly release

```
kubectl apply --filename https://storage.googleapis.com/tekton-releases-nightly/workflows/latest/release.yaml
```

### Build and install from source

```
ko apply -f config
```

## Usage

Each Workflow creates Triggers in its own namespace.
To restrict [the Workflows EventListener](./config/workflows-el.yaml) to only be able to access the Triggers
in certain namespaces, edit its namespace selector.

## Future work
- Support for connecting to GitHub repos
- Support for declaring secrets in a Workflow
- Improved syntax for Workspaces and volumes
