# Sink

## API Definitions

### POST endpoints

```
POST /
Handles a webhook by creating a PipelineRun that builds and deploys your project.
Request must be from a GitHub webhook. See documentation about them at the following link: https://developer.github.com/enterprise/2.16/webhooks/
Returns HTTP code 201 if the PipelineRun was created successfully
Returns HTTP code 400 if an error occurred with the request header or body
Returns HTTP code 500 if an error occurred reading or writing the webhooks
```