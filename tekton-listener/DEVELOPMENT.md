# Tekton Listener development guide

## Setup instructions

```bash
export KO_DOCKER_REPO='gcr.io/my-gcloud-project-name'
ko apply -f config/
```

### Minikube

To dev/test locally with minikube:

* Get the `ko` command: `go get -u github.com/google/ko/cmd/ko`
* Load your docker environment vars: `eval $(minikube docker-env)`
* Start a registry: `docker run -it -d -p 5000:5000 registry:2`
* Set `KO_DOCKER_REPO` to local registry: `export KO_DOCKER_REPO=localhost:5000/<myproject>`
* Apply tekton components: `ko apply -L -f config/`
* Create an EventBinding (such as the example above) and await cloud events.
* The Listener that the EventBinding creates can be used as an Eventing sink.