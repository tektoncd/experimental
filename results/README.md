# Tekton Results API

This package contains experimental code to support a richly queryable API for
Tekton execution history and results.

The full proposal is here:
https://docs.google.com/document/d/1-XBYQ4kBlCHIHSVoYAAf_iC01_by_KoK2aRVO0t8ZQ0/edit

The main components of this design are a **queryable indexed API server** backed
by persistent storage, and an **in-cluster watcher** to report updates to the
API server.

The API server interface is defined in `./proto/api.proto`, and a reference
implementation backed by Sqlite will live in `./cmd/api`. A reference
implementation of the in-cluster watcher will live in `./cmd/watcher`.

## Development

### Configure your database.

The reference implementation of the API Server requires a SQL database for
result storage. The database schema can be found under
[schema/results.sql](schema/results.sql). 

Initial one-time setup is required to configure the password and initial config:

```sh
kubectl create secret generic tekton-results-mysql --namespace="tekton-pipelines" --from-literal=user=root --from-literal=password=$(openssl rand -base64 20)
kubectl create configmap mysql-initdb-config --from-file="schema/results.sql" --namespace="tekton-pipelines"
```

### Deploying

To build and deploy both components, use
[`ko`](https://github.com/GoogleCloudPlatform/ko). Make sure you have a valid
kubeconfig, and have set the `KO_DOCKER_REPO` env var.

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