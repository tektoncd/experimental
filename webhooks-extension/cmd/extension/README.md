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


GET /webhooks/credentials?namespace=x
Get all credentials in namespace x
Returns HTTP code 200 and all the credentials
Returns HTTP code 500 if an error occurred getting the credentials

Example payload response
[ 
  { 
    "name": "anAccessToken", 
    accesstoken: "********",
    secrettoken: "thisIsMySecretToken"
  }
]
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


POST /webhooks/credentials?namespace=x
Create a new credential in namespace x
Request body must contain name and accesstoken. 
Request body may contain secrettoken. See https://github.com/knative/docs/blob/master/docs/eventing/samples/github-source/README.md for a discussion of this field. A random secrettoken will be created if none is supplied. 
Returns HTTP code 201 if the secret was created successfully
Returns HTTP code 400 if an error occurred with the request body 
Returns HTTP code 500 if an error occurred while creating the secret

Example POST
{
  "name": "my-access-token",
  "accesstoken": "ksdufbliubsliuvbsliucbsiucslicbsh98wehr8w9huwbcwb87ec"
}
```


### DELETE endpoints

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


DELETE /webhooks/credentials/<credential-name>?namespace=x

Deletes credential 'credential-name' from namespace x
Returns HTTP code 201 if the credential was deleted successfully
Returns HTTP code 404 if the credential wasn't found
Returns HTTP code 500 if any other errors occurred
```


These endpoints can be accessed through the dashboard.

If using Helm, you can also specify a Helm release name. If no Helm release name is provided, your Helm release name will default to be the repository name.

Specify a Helm release name by providing `releasename` in the POST request.

The release name __must be less than 64 characters in length__: if your repository name does not meet this requirement you must specify a `releasename` that is less than 64 characters.
