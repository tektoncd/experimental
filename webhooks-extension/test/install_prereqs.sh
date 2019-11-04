#!/bin/bash

# This script is for use on Docker Desktop on a Mac. It installs Tekton, Tekton Triggers and nginx. It requires:
#  - kubectl

export tekton_repo_dir=$(git rev-parse --show-toplevel)
export test_dir="${tekton_repo_dir}/webhooks-extension/test"

source ${test_dir}/config.sh
source ${test_dir}/util.sh

install_tekton ${TEKTON_VERSION}
install_tekton_triggers ${TEKTON_TRIGGERS_VERSION}
install_nginx
