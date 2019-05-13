# Extension

## API Definitions

### GET endpoints

```
GET /webhooks
Get all webhooks
Returns HTTP code 200 and all the webhooks
Returns HTTP code 500 if an error occurred getting the webhooks

Example payload response
[
 {
  "name": "go-hello-world",
  "namespace": "green",
  "gitrepositoryurl": "https://github.com/ncskier/go-hello-world",
  "accesstoken": "github-secret",
  "pipeline": "simple-pipeline"
 }
]
```

```
GET /webhooks/defaults
Get default values, currently install namespace and docker registry
Returns HTTP code 200

Example payload response
{
 "namespace": "default",
 "dockerregistry": "mydockerhubregistry"
}
```

### POST endpoints

```
POST /webhooks
Create a new webhook
Request body must contain name, namespace gitrepositoryurl, accesstoken, and pipeline
Request body may contain serviceaccount, dockerregistry, helmsecret, and repositorysecretname
Returns HTTP code 201 if the webhook was created successfully
Returns HTTP code 400 if an error occurred with the request body
Returns HTTP code 500 if an error occurred reading or writing the webhooks

Example POST
{
  "name": "go-hello-world",
  "namespace": "green",
  "gitrepositoryurl": "https://github.com/ncskier/go-hello-world",
  "accesstoken": "github-secret",
  "pipeline": "simple-pipeline"
}
```

### DELETE endpoint

```
DELETE /webhooks/<webhookid>?namespace=<my namespace>

You can optionally add &deletepipelineruns=true to remove all PipelineRuns associated with the same repository.

Returns HTTP code 201 if the webhook was deleted successfully
Returns HTTP code 400 if an error occurred with the request body
Returns HTTP code 404 if the webhook wasn't found
Returns HTTP code 405 if a query parameter alone was provided
Returns HTTP code 500 if any other errors occurred

Deletes the GithubSource (therefore the webhook from the repository) and optionally deletes all PipelineRuns for the configured repository. 
The ConfigMap used to maintain a list of configured webhooks to Pipelines is also updated.
```


These endpoints can be accessed through the dashboard.

If using Helm, you can also specify a Helm release name. If no Helm release name is provided, your Helm release name will default to be the repository name.

Specify a Helm release name by providing `releasename` in the POST request.

The release name __must be less than 64 characters in length__: if your repository name does not meet this requirement you must specify a `releasename` that is less than 64 characters.
