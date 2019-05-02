#!/bin/bash

# Knative does not bundle Istio as of 0.6
# Installation instructions: https://knative.dev/docs/install/installing-istio/
function install_istio() {
  # Get specific version
  if [[ ${ISTIO_VERSION} == "latest" ]];then
    # This requires authentication if you reach the unauthenticated API limit
    export ISTIO_VERSION="$(get_latest_release "istio/istio")"
  fi
  echo "Istio version: ${ISTIO_VERSION}"

  rm -rf istio-*
  # Install on Minikube or Docker Desktop
  # This expects ${ISTIO_VERSION} to be set
  curl -L https://git.io/getLatestIstio | sh -
  cd istio-*
  for file in install/kubernetes/helm/istio-init/files/crd*yaml;do 
    kubectl apply -f $file
  done
  # Creates `istio-system` namespace
  kubectl apply -f install/kubernetes/namespace.yaml
  if [[ $ISTIO_SIDECAR_INJECTION ==  "false" ]];then
    helm template --namespace=istio-system \
    --set global.proxy.autoInject=disabled \
    --set global.omitSidecarInjectorConfigMap=true \
    --set global.disablePolicyChecks=true \
    --set prometheus.enabled=false \
    `# Disable mixer prometheus adapter to remove istio default metrics.` \
    --set mixer.adapters.prometheus.enabled=false \
    `# Disable mixer policy check, since in our template we set no policy.` \
    --set global.disablePolicyChecks=true \
    `# Set gateway pods to 1 to sidestep eventual consistency / readiness problems.` \
    --set gateways.istio-ingressgateway.autoscaleMin=1 \
    --set gateways.istio-ingressgateway.autoscaleMax=1 \
    `# Set pilot trace sampling to 100%` \
    --set pilot.traceSampling=100 \
    install/kubernetes/helm/istio \
    > ./istio-lean.yaml
    kubectl apply -f istio-lean.yaml
  else
    # A template with sidecar injection enabled.
    helm template --namespace=istio-system \
    --set sidecarInjectorWebhook.enabled=true \
    --set sidecarInjectorWebhook.enableNamespacesByDefault=true \
    --set global.proxy.autoInject=disabled \
    --set global.disablePolicyChecks=true \
    --set prometheus.enabled=false \
    `# Disable mixer prometheus adapter to remove istio default metrics.` \
    --set mixer.adapters.prometheus.enabled=false \
    `# Disable mixer policy check, since in our template we set no policy.` \
    --set global.disablePolicyChecks=true \
    `# Set gateway pods to 1 to sidestep eventual consistency / readiness problems.` \
    --set gateways.istio-ingressgateway.autoscaleMin=1 \
    --set gateways.istio-ingressgateway.autoscaleMax=1 \
    --set gateways.istio-ingressgateway.resources.requests.cpu=500m \
    --set gateways.istio-ingressgateway.resources.requests.memory=256Mi \
    `# More pilot replicas for better scale` \
    --set pilot.autoscaleMin=2 \
    `# Set pilot trace sampling to 100%` \
    --set pilot.traceSampling=100 \
    install/kubernetes/helm/istio \
    > ./istio.yaml
    kubectl apply -f istio.yaml
  fi
  # Wait until all the pods come up
  wait_for_ready_pods istio-system 300 30
  kubectl label namespace ${DASHBOARD_INSTALL_NS} istio-injection=enabled
}

function install_knative_serving() {
  if [ -z "$1" ]; then
      echo "Usage ERROR for function: install_knative_serving [version]"
      echo "Missing [version]"
      exit 1
  fi
  version="$1"
  curl -L https://github.com/knative/serving/releases/download/${version}/serving.yaml \
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

# Note that eventing-sources.yaml was renamed from release.yaml in the v0.5.0 release, so this won't work for earlier releases as-is. 
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
    timeout 60 "ko apply -f config -n $namespace"
    popd
  fi
}

# Install https://github.com/tektoncd/experimental/tree/master/webhooks-extension
function install_webhooks_extension() { 
  check GOPATH
  check KO_DOCKER_REPO 'export KO_DOCKER_REPO=docker.io/your_docker_id'

  pushd $GOPATH/src/github.com/tektoncd/experimental/webhooks-extension
  if [ -z "$1" ]; then
    echo "Usage ERROR for function: install_webhooks_extension [target-namespace]"
    echo "Missing [namespace]"
    exit 1
  fi 
  namespace=$1
  docker login
  npm install
  npm rebuild node-sass
  npm run build_ko
  dep ensure -v
  timeout 60 "ko apply -f config -n $namespace"
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
  timeout_period=$2
  timeout ${timeout_period} "kubectl get pods -n ${namespace} && [[ \$(kubectl get pods -n ${namespace} --no-headers 2>&1 | grep -c -v -E '(Running|Completed|Terminating)') -eq 0 ]]"
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

# Loops until duration (car) is exceeded or command (cdr) returns success
# Lifted from https://github.com/openshift-cloud-functions/knative-operators/blob/master/etc/scripts/installation-functions.sh
function timeout() {
  SECONDS=0; TIMEOUT=$1; shift
  until eval $*; do
    sleep 5
    [[ $SECONDS -gt $TIMEOUT ]] && echo "ERROR: Timed out" && exit 1
  done
}

# Credit: https://gist.github.com/lukechilds/a83e1d7127b78fef38c2914c4ececc3c
# $1: ORG/REPO (GitHub only)
function get_latest_release() {
  # Authentication to prevent rate limit
  if [[ -n ${GITHUB_TOKEN} ]];then
    auth='-H "Authorization: token '${GITHUB_TOKEN}'"'
  fi
  curl -s "${auth}" -- "https://api.github.com/repos/${1}/releases/latest" |   # Get latest release from GitHub api
  grep '"tag_name":' |                                                         # Get tag line
  sed -E 's/.*"([^"]+)".*/\1/'                                                 # Pluck JSON value
}
