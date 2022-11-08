The examples in this folder are working Tekton Workflows,
but they require some additional setup steps.
As more features are added to the Workflows project, this setup description
will be updated to reflect the reduced burden these features will require.

## Setup

- First, create a random webhook secret value.
- Put this value in a Kubernetes secret named "githubsecret" under the key named "secretToken".
The secret should be in the same namespace as the GitRepository.
- Replace "url" with a repo you own that has a Dockerfile.
- Next, create a personal access token with permission to create webhooks on this repo, and put
the PAT in the secret named "githubsecret" under the key named "accessToken".
