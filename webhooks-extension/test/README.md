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
- helm: Templating is used to install Istio according to [Knative docs](https://knative.dev/docs/install/installing-istio/)
- npm: For building the user interface

## Dependency Versioning
The installation script installs the latest version of Istio and known compatible versions of Knative as per values in the version section of [config.sh](https://github.com/tektoncd/experimental/blob/master/webhooks-extension/test/config.sh). Knative Serving 0.6 has made considerable changes to the service definition so levels below this are not supported.

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
_Note: Your git provider must be able to reach the address of this extension's sink. The sink is deployed with a Knative service, so you may need to configure Knative serving. We recommend [setting up a custom domain](https://knative.dev/v0.3-docs/serving/using-a-custom-domain/) with the extension `.nip.io`. The `install_prereqs.sh` script patches the workstation IP is patched into the knative serving config-map (config-domain) as mentioned in the previous link._

Webhooks are outbound HTTP requests from (in this case Git Hub) to your Kubernetes environment. If you are behind a firewall, it's unlikely that github.com will be able to reach you. The two most common testing scenarios are: 
- An in-house Git Hub Enterprise to your Docker Desktop
- github.com to your internet-facing kubernetes cluster in a commercial public cloud environment

The checked-in defaults current reflect the first of these two options. To test with a webhook you need to do some setup work. 

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
NAME                                                          READY   STATUS            RESTARTS   AGE
knative-demo-test-1557836364-build-simple-fgzjr-pod-e47e4d    0/4     Completed         0          15m
knative-demo-test-1557836364-deploy-simple-vpjst-pod-dcd3e2   0/4     Completed         0          14m
knative-demo-test-qqdkd-rq9w6-deployment-5f999f6f87-mlcpd     0/3     PodInitializing   0          7s
tekton-dashboard-fdc9ff8cc-4f4w2                              1/1     Running           0          23h
webhooks-extension-5b849f7d78-flbb7                           1/1     Running           0          23h
knative-demo-test-qqdkd-rq9w6-deployment-5f999f6f87-mlcpd     2/3     Running           0          12s
knative-demo-test-qqdkd-rq9w6-deployment-5f999f6f87-mlcpd     3/3     Running           0          13s
webhooks-extension-sink-zx6pm-deployment-756759c79c-x9vqn     0/3     Pending           0          0s
webhooks-extension-sink-zx6pm-deployment-756759c79c-x9vqn     0/3     Pending           0          0s
webhooks-extension-sink-zx6pm-deployment-756759c79c-x9vqn     0/3     Init:0/1          0          0s
webhooks-extension-sink-zx6pm-deployment-756759c79c-x9vqn     0/3     PodInitializing   0          2s
webhooks-extension-sink-zx6pm-deployment-756759c79c-x9vqn     1/3     Running           0          8s
webhooks-extension-sink-zx6pm-deployment-756759c79c-x9vqn     2/3     Running           0          8s
webhooks-extension-sink-zx6pm-deployment-756759c79c-x9vqn     3/3     Running           0          13s
knative-demo-test-1557837318-build-simple-h72pb-pod-8b6aed    0/4     Pending           0          0s
knative-demo-test-1557837318-build-simple-h72pb-pod-8b6aed    0/4     Pending           0          0s
knative-demo-test-1557837318-build-simple-h72pb-pod-8b6aed    0/4     Pending           0          8s
knative-demo-test-1557837318-build-simple-h72pb-pod-8b6aed    0/4     Init:0/2          0          8s
knative-demo-test-1557837318-build-simple-h72pb-pod-8b6aed    0/4     Init:1/2          0          10s
knative-demo-test-1557837318-build-simple-h72pb-pod-8b6aed    0/4     PodInitializing   0          11s
knative-demo-test-1557837318-build-simple-h72pb-pod-8b6aed    3/4     Running           0          18s
knative-demo-test-1557837318-build-simple-h72pb-pod-8b6aed    2/4     Running           0          37s
knative-demo-test-1557837318-build-simple-h72pb-pod-8b6aed    0/4     Completed         0          47s
knative-demo-test-1557837318-deploy-simple-tvk7v-pod-64fb78   0/4     Pending           0          0s
knative-demo-test-1557837318-deploy-simple-tvk7v-pod-64fb78   0/4     Pending           0          0s
knative-demo-test-1557837318-deploy-simple-tvk7v-pod-64fb78   0/4     Init:0/2          0          0s
knative-demo-test-1557837318-deploy-simple-tvk7v-pod-64fb78   0/4     Init:1/2          0          3s
knative-demo-test-1557837318-deploy-simple-tvk7v-pod-64fb78   0/4     PodInitializing   0          4s
myapp-566f7f7fd-6szzf                                         0/1     Pending           0          0s
myapp-566f7f7fd-6szzf                                         0/1     Pending           0          0s
myapp-566f7f7fd-6szzf                                         0/1     ContainerCreating 0          0s
knative-demo-test-1557837318-deploy-simple-tvk7v-pod-64fb78   1/4     Running           0          10s
knative-demo-test-1557837318-deploy-simple-tvk7v-pod-64fb78   0/4     Completed         0          11s
myapp-566f7f7fd-6szzf                                         1/1     Running           0          3s
```
The running `myapp` is the built and deployed code from `GITHUB_REPO`
