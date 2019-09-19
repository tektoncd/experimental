# triggers-prototype

This area is for files related to using Triggers, currently through the Tekton Webhooks Extension.

## Ingress

You can create or delete an Ingress using the provided Task, TaskRun, and certificate generation script (typically done at install time by a cluster administrator).

1. Generate the self-signed certificate (for example, using `config/triggers_prototype/scripts/generate-certificate.sh mypassphrase listener.myexternalipaddress.nip.io mycertificatesecret`)
2. Apply the Task definition:
`kubectl apply -f config/triggers_prototype/ingress.yaml`
3. Modify the example TaskRun definition, replacing the parameters accordingly:

  - To create an Ingress, set `Mode` to `create`. To delete an Ingress, set this to `delete`
  - Check the `CertificateSecretName` parameter matches the Kubernetes secret name made in step 1 (this defaults to `mycertificatesecret`)

4. Apply the TaskRun definition:
`kubectl create -f config/triggers_prototype/test/ingress-run.yaml`

## Using Docker Desktop

In order for the EventListener service to be reachable over Ingress, you should install your own LoadBalancer - for example with:

```bash
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/master/deploy/static/mandatory.yaml
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/master/deploy/static/provider/cloud-generic.yaml
```

To test the Ingress set up, use the following `curl` command:

`curl -k -L -d 'foo' listener.<your external IP address>.nip.io -v`

and you should see a response back from the running EventListener pod.

In this case, `-k` is to allow a self-signed certificate to be used, `-L` is to follow redirects and `-d` specifies the data to be sent.

# GitHub Webhooks

You can create or delete GitHub webhooks using the provided Task and TaskRun.

- Apply the Task definition:
`kubectl apply -f config/triggers_prototype/github/webhook.yaml`

- Create a GitHub secret, for example with the Tekton Webhooks Extension or through `kubectl`. You will refer to the secret's name in the next step.

If you opt to use `kubectl`, create the following file (for example named `secret.yaml`):

```
apiVersion: v1
kind: Secret
metadata:
  name: basic-user-pass
  annotations:
    tekton.dev/git-0: https://github.com
type: kubernetes.io/basic-auth
stringData:
  accessToken: <your access token used to access the Git provider>
  secretToken: <anything here - it's used to validate the webhook>
```

then do `kubectl create -f secret.yaml`.

Modify the TaskRun definition, replacing the parameters accordingly.

To create a webhook, set `Mode` to `create`. To delete a webhook, set this to `delete`.

- Apply the TaskRun definition, thus running the Task:
`kubectl create -f config/triggers_prototype/github/webhook-run.yaml`

For deleting, you'll need to specify the WebhookName parameter and can omit the ExternalUrl.

To use an access token, you can specify GithubUserNameKey as "".

Note that in order to not receive 503 service unavailable errors from the webhook when trying to reach your Ingress, you should ensure your Ingress is correctly configured (for example, check that the service name and port matches that of the EventListener you are using it with).
