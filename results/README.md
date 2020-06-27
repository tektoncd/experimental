
# Tekton Results API

This package contains experimental code to support a richly queryable API for Tekton execution history and results.

The full proposal is here: https://docs.google.com/document/d/1-XBYQ4kBlCHIHSVoYAAf_iC01_by_KoK2aRVO0t8ZQ0/edit

The main components of this design are a **queryable indexed API server** backed by persistent storage, and an **in-cluster watcher** to report updates to the API server.

The API server interface is defined in `./proto/api.proto`, and a reference
implementation backed by MySQL lives in `./cmd/api`. A reference implementation
of the in-cluster watcher lives in `./cmd/watcher`.

## Development

### One-Time Setup

Create a random secret password for the `root` user on the database:

```
kubectl create secret generic mysql-db-creds \
    -n tekton-pipelines \
    --from-literal=mysql-db-user=root \
    --from-literal=mysql-db-password=$(head -c 20 /dev/urandom | base64)
```

This creates a random password which will be used by both the mysql instance,
and the API server that connects to it. Its value doesn't matter, but if you
need to see it for some reason, you can run:

```
kubectl get secret mysql-db-creds -n tekton-pipelines -ojsonpath='{.data.mysql-db-password}' | base64 -D
```


### Deploying

To build and deploy both components, use [`ko`](https://github.com/GoogleCloudPlatform/ko). Make sure you have a valid kubeconfig, and have set the `KO_DOCKER_REPO` env var.

```
ko apply -f config/
```

To only build and deploy one component:

```
ko apply -f config/watcher.yaml
```

### Regenerating protobuf-generated code

1. Install protobuf compiler

e.g., for macOS:

```
brew install protobuf
```

2. Install the protoc Go plugin

```
$ go get -u github.com/golang/protobuf/protoc-gen-go
```

3. Rebuild the generated Go code

```
$ go generate ./proto/
```

### Teardown

To delete associated resources:

```
kubectl delete -f config/
```

And don't forget to delete the secret:

```
kubectl delete secret mysql-db-creds -n tekton-pipelines
```
>>>>>>> Connect to a local MySQL instance to store results
