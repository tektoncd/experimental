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

These endpoints can be accessed through the dashboard.

If using Helm, you can also specify a Helm release name. If no Helm release name is provided, your Helm release name will default to be the repository name.

Specify a Helm release name by providing `releasename` in the POST request.

The release name __must be less than 64 characters in length__: if your repository name does not meet this requirement you must specify a `releasename` that is less than 64 characters.
