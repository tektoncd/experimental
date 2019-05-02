#!/bin/bash

function install_istio_nodeport() {
    if [ -z "$1" ]; then
        echo "Usage ERROR for function: install_istio_nodeport [version]"
        echo "Missing [version]"
        exit 1
    fi
    version="$1"
    # Install on Minikube or Docker Desktop
    # We are changing LoadBalancer to NodePort for the istio-ingress service
    kubectl apply --filename https://github.com/knative/serving/releases/download/${version}/istio-crds.yaml &&
    curl -L https://github.com/knative/serving/releases/download/${version}/istio.yaml \
      | sed 's/LoadBalancer/NodePort/' \
      | kubectl apply --filename -

    # This works but why are we only labelling the default namespace? 
    # Isn't this needed on those namespaces in which we use knative-eventing?
    # If not is Istio really required for our purposes? 
    kubectl label namespace default istio-injection=enabled

    # Wait until all the pods come up
    wait_for_ready_pods istio-system 300 30
}

function install_knative_serving_nodeport() {
    if [ -z "$1" ]; then
        echo "Usage ERROR for function: install_knative_serving_nodeport [version]"
        echo "Missing [version]"
        exit 1
    fi
    version="$1"
    # Use NodePort instead of LoadBalancer for Minikube or Docker Desktop
    curl -L https://github.com/knative/serving/releases/download/${version}/serving.yaml \
    | sed 's/LoadBalancer/NodePort/' \
    | kubectl apply --filename -
    # Wait until all the pods come up
    wait_for_ready_pods knative-serving 180 20
}

function install_knative_eventing() {
    if [ -z "$1" ]; then
        echo "Usage ERROR for function: install_knative_eventing [version]"
        echo "Missing [version]"
        exit 1
    fi
    version="$1"
    kubectl apply --filename https://github.com/knative/eventing/releases/download/${version}/release.yaml
    # Wait until all the pods come up
    wait_for_ready_pods knative-eventing 180 20
}

function install_knative_eventing_sources() {
    if [ -z "$1" ]; then
        echo "Usage ERROR for function: install_knative_eventing_sources [version]"
        echo "Missing [version]"
        exit 1
    fi
    version="$1"
    kubectl apply --filename https://github.com/knative/eventing-sources/releases/download/${version}/release.yaml
    # Wait until all the pods come up
    wait_for_ready_pods knative-sources 180 20
}

# Install Tekton (instructions here: https://github.com/tektoncd/pipeline/blob/master/docs/install.md#adding-the-tekton-pipelines)
function install_tekton() {
    if [ -z "$1" ]; then
        echo "Usage ERROR for function: install_tekton [version]"
        echo "Missing [version]"
        exit 1
    fi
    version="$1"
    kubectl apply --filename https://storage.googleapis.com/tekton-releases/previous/${version}/release.yaml
    # Wait until all the pods come up
    wait_for_ready_pods tekton-pipelines 180 20
}



# Wait until all pods in a namespace are Running or Complete
# wait_for_ready [namespace] [timeout] <sleepTime>
# <sleepTime> is optional
function wait_for_ready_pods() {
    if [ -z "$1" ] || [ -z "$2" ]; then
        echo "Usage ERROR for function: wait_for_ready_pods [namespace] [timeout] <sleepTime>"
        [ -z "$1" ] && echo "Missing [namespace]"
        [ -z "$2" ] && echo "Missing [timeout]"
        exit 1
    fi
    namespace=$1
    timeout=$2
    sleepTime=${3:-"2"}

    # Loop check for ready resources
    emptyResponse="No resources found."
    # kubectl command prints to stderr, so redirect it to stdout
    response=$(kubectl get pods --namespace $namespace --field-selector status.phase!=Running,status.phase!=Succeeded 2>&1)
    readyResp=$(kubectl get pods --namespace $namespace -o json | jq '.items[]
        | {phase: .status.phase, conditions: .status.conditions}
        | select(.phase == "Running")
        | .conditions[]
        | select(.type == "Ready")
        | select(.status != "True")')
    ctr=0
    until [ "$response" = "$emptyResponse" ] && [ "$readyResp" = "" ]; do
        echo "waiting for pods in namespace $namespace:"
        echo "$response"
        if [ "$response" = "$emptyResponse" ]; then
            echo "waiting for ready:"
            echo "$readyResp" | jq '.'
        fi
        if [ "$timeout" -le "$ctr" ]; then
            echo "ERROR: exceeded timeout (${timeout}s) for namespace '${namespace}'"
            kubectl get pods --namespace $namespace
            kubectl describe pods --namespace $namespace
            return 1
        fi
        sleep "$sleepTime"
        ctr=$((ctr+sleepTime))
        # kubectl command prints to stderr, so redirect it to stdout
        response=$(kubectl get pods --namespace $namespace --field-selector status.phase!=Running,status.phase!=Succeeded 2>&1)
        readyResp=$(kubectl get pods --namespace $namespace -o json | jq '.items[]
        | {phase: .status.phase, conditions: .status.conditions}
        | select(.phase == "Running")
        | .conditions[]
        | select(.type == "Ready")
        | select(.status != "True")')
    done
    kubectl get pods --namespace $namespace
}
