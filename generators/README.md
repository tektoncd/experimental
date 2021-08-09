
# Tekton Generators
This project contains experimental code to create a tool for generating Tekton spec from simplified configs. This is important because right now it is not easy for users to use Tekton resources to bootstrap common workflows in a configurable way and users usually need to set up a couple of configuration files on their own. As a result, the objective of Tekton Generators is to help users create and run their pipelines more easily and efficiently.

Generators can use a simple spec input to automate the pipeline set up for users. Different ways of running the pipeline (eg. PipelineRun, Trigger) should also be generated at the same time. Users may need to use the resources on their cluster to help build the pipeline. A command line tool can be used to help interact with generators more easily.

See [tektoncd/pipeline/#2590](https://github.com/tektoncd/pipeline/issues/2590) for more information and background.

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

## CLI
Users can use the generators CLI to choose to generate the pipeline with PipelineRun or Triggers to run it. 
The following commands help you understand and effectively use the generators CLI:
- `tkn-gen pipelinerun`: Manage the generated config with PipelineRun.
- `tkn-gen trigger`: Manage the generated config with Trigger.

In terms of the commands `tkn-gen pipelinerun` and `tkn-gen trigger`, their sub-commands are as follows:
-   `show`: Print generated configuration.    
-   `write`: Write generated configuration to disk.   
-   `apply`: Apply generated configuration to Kubernetes (based on local k8s context).  
-   `delete`: Delete generated resources from the k8s cluster.

For example, assume we have the input file `test.yaml`, we can run the spec in the file with the PipelineRun on the cluster by:
```
tkn-gen pipelinerun apply -f test.yaml
```

For every `tkn-gen` command, you can use `-h` or `--help` flags to display specific help for that command.


## GitHub Generator

GitHub is common for users to build their work. In order to make it easier to configure the Tekton resources, the goal is to build the GitHub type of Tekton Generator to help users work with their pipelines.

### Credentials 
#### GitHub token
You will need to create a [GitHub Personal Access Token](https://docs.github.com/en/github/authenticating-to-github/creating-a-personal-access-token#creating-a-token) and set it in the Kubernetes scecret `github` in the key `token` like this:
```
kubectl create secret generic github --from-literal token="YOUR_GITHUB_PERSONAL_ACCESS_TOKEN"
```

#### Webhook secret
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

### Service Account
The [`serviceAccountName`](https://github.com/tektoncd/triggers/blob/master/docs/eventlisteners.md#serviceAccountName) is a required field for Tekton Triggers.  You need to provide it in the input GitHub config settings. You can create the service account with the [webhook secret](#webhook-secret) that is used in the Kubernetes cluster like this :
```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: tekton-generators-github-sa
secrets:
  - name: webhook-secret
```
With the service account, you also need to create a role to follow certain roles and the role binding. See the [example](https://github.com/tektoncd/triggers/blob/main/examples/rbac.yaml) to create your own.
Use `kubectl apply` to create these resources on the cluster as well.

**Please Note: If you use Kaniko in the task, you need to make the service account have enough credentials to push images.  On GKE, it has been configured already. You can ignore this.**

### Dependent Tasks

The generated config would expect to use the tasks already on the cluster adding to the pipeline. The tasks include [`git-clone`](https://github.com/tektoncd/catalog/blob/v1beta1/git/git-clone.yaml) and [`github-set-status`](https://github.com/tektoncd/catalog/blob/v1beta1/github/set_status.yaml). The `git-clone` task is used to help clone a repo into the workspace. The `status-task` is used to help allow external services to mark GitHub commits with a state.
**Please Note: this git-clone Task is only able to fetch code from the public repo for the time being.**

#### Install the Task
You can install the tasks with the specified revision(commit SHA) on the command line like this:
```
kubectl apply -f https://raw.githubusercontent.com/tektoncd/catalog/<revision>/github/set_status.yaml
kubectl apply -f https://raw.githubusercontent.com/tektoncd/catalog/<revision>/git/git-clone.yaml
```

### Input configurations
The input provided by users is a simplified config file. Some useful identifiers like API version, Kind and Metadata are included in the input config. Users should be able to specify the runtime configuration however they want. A common case is to get started with users’ GitHub repo and build steps. This is to build the initial schema of the github generator. Here is the example of the GitHub input config:
```yaml
kind: GitHub
metadata:
  name: github-build
spec:
  url: "https://github.com/YolandaDu1997/hello-world"
  revision: 6c6ed17cd60127f96da41f51224914b2e825f939
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
#### Fields
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

With the input config, we can generate the Tekton task with the steps built by users, then the Pipeline using taskRef with the prepended git-clone task, generated task and finally task. The tasks are executed in a specific order within the Pipeline. Basically, the pipeline will do the following:
- Clone the given repo to the shared workspaces
- Build the steps that users want in the input config
- Set the commit status according to the execution result

### Run with PipelineRun
If you want to run Pipeline with PipelineRun, the fields `secretName`, `secretKey` and `serviceAccountName` are optional. You need to follow the document to install the dependent catalog tasks. Use the CLI to manage the PipelineRun with the specified parameters values, workspaces with specific disk storage and access modes in the input config. Then it will pass them to the Pipeline during execution.

### Run with Trigger 
If you want to run Pipeline with Trigger, it is optional to set up the commit sha, because the generated Tekton EventListener will contain triggers to listen to both GitHub push and GitHub pull request events. We can capture fields like the commit sha from an event and store them as parameters in the TriggerBinding, which then can be passed to TriggerTemplate. 

The resource template used in the TriggerTemplate is also the Tekton PipelineRun. The difference from the one described before is that the value of parameters in this PipelineRun is coming from TriggerTemplate. When generating the EventListener, the interceptors of processing GitHub push and pull request events are included in the triggers. The EventListener connects the TriggerBinding and TriggerTemplate. When the HTTP based events with JSON payloads comes, it creates the Template resources accordingly.

To run the pipeline with Triggers, except for the dependent catalog tasks, other dependencies like [webhook secret](#webhook-secret), [service account](#service-account), [role-based access control (RBAC)](https://github.com/tektoncd/triggers/blob/main/examples/rbac.yaml) are also required.
