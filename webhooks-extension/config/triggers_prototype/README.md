# triggers-prototype

This area is for files related to using Triggers, currently through the Tekton Webhooks Extension.

## Ingress

You can create or delete Ingress using the provided Task and TaskRun.

- Apply the Task definition:
`kubectl apply -f config/triggers_prototype/ingress.yaml`

Modify the TaskRun definition, replacing the parameters accordingly.

To create Ingress, set `Mode` to `create`. To delete Ingress, set this to `delete`.

- Apply the TaskRun definition, thus running the Task:
`kubectl create -f config/triggers_prototype/ingress-run.yaml`

## Using Docker Desktop

Note that in order for the EventListener service to be reachable over Ingress, you should install your own LoadBalancer - for example with:

```bash
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/master/deploy/static/mandatory.yaml

kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/master/deploy/static/provider/cloud-generic.yaml
```

To test the Ingress set up you can then do the curl:

`curl -k -L -d 'foo' listener.<your external IP address used.nip.io -v`

and you should see a response back from the running EventListener pod.
