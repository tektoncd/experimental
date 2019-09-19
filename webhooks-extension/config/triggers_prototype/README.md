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
