## Notes for Amazon EKS

After creation of your first webhook, the following manual steps are necessary to make the webhook work in Amazon EKS environment.

1. Edit `el-tekton-webhooks-eventlistener` Ingress and add 2 annotations in the `metadata` section.

```
  metadata:
    annotations:
      alb.ingress.kubernetes.io/scheme: internet-facing
      kubernetes.io/ingress.class: alb
```

2. Edit `tekton-webhooks-eventlistener` EventListener and add `serviceType` `LoadBalancer` in the `spec` section

```
  spec:
    serviceType: LoadBalancer
```

3. Wait for `get ingress el-tekton-webhooks-eventlistener -n tekton-pipelines` showing the ADDRESS for the el-tekton-webhooks-eventlistener ingress

4. Edit `el-tekton-webhooks-eventlistener` Ingress again and update the URL of the `host` with the ADDRESS of the ingress


```
  spec:
    rules:
    - host: xxxx.yyy.elb.amazonaws.com
```

5. Update the `Payload URL` in the webhook in github.com repositry (Settings->Webhoks->"webhook with dummy URL"->Payload URL) to the ADDRESS of the ingress