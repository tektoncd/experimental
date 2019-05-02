# Tekton Listener and Event Bindings

This experimental directory defines two new CRDs - TektonListeners and EventBindings.

## TektonListener
The `TektonListener` defines a process by which PipelineRuns can be directly triggered by Cloudevents, which it consumes and handles to kickoff the pipeline.

To do this, an optional CRD `TektonListener` is provided. Once defined, the listener provides support for consuming CloudEvent and producing a predefined PipelineRun. It is intentionally designed to allow for other sources beyond CloudEvents.

The only event-type supported is `com.github.checksuite`.

An example TektonListener:
```
apiVersion: tekton.dev/v1alpha1
kind: TektonListener
metadata:
  name: test-build-tekton-listener
  namespace: tekton-pipelines
spec:
  selector:
    matchLabels:
      app: test-build-tekton-listener
  serviceName: test-build-tekton-listener
  template:
    metadata:
      labels:
        role: test-build-tekton-listener
    spec:
      serviceAccountName: tekton-pipelines-controller
  listener-image: github.com/tektoncd/pipeline/cmd/tektonlistener
  event-type: com.github.checksuite
  namespace: tekton-pipelines
  port: 80
  runspec:
    pipelineRef:
      name: demo-pipeline
    trigger:
      type: manual
    serviceAccount: 'default'
    resources:
    - name: source-repo
      resourceRef:
        name: skaffold-git
    - name: web-image
      resourceRef:
        name: skaffold-image-leeroy-web
    - name: app-image
      resourceRef:
        name: skaffold-image-leeroy-app
```

## EventBinding
The `EventBinding` allows a new, higher-level method to bind an Event with a specific PipelineRun. Once an EventBinding is created, no other resources are nneeded to accept, process and run Pipeline Resources. For the sake of security and to keep configuration simple, individual EventBindings are scoped to a specific pipeline - and Bindings also create all their own PipelineResources (and clean them up on removal as well). 

An example EventBinding:

```
apiVersion: tekton.dev/v1alpha1
kind: EventBinding
metadata:
  name: test-event-binding
  namespace: tekton-pipelines
spec:
  selector:
    matchLabels:
      app: test-event-binding
  template:
    metadata:
      labels:
        role: test-build-tekton-listener
  pipelineRef:
    name: demo-pipeline
  sourceref:
    name: demo-source
  resourceTemplates:
   - name: gitTemplate
     template:
       metadata:
       spec:
         type: git
         params:
         - name: revision
           valueFrom:
             fieldName: event.repo.revision
         - name: url
           valueFrom:
             fieldName: event.repo.name
  resources:
  - templateRef:
     name: gitTemplate
   resourceName: git
  - pipelineRef:
   name: demo-pipeline

```

# Minikube instructions

To dev/test locally with minikube:

* Get the `ko` command: `go get -u github.com/google/ko/cmd/ko`
* Load your docker environment vars: `eval $(minikube docker-env)`
* Start a registry: `docker run -it -d -p 5000:5000 registry:2`
* Set `KO_DOCKER_REPO` to local registry: `export KO_DOCKER_REPO=localhost:5000/<myproject>`
* Apply tekton components: `ko apply -L -f config/`
* Create an EventBinding (such as the example above) and await cloud events.
* The Listener that the EventBinding creates can be used as an Eventing sink.


Since the Service fullfills the [Addressable](https://github.com/knative/eventing/blob/master/docs/spec/interfaces.md#addressable) contract, the listener service can be used as a sink for [github source](https://knative.dev/docs/reference/eventing/eventing-sources-api/#GitHubSource), for example. So, once you have created the githubsource to have our new Listener as its sink, events can begin flowing and Pipelines begin running.
