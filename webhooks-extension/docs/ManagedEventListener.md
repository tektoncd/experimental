# Important considerations of using the Webhooks Extension

A managed EventListener is provided that should not be modified.

When you create a webhook, a number of Trigger resources are referenced in the EventListener that's created or updated as more webhooks are added.

If these Trigger resources can no longer be found (be it through deletion or changed RBAC policies), broken webhooks will display in the table of webhooks and these will not be able to be deleted through the user-interface. In addition, you'll likely face trouble creating new webhooks referencing the same Pipeline and Git repository.

Procedure (with example below):
1) If the TriggerBinding exists somewhere as yaml, reapply it into the install namespace for the webhooks extension.
2) If the TriggerBinding is a generated one (name starts with `wext` (lowercase), create an empty one with that name).
3) Create any upfront provided TriggerBindings again. For example, if your pipeline was called `simple-pipeline-2`, you will be using `simple-pipeline-2-pullrequest-binding` and `simple-pipeline-2-push-binding`.

Here's example yaml you would apply to recreate the TriggerBindings such that deletion can happen successfully. This also serves to demonstrate the naming pattern used so you can identify the impacted webhook.


```
apiVersion: tekton.dev/v1alpha1
kind: TriggerBinding
metadata:
  name: wext-myhook-*unique string here*
  namespace: tekton-pipelines
---
apiVersion: tekton.dev/v1alpha1
kind: TriggerBinding
metadata:
  name: wext-monitor-task-github-binding-*unique string here*
  namespace: tekton-pipelines
---
apiVersion: tekton.dev/v1alpha1
kind: TriggerBinding
metadata:
  name: simple-pipeline-2-pullrequest-binding
  namespace: tekton-pipelines
---
apiVersion: tekton.dev/v1alpha1
kind: TriggerBinding
metadata:
  name: simple-pipeline-2-push-binding
  namespace: tekton-pipelines
```

You'll then be able to delete the webhook normally through the user-interface.