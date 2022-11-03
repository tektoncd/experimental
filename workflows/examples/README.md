The examples in this folder are working Tekton Workflows,
but they require some additional setup steps.
As more features are added to the Workflows project, this setup description
will be updated to reflect the reduced burden these features will require.

## Setup

- First, create a random webhook secret value.
- Put this value in a Kubernetes secret named "webhook-secret" under the key named "token".
- Replace "url" with a repo you own that has a Dockerfile.
- Next, set up a GitHub webhook on this repo with the same secret.
- Point this webhook at the ingress created for the Workflows EventListener
"workflows-listener" in the "tekton-workflows" namespace.