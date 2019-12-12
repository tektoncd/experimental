# Scripting
This directory will contain scripts used for several related purposes. 

- As a developer I want to set up a local test environment
  - From CLEAN: install all prerequisites
  - Having prereqs installed, set up a pipeline and webhook for a simple test repository
- We'll want automated tests that do much the same things. 

This is a work in progress and will take a while to settle down. 

## Install tooling requirements
You must install these tools:

- go: The language Tekton Pipelines is built in
- git: For source control
- dep: For managing external Go dependencies. - Please Install dep v0.5.0 or greater.
- ko: For development. ko version v0.1 or higher is required for pipeline to work correctly.
- kubectl: For interacting with your kube cluster
- [Node.js & npm](https://nodejs.org/): For building and running the frontend locally. _Node.js 10.x is strongly recommended_
- jq: Used in the setup_webhook_simple_test.sh script.
- kustomize: For development, unless and until `ko -k` happens. 

## Dependency Versioning
The installation script installs versions of Tekton and Tekton Triggers as per values in the version section of [config.sh](https://github.com/tektoncd/experimental/blob/master/webhooks-extension/test/config.sh).

## Install Dependencies/Prereqs
- Update `config.sh` as necessary
- Run `install_prereqs.sh`. 

## Install Dashboard and wehooks extension
- Update `config.sh` as necessary
- Check that `GOPATH` is set properly
- `docker login` (or be prompted later within execution)
- `export KO_DOCKER_REPO=docker.io/[your_dockerhub_id]`
- Run `install_dashboard_and_extension.sh`

# Testing 
## Test with a webhook
_Note: Your git provider must be able to reach the address of this extension's ingress/route as specified in the extension-deployment.yaml as environment variable WEBHOOK_CALLBACK_URL.

Webhooks are outbound HTTP requests from (in this case GitHub) to your Kubernetes environment. If you are behind a firewall, it's unlikely that github.com will be able to reach you. The two most common testing scenarios are: 

- An in-house Git Hub Enterprise to your Docker Desktop
- github.com to your internet-facing kubernetes cluster in a commercial public cloud environment

The checked-in defaults currently reflect the first of these two options. To test with a webhook you need to do some setup work. 

### Credentials 
Edit `credentials.sh` from 
```
DOCKERHUB_USERNAME=[your-dockerhub-id]
DOCKERHUB_PASSWORD=[your-docker-hub-password]
GITHUB_USERNAME=[your-github-login-id]
GITHUB_TOKEN=[github-token-with-wehooks-permissions]
```
to something of the form, 
```
DOCKERHUB_USERNAME=mnuttall
DOCKERHUB_PASSWORD=thisIsNotMyPassword
GITHUB_USERNAME=mnuttall
GITHUB_TOKEN=fbufliwufbe4wliufiuwebfliseubfweiluf
```

### Configuration
Edit the GITHUB settings in `config.sh`: 
```
# Set this to your github - used to create webhooks
export GITHUB_URL="https://github.ibm.com"

# This is the repo you want to set up a webhook for. See github.com/mnuttall/simple for a public copy of this repo. 
export GITHUB_REPO="https://github.ibm.com/MNUTTALL/simple"
```

### All set up, let's go! 

Now you're ready to go: 
- run `setup_webhook_simple_test.sh`
- `git push` a change to the git repository pointed to by GITHUB_REPO
- Watch Tekton and the webhooks-extension do their thing. 

For example, 
```
kubectl get pods -w

simple-pipeline-run-v7xf8-build-simple-crtjq-pod-14cfbf   0/5       Pending   0         0s
simple-pipeline-run-v7xf8-build-simple-crtjq-pod-14cfbf   0/5       Pending   0         0s
simple-pipeline-run-v7xf8-build-simple-crtjq-pod-14cfbf   0/5       Pending   0         8s
simple-pipeline-run-v7xf8-build-simple-crtjq-pod-14cfbf   0/5       Init:0/3   0         8s
simple-pipeline-run-v7xf8-build-simple-crtjq-pod-14cfbf   0/5       Init:1/3   0         9s
simple-pipeline-run-v7xf8-build-simple-crtjq-pod-14cfbf   0/5       Init:2/3   0         11s
simple-pipeline-run-v7xf8-build-simple-crtjq-pod-14cfbf   0/5       PodInitializing   0         12s
simple-pipeline-run-v7xf8-build-simple-crtjq-pod-14cfbf   5/5       Running   0         18s
simple-pipeline-run-v7xf8-build-simple-crtjq-pod-14cfbf   5/5       Running   0         18s
simple-pipeline-run-v7xf8-build-simple-crtjq-pod-14cfbf   4/5       Running   0         20s
simple-pipeline-run-v7xf8-build-simple-crtjq-pod-14cfbf   3/5       Running   0         21s
simple-pipeline-run-v7xf8-build-simple-crtjq-pod-14cfbf   2/5       Running   0         45s
simple-pipeline-run-v7xf8-build-simple-crtjq-pod-14cfbf   1/5       Running   0         2m
simple-pipeline-run-v7xf8-build-simple-crtjq-pod-14cfbf   0/5       Completed   0         2m
simple-pipeline-run-v7xf8-deploy-simple-74w6n-deployment--mt8ld-pod-03efde   0/1       Pending   0         0s
simple-pipeline-run-v7xf8-deploy-simple-74w6n-deployment--mt8ld-pod-03efde   0/1       Pending   0         0s
simple-pipeline-run-v7xf8-deploy-simple-74w6n-deployment--mt8ld-pod-03efde   0/1       Init:0/2   0         0s
simple-pipeline-run-v7xf8-deploy-simple-74w6n-deployment--mt8ld-pod-03efde   0/1       Init:1/2   0         2s
simple-pipeline-run-v7xf8-deploy-simple-74w6n-deployment--mt8ld-pod-03efde   0/1       PodInitializing   0         3s
simple-pipeline-run-v7xf8-deploy-simple-74w6n-deployment--mt8ld-pod-03efde   1/1       Running   0         6s
simple-pipeline-run-v7xf8-deploy-simple-74w6n-deployment--mt8ld-pod-03efde   1/1       Running   0         6s
simple-pipeline-run-v7xf8-deploy-simple-74w6n-deployment--mt8ld-pod-03efde   0/1       Completed   0         8s
simple-pipeline-run-v7xf8-deploy-simple-74w6n-pod-407fd1   0/3       Pending   0         0s
simple-pipeline-run-v7xf8-deploy-simple-74w6n-pod-407fd1   0/3       Pending   0         0s
simple-pipeline-run-v7xf8-deploy-simple-74w6n-pod-407fd1   0/3       Init:0/2   0         0s
simple-pipeline-run-v7xf8-deploy-simple-74w6n-pod-407fd1   0/3       Init:1/2   0         3s
simple-pipeline-run-v7xf8-deploy-simple-74w6n-pod-407fd1   0/3       PodInitializing   0         4s
simple-pipeline-run-v7xf8-deploy-simple-74w6n-pod-407fd1   2/3       ErrImagePull   0         21s
simple-pipeline-run-v7xf8-deploy-simple-74w6n-pod-407fd1   2/3       ImagePullBackOff   0         23s
simple-pipeline-run-v7xf8-deploy-simple-74w6n-pod-407fd1   3/3       Running   0         37s
simple-pipeline-run-v7xf8-deploy-simple-74w6n-pod-407fd1   3/3       Running   0         37s
simple-pipeline-run-v7xf8-deploy-simple-74w6n-pod-407fd1   1/3       Running   0         40s
myapp-86d96bd579-skzk8   0/1       Pending   0         0s
myapp-86d96bd579-skzk8   0/1       Pending   0         0s
myapp-86d96bd579-skzk8   0/1       ContainerCreating   0         0s
simple-pipeline-run-v7xf8-deploy-simple-74w6n-pod-407fd1   0/3       Completed   0         42s
myapp-86d96bd579-skzk8   1/1       Running   0         3s

```
The running `myapp` is the built and deployed code from `GITHUB_REPO`
