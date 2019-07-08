#!/bin/bash

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

# Loops until duration (car) is exceeded or command (cdr) returns success
# Lifted from https://github.com/openshift-cloud-functions/knative-operators/blob/master/etc/scripts/installation-functions.sh
function timeout() {
  SECONDS=0; TIMEOUT=$1; shift
  until eval $*; do
    sleep 5
    [[ $SECONDS -gt $TIMEOUT ]] && echo "ERROR: Timed out" && exit 1
  done
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

install_knative_serving $1
install_knative_eventing $1
install_knative_eventing_sources $1