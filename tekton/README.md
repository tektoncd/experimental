# Tekton Resources

This folder contains tekton resources used build and release custom tasks.
The resources included in `kustomization.yaml` are installed nightly on the `dogfooding`
cluster, and used to run nightly releases.

Individual projects will be updated to use these common release resources
([#805](https://github.com/tektoncd/experimental/issues/805)). Please don't
add more copy-pasted release Pipelines for releasing Custom Tasks.

## Nightly releases

[The release pipeline](release-pipeline.yaml) is
[triggered nightly by Tekton](https://github.com/tektoncd/plumbing/tree/main/tekton/resources/nightly-release).

This Pipeline uses:

- [publish](publish.yaml)
- [git-clone](https://hub.tekton.dev/tekton/task/git-clone)
- [gcs-upload](https://hub.tekton.dev/tekton/task/gcs-upload) 
- [golang-build](https://hub.tekton.dev/tekton/task/golang-build)
- [golang-test](https://hub.tekton.dev/tekton/task/golang-test)

### Service account and secrets

In order to release, these Pipelines expects a service account JSON file to be 
passed via a workspace.

### Container registry access

The `publish-release` task uses `crane` to authenticate to the container registry as well
as to tag container images to the various regions. It uses `ko` to build the container
images, publish them to the container registry and store the release file on the workspace.

The image which we use for this is built from a [`Dockerfile`](https://github.com/tektoncd/plumbing/blob/main/tekton/images/ko/Dockerfile)
in the plumbing repo.
