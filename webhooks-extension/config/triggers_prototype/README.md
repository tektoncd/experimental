## GitHub validation task

The provided validation task is to be used in combination with an EventListener passing in EventOrigin, Github-Secret and Github-Secret-Key as parameters.

The secret key should be `secretToken` if using a secret created through the Tekton Webhooks Extension.

Two steps of validation can be performed.

1) Mandatory every time: that the secret for the webhook matches with the known secret on this Kubernetes cluster (used by the validate task)
2) Optional if you provide the EventOrigin oarameter: that the repository URL for the incoming webhook matches what the EventListener is listening for (done on a "per Trigger" basis)