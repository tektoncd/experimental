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