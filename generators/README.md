# Tekton Generators

This project contains experimental code to create a tool for generating Tekton spec from simplified configs. The goal is to help users bootstrap pipelines in a configurable way.

See [tektoncd/pipeline/#2590](https://github.com/tektoncd/pipeline/issues/2590) information and background.

## GitHub token
You will need to create a [GitHub Personal Access Token](https://docs.github.com/en/github/authenticating-to-github/creating-a-personal-access-token#creating-a-token) and set it in the Kubernetes scecret `github` in the key `token` like this:
```
kubectl create secret generic github --from-literal token="YOUR_GITHUB_PERSONAL_ACCESS_TOKEN"
```
## Webhook secret
You would expect to create a [Webhook secret token](https://developer.github.com/webhooks/securing/#setting-your-secret-token) and configure the GitHub webhook to use this value.
You can contain this value in the Kubernetes secret like this :
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: webhook-secret
type: Opaque
stringData:
  secretToken: "YOUR-WEBHOOK-SECRET-TOKEN"
```
Then you can create it on the command line with `kubectl` like this :
```
kubectl apply -f secret.yaml
```
Now it can be passed as a reference to the GitHub interceptor.
## Service Account
The [`serviceAccountName`](https://github.com/tektoncd/triggers/blob/master/docs/eventlisteners.md#serviceAccountName) is a required field for Tekton Triggers.  You need to provide it in the input GitHub config settings. You can create the service account with the [webhook secret](#webhook-secret) that is used in the Kubernetes cluster like this :
```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: tekton-generators-github-sa
secrets:
  - name: webhook-secret
```
With the service account, you also need to create a role to follow certain roles and the role binding. See the [example](https://github.com/tektoncd/triggers/tree/master/examples/role-resources/triggerbinding-roles) to create your own.
Use `kubectl apply` to create these resources on the cluster as well.
**Please Note: If you use Kaniko in the task, you need to make the service account have enough credentials to push images.  On GKE, it has been configured already. You can ignore this.**

## Dependent Tasks
The generated config would expect to use the tasks already on the cluster adding to the pipeline. The tasks include [`git-clone`](https://github.com/tektoncd/catalog/blob/v1beta1/git/git-clone.yaml) and [`github-set-status`](https://github.com/tektoncd/catalog/blob/v1beta1/github/set_status.yaml). The `git-clone` task is used to help clone a repo into the workspace. The `status-task` is used to help allow external services to mark GitHub commits with a state. 
**Please Note: this git-clone Task is only able to fetch code from the public repo for the time being.**
### Install the Task
You can install the tasks with the specified revision(commit SHA) on the command line like this:
```
kubectl apply -f https://raw.githubusercontent.com/tektoncd/catalog/<revision>/github/set_status.yaml
kubectl apply -f https://raw.githubusercontent.com/tektoncd/catalog/<revision>/git/git-clone.yaml
```

## Input configurations
Here is the example of the GitHub input config:
```yaml
kind: GitHub
metadata:
  name: github-build
spec:
  url: "https://github.com/YolandaDu1997/hello-world"
  branch: "master"
  storage: 1Gi
  secretName: webhook-secret
  secretKey: secretToken
  serviceAccountName: tekton-generators-demo
  steps:
    - name: build
      image: gcr.io/kaniko-project/executor:latest
      command:
        - /kaniko/executor
      args:
        - --context=dir://$(workspaces.input.path)/src
        - --destination=gcr.io/<use your project>/kaniko-test
        - --verbosity=debug
```
### Fields

 - **kind**: the kind of generators (*required*)
 - **metadata**: the metadata that uniquely identifies the `GitHub` resource object. For example, a `name` (*required*)
 - **spec**: the GitHub sepc
	 - **url**: GitHub url to clone (*required*)
	 - **revision**: git reversion to clone (*default:* master)
	 - **branch**: the remote branch where to trigger the pipelinerun (*default:* master)
	 - **storage**: the disk storage needed in the workspace (*default:* 1Gi)
	 - **secretName**: the [webhook secret](#webhook-secret) name (*required* when generating Triggers)
	 - **secretKey**: the [secret key](#webhook-secret) of token in the webhook secret (*required* when generating Triggers)
	 - **serviceAccountName**: the name of the service account used in triggers (*required* when generating Triggers)
	 - **steps**: the [Tekton steps](https://github.com/tektoncd/pipeline/blob/master/docs/tasks.md#defining-steps) to run in the pipeline (*required*)

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
