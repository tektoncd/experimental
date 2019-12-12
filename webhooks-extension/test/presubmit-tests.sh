#!/usr/bin/env bash

# Copyright 2018 The Tekton Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This script runs the presubmit tests; it is started by prow for each PR.
# For convenience, it can also be executed manually.
# Running the script without parameters, or with the --all-tests
# flag, causes all tests to be executed, in the right order.
# Use the flags --build-tests, --unit-tests and --integration-tests
# to run a specific set of tests.

# Markdown linting failures don't show up properly in Gubernator resulting
# in a net-negative contributor experience.
export DISABLE_MD_LINTING=1
export TEST_FOLDER=$(pwd)

source $(dirname $0)/../../vendor/github.com/tektoncd/plumbing/scripts/presubmit-tests.sh

function get_node() {
    echo "Script is running as $(whoami) on $(hostname)"
    # It's Stretch and https://github.com/tektoncd/dashboard/blob/master/package.json
    # denotes the Node.js and npm versions
    apt-get update
    apt-get install -y curl
    curl -O https://nodejs.org/dist/v10.15.3/node-v10.15.3-linux-x64.tar.xz
    tar xf node-v10.15.3-linux-x64.tar.xz
    export PATH=$PATH:$(pwd)/node-v10.15.3-linux-x64/bin
}

function extra_initialization() {
    dep ensure -v
    get_node
    echo ">> npm version"
    npm --version
    echo ">> Node.js version"
    node --version
}

function post_build_tests() {
    popd
}

function node_npm_install() {
    local failed=0
    mkdir ~/.npm-global
    npm config set prefix '~/.npm-global'
    export PATH=$PATH:$HOME/.npm-global/bin
    npm ci || failed=1 # similar to `npm install` but ensures all versions from lock file
    return ${failed}
}

function node_test() { 
    local failed=0
    echo "Running node tests from $(pwd)"
    node_npm_install || failed=1
    npm run lint || failed=1
    npm run test ci || failed=1
    echo ""
    
    echo "Checking bundle hash matches"
    npm run build
  
    hash=$(ls -t dist/extension.*.js | head -1 | cut -f 2 -d '.')
    echo "LATEST HASH: $hash"

    yaml=$(grep -i "tekton-dashboard-bundle-location:" base/300-extension-service.yaml | cut -f 2 -d ':' | cut -f 2 -d '.')
    echo "YAML HASH in base/300-extension-service.yaml: $yaml"
    
    if [[ $hash != $yaml ]]; then
      echo "######## FAIL/ERROR ########"
      echo "--------------------------------------------------------------------------"
      echo "HASH MISMATCH BETWEEN ACTUAL BUILD AND YAML: check values in base/300-extension-service.yaml"
      echo "--------------------------------------------------------------------------"
      failed=1
    fi
    
    return ${failed}
}

function post_unit_tests() {
    popd
}

function post_integration_tests() {
    popd
}

function pre_build_tests() {
    pushd ${TEST_FOLDER}
}

function pre_unit_tests() {
    pushd ${TEST_FOLDER}
    header "webhooks-extension pre_unit_tests"
    # Runs linting and UI tests and returns the exit code
    node_test
    exit_code=$?
    return $exit_code
}

function pre_integration_tests() {
    pushd ${TEST_FOLDER}
}

# June 28th 2019: work around https://github.com/tektoncd/plumbing/issues/44
function unit_tests() {
  local failed=0
  echo "Using overridden unit_tests"  
  go test -v -race ./... || failed=1
  echo "unit_tests returning $@"
  return ${failed}
}

# We use the default build, unit and integration test runners.
main $@
