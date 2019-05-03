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

# Note that eventing-sources.yalm was renamed from release.yaml in the v0.5.0 release, so this won't work for earlier releases as-is. 
function install_knative_eventing_sources() {
    if [ -z "$1" ]; then
        echo "Usage ERROR for function: install_knative_eventing_sources [version]"
        echo "Missing [version]"
        exit 1
    fi
    version="$1"
    kubectl apply --filename https://github.com/knative/eventing-sources/releases/download/${version}/eventing-sources.yaml
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

# Install Dashboard (https://github.com/tektoncd/dashboard)
#
# As of May 2nd this is a bit fraught since dashboard does not yet have images pushed to gcr.io
# 
# So: look to see if $GOPATH/src/github.com/tektoncd/dashboard exists
# If it doesn't : check that the user is ok for us to clone and ko apply it
# If it does: check that the user is ok for us to cd into and ko apply it

function install_dashboard() { 
  if [ -z "$1" ]; then
        echo "Usage ERROR for function: install_dashboard [namespace]"
        echo "Missing [namespace]"
        exit 1
  fi
  namespace="$1"
  check GOPATH
  check KO_DOCKER_REPO docker.io/your_docker_id

  dashboard_dir=$GOPATH/src/github.com/tektoncd/dashboard 
  if [ -d $dashboard_dir ]; then 
    echo "Dashboard source detected at $dashboard_dir"
    read -p "Do you wish to ko apply the current source? " -n 1 -r apply
  else
    echo "Dashboard source not detected in $dashboard_dir"
    read -p "Do you wish to git clone and ko apply the master version? " -n 1 -r cloneAndApply
  fi
  if [[ $cloneAndApply =~ ^[Yy]$ ]]; then
    pushd $GOPATH/src/github.com/tektoncd/
    git clone https://github.com/tektoncd/dashboard.git
    popd
    apply="y"
  fi
  if [[ $apply =~ ^[Yy]$ ]]; then
    pushd $GOPATH/src/github.com/tektoncd/dashboard
    docker login
    npm install
    npm run build_ko
    dep ensure -v
    ko apply -f config -n $namespace
    popd
  fi
}

# Install https://github.com/tektoncd/experimental/tree/master/webhooks-extension
function install_webhooks_extension() { 
  if [ -z "$2" ]; then
    echo "Usage ERROR for function: install_webhooks_extension [docker-registry] [target-namespace"
    echo "Missing [namespace]"
    exit 1
  fi 
  possiblyKoDockerRegistry=$1 
  namespace=$2

  # dockerRegistry is all the text after the last / in possiblyKoDockerRegistry which may have been given via KO_DOCKER_REPO
  dockerRegistry=$(echo $possiblyKoDockerRegistry | awk -F "/" '{print $NF}')    
  echo "dockerRegistry='$possiblyKoDockerRegistry' stripped to '$dockerRegistry', namespace='$namespace'"

  echo "build cmd/extension ..."
  pushd $GOPATH/src/github.com/tektoncd/experimental/webhooks-extension
  docker build -t ${dockerRegistry}/extension:latest -f cmd/extension/Dockerfile . || fail "extension docker build failed"
  docker push ${dockerRegistry}/extension:latest || fail "extension docker push failed"

  echo "build cmd/sink ..."
  docker build -t ${dockerRegistry}/extension-sink:latest -f cmd/sink/Dockerfile . || fail "sink docker build failed"
  docker push ${dockerRegistry}/extension-sink:latest || fail "sink docker push failed"

  # Copy and replace the config/ yaml files into install/
  mkdir -p install
  cp -r config/ install/
  for file in install/*.yaml; do
      sed -i '' 's/DOCKER_REPO/'"$dockerRegistry"'/g' "${file}"
  done

  # Install into the NAMESPACE
  echo "install in namespace ${namespace}"
  kubectl apply -f install/ -n ${namespace} || fail "kubectl apply failed"

  echo "the extension and sink yaml files were successfully applied into the $namespace namespace"
  popd
  wait_for_ready_pods $namespace 60 10
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

function check() { 
  if [ -z "$1" ]; then
    echo "Usage ERROR for function: check [varname] (example-value)"
    echo "Missing [varname]"
    exit 1
  fi
  vToCheck=${!1}
  example=$2
  if [ -z "$vToCheck" ]; then 
    echo -n "$1 is unset. Please set $1 before running this script. "
    if [[ ! -z "$example" ]]; then
      echo "For example, '$example'"
    else
      echo ""
    fi
    exit 1
  else 
    echo  "Detected $1 set to '${vToCheck}'"
  fi
}

# Helper function to exit script
function fail() {
    echo "Error: $1."
    exit 1
}
