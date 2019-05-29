#!/bin/bash

##### Version specs
# These defaults are known compatible versions
export KNATIVE_VERSION="v0.6.0"
export TEKTON_VERSION="0.3.0"
# You can also Specify exact version/release: https://github.com/istio/istio/releases
export ISTIO_VERSION="latest"
# Side car injection gets stuck in "Container Creating" state when disabled
export ISTIO_SIDECAR_INJECTION="true"
# To prevent Git Hub rate limiting when pulling latest Istio
export GITHUB_TOKEN=''

##### Dashboard specs
export DASHBOARD_INSTALL_NS="default"

# Note that to receive webhooks, your github must be able to http POST to your Tekton installation. 
# Our initial testing has used Docker Desktop and GitHub Enterprise. 

# Set this to your github - used to create webhooks
export GITHUB_URL="https://github.ibm.com"

# This is the repo you want to set up a webhook for. See github.com/mnuttall/simple for a public copy of this repo. 
export GITHUB_REPO="https://github.ibm.com/MNUTTALL/simple" 