#!/bin/bash

# This script is for use on Docker Desktop on a Mac. It installs and configures the Tekton dashboard and the webhooks extension. 
# Typically for use after install_prereqs.sh
#
# Prereqs: 
# - ko
# - GOPATH set correctly
# - KO_DOCKER_REPO set to docker.io/your_dockerhub_id
# 
# Script could obviously be extended to take these as command line arguments

export tekton_repo_dir=$(git rev-parse --show-toplevel)
export test_dir="${tekton_repo_dir}/webhooks-extension/test"

source ${test_dir}/config.sh
source ${test_dir}/util.sh

echo "Installing Dashoard and webhooks extension into $DASHBOARD_INSTALL_NS namespace"
install_webhooks_extension $KO_DOCKER_REPO $DASHBOARD_INSTALL_NS
install_dashboard $DASHBOARD_INSTALL_NS

