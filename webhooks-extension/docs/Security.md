# Security

## Webhook 

Webhook SSL verification is enabled by setting the `WEBHOOK_CALLBACK_URL` environment variable to an `https://` endpoint and by setting the `SSL_VERIFICATION_ENABLED` environment variable to `"true"`.  These environment variables are found in the installation yaml file and need setting in advance of install.

The certificate setup is left as an exercise for the reader, but should you wish to have an `https://` connection but with certificate validation disabled, you could set the `WEBHOOK_CALLBACK_URL` environment variable to an `https://` endpoint and set the `SSL_VERIFICATION_ENABLED` environment variable to `"false"`.  Please ensure you understand the security risks of disabling certificate validation.

An additional security mechanism which is always enabled, is the validation of the `secret token` associated with the webhook.  This secret token is generated for you when you create the webhook in the UI and automatically checked by an interceptor service running behind the eventlistener.

## Certificate Verification

There are a number of additional places that verify the certificate from the git server:

- Tekton's Git Pipeline Resource

  Tekton's `pipelineresource` of type `git` performs a git clone of the source and can fail if you do not have signed certifcates in place for your git server (most comonly when using self hosted git servers and self signed certificates).

- Tekton's Pull Request Pipeline Resource

  Tekton's `pipelineresource` of type `pullrequest` performs interactions with the git server's api.  As with the git clone, the pull request pipeline resource can fail if there is a problem verifying the git server's certificate. 

- Webhook Extension Monitoring Task

  The webhook extension performs REST requests to the git server's api endpoint and additionally makes use of Tekton's pull request pipeline resource.  Certificate failures will cause status updates and reporting not to succeed.

### Disabling Certificate Verification

By setting the environment variable `SSL_VERIFICATION_ENABLED` to `"false"` you will disable certificate validation in the monitor task that udpdates status on your pull requests.  The setting is made available to your trigger templates as the parameters `webhooks-tekton-ssl-verify` and `webhooks-tekton-insecure-skip-tls-verify` which can then be used to set the requisite values on your pipelineresources (or you could hardcode the required values).

- Tekton's Git Pipeline Resource

  Ensure your `pipelineresource` of type `git` has an `sslVerify` param set to `false` - or utilise the value from the environment variable.

  ```
    - apiVersion: tekton.dev/v1alpha1
    kind: PipelineResource
    metadata:
      name: git-source-$(uid)
      namespace: $(params.webhooks-tekton-target-namespace)
    spec:
      params:
      - name: revision
        value: $(params.gitrevision)
      - name: url
        value: $(params.gitrepositoryurl)
      - name: sslVerify
        value: $(params.webhooks-tekton-sslVerify)
      type: git
  ```

- Tekton's Pull Request Pipeline Resource

  Ensure your `pipelineresource` of type `pullrequest` has an `insecure-skip-tls-verify` param set to `true` - or utilise the value from the environment variable.

  ```
   - apiVersion: tekton.dev/v1alpha1
    kind: PipelineResource
    metadata:
      name: pull-request-$(uid)
      namespace: tekton-pipelines
    spec:
      params:
      - name: url
        value: $(params.pullrequesturl)
      - name: insecure-skip-tls-verify
        value: $(params.insecure-skip-tls-verify)
      secrets:
      - fieldName: authToken
        secretKey: $(params.gitsecretkeyname)
        secretName: $(params.gitsecretname)
      type: pullRequest
  ```


