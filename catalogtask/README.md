# CatalogTask: Run Tasks from the Catalog without kubectl apply1

## Summary

This Custom Task controller demonstrates a way we could
resolve remote tasks from the catalog.

The goal of this project is simple: Users should never
have to type `kubectl apply -f git-clone.yaml` again.

## Usage

Run this controller locally with `./start.sh`.

You might need to edit it to make it work!

Warning: This will write the file cache to your local
disk (/tmp/ by default) for development purposes! It's
only a single git repo though.

Then apply a Run that uses a catalog task:

```yaml
# catalogtask/samples/run.yaml
apiVersion: tekton.dev/v1alpha1
kind: Run
metadata:
  name: catalog-ref-test
  namespace: default
spec:
  ref:
    apiVersion: catalogtask.tekton.dev/v1alpha1
    kind: Task
    name: git-clone--0.3
  params:
  - name: url
    value: https://github.com/tektoncd/pipeline.git
  workspaces:
  - name: output
    emptyDir: {}
```

The `git-clone` task will be fetched from the catalog
and executed as a TaskRun with the `task.spec` in the
`taskSpec` field and the `parameters` + `workspaces`
passed down from the Run.

`taskRun.Status.TaskRunResults` will be copied into the
Run's `run.Status.Results` when the TaskRun completes.

### TODO: This doesn't currently work in a cluster

At the moment my deployment won't run in a cluster and
I haven't figured out why. The controller binary exits
immediately with the `-h` usage instructions printed.

## Examples

- [Using CatalogTasks in a Run](./samples/run.yaml)
- [Using CatalogTasks in a Pipeline](./samples/run.yaml)

## Config and Syntax

### Private Catalogs

By default this controller will boot up using the
open source tekton catalog at `https://github.com/tektoncd/catalog.git`

To configure a private catalog instead
set the `CATALOG_GIT_URL` environment variable in
the [deployment](./config/500-controller.yaml).

If your private catalog requires credentials like
SSH keys then add these as `volumeMounts` to the controller's
template in
[./config/500-controller.yaml](./config/500-controller.yaml).

### Specifying Task Version

By default using a CatalogTask will pick its latest version. E.g. this
example will use the latest version of `github-close-issue` that it can
find (0.2 at time of writing):

```yaml
  ref:
    apiVersion: catalogtask.tekton.dev/v1alpha1
    kind: Task
    name: github-close-issue
```

The catalog stores tasks in versioned directories like this:

```
/task/github-close-issue/0.2/gihub-close-issue.yaml
```

You can specify a specific version of a Catalog Task to use
with this slightly awkward syntax:

```
  ref:
    apiVersion: catalogtask.tekton.dev/v1alpha1
    kind: Task
    name: github-close-issue--0.1 # <- Notice the --0.1 , that's the "version" syntax :/
```

### Performance

For **Xtreme Performance** use an in-memory `emptyDir`
volume as the catalog's cache. Pass the path to the cache
to the controller with the `CACHE_PATH` env var.

We do this in the [default deployment](./config/500-controller.yaml).

## Next Steps

- Get this running in clusters (a.k.a. why is my controller
deployment failing ?!?!)
- Allow an operator to pick specific versions of tasks that
are allowed (e.g. "I only want to allow git-clone v0.2")
- Give operators a way to fetch newer versions when the catalog
repo is updated.
