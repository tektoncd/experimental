#!/bin/bash

export tekton_repo_dir=$(git rev-parse --show-toplevel)
export test_dir="${tekton_repo_dir}/webhooks-extension/test"

source ${test_dir}/config.sh
source ${test_dir}/util.sh

if [ $DASHBOARD_INSTALL_NS != "default" ]; then
  kubectl delete ns ${DASHBOARD_INSTALL_NS}
else
  kubectl delete deployment tekton-dashboard -n default
  kubectl delete deployment webhooks-extension -n default
  kubectl delete githubsource knative-demo-test -n default
fi
kubectl delete ns tekton-pipelines
kubectl delete ns knative-eventing
kubectl delete ns knative-serving
kubectl delete ns istio-system
