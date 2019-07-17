# Developing

Involved development scripts and instructions can be found in the [development installation guide](https://github.com/tektoncd/experimental/blob/master/webhooks-extension/test/README.md#scripting)

## UI development

If modifying the UI code and wanting to deploy your updated version:

1) Run `npm run build` this will create a new file in the dist directory
2) Update config/extension-service.yaml to reference this newly created file in the dist directory, this should simply be a matter of changing the hash value - do not change the web/ prefix

    `tekton-dashboard-bundle-location: "web/extension.214d2396.js"`

3) Reinstall the extension. It will automatically be picked up by the running dashboard

## Linting

Run `npm run lint` to execute the linter. This will ensure code follows the conventions and standards used by the project.

Run `npm run lint:fix` to automatically fix a number of types of problem including code style.

## Running tests

`GO_ENABLED=0 go test github.com/tektoncd/experimental/webhooks-extension/pkg/endpoints -v -race`

To run a specific test:
`GO_ENABLED=1 go test github.com/tektoncd/experimental/webhooks-extension/pkg/endpoints -v -race -run [test_name]`

## API Definitions

- [Extension API definitions](cmd/extension/README.md)
  - The extension API endpoints can be accessed through the dashboard.
- [Sink API definitions](cmd/sink/README.md)

### Example creating a webhook

We would recommend using the extension via the Tekton Dashboard UI - however, for development purposes you can create your webhook using curl:

```bash
data='{
  "name": "go-hello-world",
  "namespace": "green",
  "gitrepositoryurl": "https://github.com/ncskier/go-hello-world",
  "accesstoken": "github-secret",
  "pipeline": "simple-pipeline",
  "dockerregistry": "mydockerregistry"
  "pulltask": "monitor-result-task",
  "onsuccesscomment": "OK",
  "onfailurecomment": "ERROR",
}'
curl -d "${data}" -H "Content-Type: application/json" -X POST http://localhost:8080/webhooks
```

`pulltask`: task name that monitors the pipeline and updates the pull request when it finishes. The dfault task is `monitor-result-task`

`onsuccesscomment`: comment text that is put into the pull request when the pipelinerun finishes sccessfully.  The default is `OK: <pipelinerun name>`

`onfailurecomment`: comment text that is put into the pull request when the pipelinerun finishes with errors.  The default is `ERROR: <pipelinerun name>`


When curling through the dashboard, use the same endpoints; for example, assuming the dashboard is at `localhost:9097`:

```bash
curl -d "${data}" -H "Content-Type: application/json" -X POST http://localhost:9097/webhooks
```

Reference the [Knative eventing GitHub source sample](https://knative.dev/docs/eventing/samples/github-source/) to properly create the `accesstoken` secret. This is the secret that is used to create GitHub webhooks.

## Architecture information

Each webhook that the user creates will store its configuration information in a config map in the install namespace. This information is used by the sink to create `PipelineRuns` for webhook events.