# Tekton Sample Custom Task Controller

The Tekton Sample Custom Task Controller is a simple implementation of a Tekton custom task
that evaluates a regular expression.

The sample demonstrates how to
* be notified of Run objects that reference a particular `apiVersion` and `kind`
* process parameters in a Run object
* set the successful condition and a result in the Run object's status

For more information about the running custom tasks,
see the [Run documentation](https://github.com/tektoncd/pipeline/blob/master/docs/runs.md).

## Installing the custom task

You need to have [Go](https://golang.org/) and [ko](https://github.com/google/ko) installed on your workstation
to build and deploy the sample custom task controller.

You need to [configure `ko` to push images to your Docker repository](https://github.com/google/ko#usage).

1. Clone the repository to your workstation.  
2. Set the current directory to the location where you cloned the repository.
3. Use the following command to build and deploy the controller.
    ```
    $ ko apply -f config/
    ```
    `ko apply` will invoke `kubectl` and therefore apply to whatever kubectl context is active.
4. Check that the pod for the controller is running.
    ```
    $ kubectl get pods -n tekton-sample-custom-task
    NAME                                                   READY   STATUS    RESTARTS   AGE
    tekton-sample-custom-task-controller-79bd557c4-sp2j4   1/1     Running   0          1m
    ```

## Running the custom task

You can use the YAML files provides in the [examples](examples) folder to run the sample custom task.

## Making your own custom task

You can use the sample as a base for your own custom task.
You should change the `groupName`, `version` and `kind` constants defined in [controller.go](pkg/reconciler/sample/controller.go)
to values that are specific and unique to your custom task.

