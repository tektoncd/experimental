#!/bin/bash

export tekton_repo_dir=$(git rev-parse --show-toplevel)
export test_dir="${tekton_repo_dir}/webhooks-extension/test"

source ${test_dir}/config.sh
source ${test_dir}/util.sh

if [ $DASHBOARD_INSTALL_NS != "default" ]; then
  kubectl delete ns ${DASHBOARD_INSTALL_NS}
else
  kubectl delete deployment tekton-dashboard -n ${DASHBOARD_INSTALL_NS}
  kubectl delete deployment webhooks-extension -n ${DASHBOARD_INSTALL_NS}
  kubectl delete deployment tekton-webhooks-extension-validator -n ${DASHBOARD_INSTALL_NS}
  kubectl delete service webhooks-extension -n ${DASHBOARD_INSTALL_NS}
  kubectl delete service tekton-webhooks-extension-validator -n ${DASHBOARD_INSTALL_NS}
  kubectl delete task ingress-task -n ${DASHBOARD_INSTALL_NS}
  kubectl delete task monitor-task -n ${DASHBOARD_INSTALL_NS}
  kubectl delete task route-task -n ${DASHBOARD_INSTALL_NS}
  kubectl delete triggertemplate monitor-task-template -n ${DASHBOARD_INSTALL_NS}
  kubectl delete triggerbinding monitor-task-binding -n ${DASHBOARD_INSTALL_NS}
fi
