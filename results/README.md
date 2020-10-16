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
A local Mysql server configuration will be `./config/mysql-*.yaml`.

## Development

### Configure your database.

The API Server requires a SQL database to connect to for result storage.
database schema can be found under [schema/results.sql](schema/results.sql). We provide a MySQL server as default. To deploy the default MySQL server in Tekton, you need to first set the environment variable `MYSQL_ROOT_PASSWORD`, then run:

```
ko apply -f config/mysql-pv.yaml && ko apply -f config/mysql-deployment.yaml
```

Connection parameters are
[configured via environment variables](cmd/api/README.md). Configure these in
the API deployment config in [config/api.yaml](config/api.yaml).

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

4. Install the sqlite3

```
$ apt-get install sqlite3
```
