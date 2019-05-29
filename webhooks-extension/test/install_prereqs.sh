#!/bin/bash

# This script is for use on Docker Desktop on a Mac. It installs Istio, Knative and Tekton. It requires:
#  - kubectl
#  

export tekton_repo_dir=$(git rev-parse --show-toplevel)
export test_dir="${tekton_repo_dir}/webhooks-extension/test"

source ${test_dir}/config.sh
source ${test_dir}/util.sh
  
install_istio
install_knative_serving ${KNATIVE_VERSION}
install_knative_eventing ${KNATIVE_VERSION}
install_knative_eventing_sources ${KNATIVE_VERSION}
install_tekton ${TEKTON_VERSION}

# Docker desktop: cluster IP = host IP. Obviously not true for other types of cluster. 
#   kubectl cluster-info | cut -d'/' -f3 | cut -d':' -f1 | head -n 1
# returns 'localhost' for Docker Desktop; not tested on other cluster types. 
# For minikube, ip=$(minikube ip)
ip=$(ifconfig | grep netmask | sed -n 2p | cut -d ' ' -f2) 
kubectl patch configmap config-domain --namespace knative-serving --type='json' \
  --patch '[{"op": "add", "path": "/data/'"${ip}.nip.io"'", "value": ""}]'
