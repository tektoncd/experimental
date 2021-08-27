# GitHub Notifier

This controller listens for Tekton TaskRuns that are annotated with specific
integration values to post information back to GitHub.

This runs in 2 modes:

- GitHub App : Converts the TaskRun into a
  [CheckRun](https://docs.github.com/en/rest/guides/getting-started-with-the-checks-api)
  with corresponding status and logs.
- GitHub OAuth : Converts the TaskRun into a
  [Status](https://docs.github.com/en/github/collaborating-with-issues-and-pull-requests/about-status-checks).

This does not do anything to grant the running TaskRun access to GitHub
credentials (e.g. for cloning the repo) -
https://github.com/tektoncd/catalog/tree/master/task/github-app-token/0.1 can be
used for this purpose for the time being (though note this doesn't support
fine-grained installation permissions).

## Creating CheckRuns/Statuses

The controller uses annotations with the prefix `github.integrations.tekton.dev`
to identify and track TaskRuns to publish.

| Annotation                                  | Description                                                                                                                                                                                                                                                                                                                                                                 |
| ------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| github.integrations.tekton.dev/installation | (GitHub App only) GitHub App Installation ID. This can be found in GitHub webhook events under the [`installations.id` field](https://docs.github.com/en/enterprise-server@2.20/developers/webhooks-and-events/webhook-events-and-payloads#webhook-payload-object-common-properties).                                                                                       |
| github.integrations.tekton.dev/owner        | GitHub org or user who owns the repo (for `github.com/tektoncd/test`, this should be `tektoncd`).                                                                                                                                                                                                                                                                           |
| github.integrations.tekton.dev/repo         | GitHub repo name (for `github.com/tektoncd/test`, this should be `test`).                                                                                                                                                                                                                                                                                                   |
| github.integrations.tekton.dev/checkrun     | (GitHub App / output only) GitHub CheckRun ID. If set, the controller will update this CheckRun instead of creating a new one.                                                                                                                                                                                                                                              |
| github.integrations.tekton.dev/name         | Display name to use for GitHub CheckRun/Status. If not specified, defaults to `{{ .Namespace }}/{{ .Name }}`. You can use `text/template` templating syntax to generate name and access any variables of [`TaskRun`](https://github.com/tektoncd/pipeline/blob/main/pkg/apis/pipeline/v1beta1/taskrun_types.go) inside.                                                     |
| github.integrations.tekton.dev/url          | Details URL to use for GitHub CheckRun/Status. If not specified, defaults to `https://dashboard.dogfooding.tekton.dev/#/namespaces/{{ .Namespace }}/taskruns/{{ .Name }}`. You can use `text/template` templating syntax to generate URL and access any variables of [`TaskRun`](https://pkg.go.dev/github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1#TaskRun) inside. |

For an example a full TaskRun with the initial annotations set, see
[pkg/controller/testdata/taskrun.yaml](pkg/controller/testdata/taskrun.yaml)
(these are the annotations you'll need to set for the notifier to work).

## Running the controller

| Environment Variable | Description                                                                                                                                                                                                                |
| -------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| GITHUB_APP_ID        | ID of your GitHub App. Can be found under https://github.com/settings/apps > Edit > General > About > App ID                                                                                                               |
| GITHUB_APP_KEY       | Path to the [private key of your GitHub App](https://docs.github.com/en/free-pro-team@latest/developers/apps/authenticating-with-github-apps#generating-a-private-key)                                                     |
| GITHUB_TOKEN         | GitHub Personal Access Token (https://docs.github.com/en/github/authenticating-to-github/creating-a-personal-access-token). We strongly recommend **not** using your own personal GitHub Account - use a bot user instead. |

If both GitHub App and GitHub OAuth variables are provided, the controller will
use GitHub App.
