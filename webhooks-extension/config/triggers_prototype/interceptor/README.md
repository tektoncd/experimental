# tekton-validate-github-event

For https://github.com/tektoncd/experimental/issues/245

Checks the following

1) Valid X-Hub signature (secret token used in validation service matches the secret token used on the webhook)
2) Repository URL matches the input URL parameter - so we only activate Triggers for selected repositories
3) Eventually - that it's only for a push or pull request event

## Build and push

`docker build -t mydockerusername/myrepository:latest .`
`docker push mydockerusername/myrepository:latest`

**Make sure you modify the GitHub validation TaskRun to refer to the resulting image coordination**