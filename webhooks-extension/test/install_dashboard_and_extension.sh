#!/bin/bash

# This script is for use on Docker Desktop on a Mac. It installs and configures the Tekton dashboard and the webhooks extension.
# Typically for use after install_prereqs.sh
#
# Prereqs:
# - Tekton pipelines applied
# - ko
# - GOPATH set correctly
# - KO_DOCKER_REPO set to docker.io/$DOCKERHUB_USERNAME from credentials.sh
#
# Script could obviously be extended to take these as command line arguments

export tekton_repo_dir=$(git rev-parse --show-toplevel)
export test_dir="${tekton_repo_dir}/webhooks-extension/test"

source ${test_dir}/config.sh
source ${test_dir}/util.sh

echo -e "\n\nPlease choose which versions you would like to install:"
echo "1) Current Releases"
echo "2) Nightly Builds"
echo "3) Developments Builds"

condition_check='[[ $VERSION == "" ]] || [[ $VERSION != [1-3] ]]'
while $(eval "${condition_check}");
do
    read -p "Version: " -n 1 -r VERSION
    if $(eval "${condition_check}"); then
      echo -e "\n\nInvalid input, try again."
    fi
done

if [[ $VERSION == "1" ]]; then
  VERSION_TEXT="CURRENT RELEASES"
elif [[ $VERSION == "2" ]]; then
  VERSION_TEXT="NIGHTLY BUILDS"
else
  VERSION_TEXT="DEVELOPMENT BUILDS"
fi

# Check that GOPATH and KO_DOCKER_REPO are set properly
check GOPATH
check KO_DOCKER_REPO "export KO_DOCKER_REPO=docker.io/${DOCKERHUB_USERNAME}"

echo -e "\n\nInstalling the $VERSION_TEXT of the Tekton Dashoard and Webhooks Extension into $DASHBOARD_INSTALL_NS namespace.\n"

install_webhooks_extension $DASHBOARD_INSTALL_NS $VERSION
install_dashboard $DASHBOARD_INSTALL_NS $VERSION

wait_for_ready_pods $DASHBOARD_INSTALL_NS 60 10
