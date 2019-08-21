# Tekton Webhooks Extension CI/CD

This directory contains the Tekton `Tasks` and `Pipelines` used to create Webhooks Extension releases. 

These tasks run on your local cluster, and then copy the release artifacts - Docker images and yaml files - into [the `tekton releases` bucket in Google Cloud Platform](https://console.cloud.google.com/storage/browser/tekton-releases/webhooks-extension). Your cluster must contain keys for a Google account with the necessary authorization in order for this to work. 

## Release process

First, ensure that your credentials are set up correctly. You will need an account with access to [Google Cloud Platform](https://console.cloud.google.com). Your account must have 'proper authorization to release the images and yamls' in the [`tekton-releases` GCP project](https://github.com/tektoncd/plumbing#prow). Your account must have `Permission iam.serviceAccountKeys.create`. Contact @bobcatfish or @dlorenc if you are going to be creating releases and require this authorization.

- You will need to install the [`gcloud` CLI](https://cloud.google.com/sdk/gcloud/) in order to get keys out of Google Cloud and into your local cluster. These credentials will be patched onto the service account to be used by the Tekton PipelineRuns. You do not need to create a new GCP project or pay Google any money. 
- It's convenient to use the ['tkn' CLI](https://github.com/tektoncd/cli) to kick off builds, so grab that as well. 

Create and run this Bash script:
```bash
KEY_FILE=release.json
GENERIC_SECRET=release-secret
# The kubernetes ServiceAccount that will be used by your Tekton tasks. 'default' is the default. It should all ready exist. 
SERVICE_ACCOUNT=default 
GCP_ACCOUNT="release-right-meow@tekton-releases.iam.gserviceaccount.com"

# 1. Create a private key for the service account, which you can use
gcloud iam service-accounts keys create --iam-account $GCP_ACCOUNT $KEY_FILE

# 2. Create kubernetes secret, which we will use via a service account and directly mounting
kubectl create secret generic $GENERIC_SECRET --from-file=./$KEY_FILE

# 3. Add the docker secret to the service account
kubectl patch serviceaccount $ACCOUNT -p "{\"secrets\": [{\"name\": \"$GENERIC_SECRET\"}]}"
```

Next:

1. Install [Tekton pipelines](https://github.com/tektoncd/pipeline) into your local cluster. 
1. Determine the commit ID you'd like to be publishing.
1. Edit the `tekton-experimental-git` PipelineResource in `resources.yaml` and set `spec.params.revision.value` to 'vX.Y.Z' e.g., `v0.2.5`. This can also be a Git commit if you have not created a release yet. 
1. From the root directory of the Webhooks Extension folder, create the Tekton Webhooks Extension release pipeline:
```bash
kubectl apply -f tekton -n tekton-pipelines
``` 

Now you can kick off the release build:
```bash
tkn pipeline start webhooks-extension-release -p versionTag=<version to publish> -r source-repo=tekton-experimental-git -r bucket=tekton-bucket -r builtWebhooksExtensionExtensionImage=webhooks-extension-extension-image -r builtWebhooksExtensionSinkImage=webhooks-extension-sink-image -n tekton-pipelines
```

Monitor the build logs to see the image coordinates that the image is pushed to. The `release.yaml` should appear under https://console.cloud.google.com/storage/browser/tekton-releases/webhooks-extension. 

## Manually fix the release up

We have a number of tasks that are yet to be automated:
- Fix the image tags of any unpinned images, and the newly generated images, in `webhooks-extension_release.yaml`
- Write the release notes
- Attach the `webhooks-extension_release.yaml` file to the corresponding Dashboard release, ensuring the correct bundle location is referenced in this file (this is printed from the build job and was checked in with the commit you are using too)