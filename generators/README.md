# Tekton Generators

This project contains experimental code to create a tool for generating Tekton spec from simplified configs. The goal is to help users bootstrap pipelines in a configurable way.

See [tektoncd/pipeline/#2590](https://github.com/tektoncd/pipeline/issues/2590) information and background.

## GitHub Token
You would expect to have a [GitHub Personal Access Token](https://docs.github.com/en/github/authenticating-to-github/creating-a-personal-access-token#creating-a-token) set in the kubernetes secret with a GitHub token like this :
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: github-secret
  type: Opaque
stringData:
  secretToken: "YOUR-GITHUB-ACCESS-TOKEN"
```
Then you can create it on the command line with `kubectl` like this :
```
kubectl apply -f secret.yaml
```

## Dependent Tasks
The generated config would expect to use the tasks already on the cluster adding to the pipeline. The tasks include [`git-clone`](https://github.com/tektoncd/catalog/blob/master/git/git-clone.yaml) and [`github-set-status`](https://github.com/tektoncd/catalog/blob/master/github/set_status.yaml). The `git-clone` task is used to help clone a repo into the workspace. The `status-task` is used to help allow external services to mark GitHub commits with a state. 
**Please Note: this git-clone Task is only able to fetch code from the public repo for the time being.**
### Install the Task
You can install the tasks with the specified revision(commit SHA) on the command line like this:
```
kubectl apply -f https://raw.githubusercontent.com/tektoncd/catalog/<revision>/github/set_status.yaml
kubectl apply -f https://raw.githubusercontent.com/tektoncd/catalog/<revision>/git/git-clone.yaml
```

## Features

This experimental project has been broken down into the features as follows:

1. Parse the yaml file with io.Reader and store the result in the self-defined struct
2. Create tool that given an input spec with steps, generates the resulting Tekton resources for the particular type.
- Create binary that invokes another binary on the path based on the type.
- Read input steps and generate resulting pipeline config that mounts GitHub workspace and configures steps accordingly.
3. Add support for writing output to disk.
4. Add support for tasks and pipelines.
5. Add support for applying config to cluster.
6. Add support for deleting configs from cluster.
