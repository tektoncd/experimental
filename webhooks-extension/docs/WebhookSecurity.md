# Webhook Security

Webhook SSL verification is enabled by setting the `WEBHOOK_CALLBACK_URL` environment variable to an `https://` endpoint and by setting the `SSL_VERIFICATION_ENABLED` environment variable to `"true"`.  These environment variables are found in the installation yaml file and need setting in advance of install.

The certificate setup is left as an exercise for the reader as this is untested and undocumented at this time.

An additional security mechanism which is always enabled, is the validation of the `secret token` associated with the webhook.  This secret token is generated for you when you create the webhook in the UI and automatically checked by an interceptor service running behind the eventlistener.