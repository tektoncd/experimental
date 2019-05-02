#!/bin/bash

export tekton_repo_dir=$(git rev-parse --show-toplevel)
export test_dir="${tekton_repo_dir}/webhooks-extension/test"

source ${test_dir}/config.sh
source ${test_dir}/util.sh

kubectl delete ns ${DASHBOARD_INSTALL_NS}
kubectl delete ns knative-eventing
kubectl delete ns knative-serving
kubectl delete ns istio-system