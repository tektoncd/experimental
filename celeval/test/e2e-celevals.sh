#!/usr/bin/env bash

# Copyright 2021 The Tekton Authors
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

# Helper functions for E2E tests.

source $(git rev-parse --show-toplevel)/vendor/github.com/tektoncd/plumbing/scripts/e2e-tests.sh

function install_celeval_crd() {
  echo ">> Deploying CELEval custom task"
  ko resolve -f config/ \
      | sed -e 's%"level": "info"%"level": "debug"%' \
      | sed -e 's%loglevel.controller: "info"%loglevel.controller: "debug"%' \
      | sed -e 's%loglevel.webhook: "info"%loglevel.webhook: "debug"%' \
      | kubectl apply -f - || fail_test "Build pipeline installation failed"
  verify_celeval_installation
}

function verify_celeval_installation() {
  # Make sure that everything is cleaned up in the current namespace
  delete_celeval_resources

  # Wait for pods to be running in the namespaces we are deploying to
  wait_until_pods_running tekton-pipelines || fail_test "CELEval controller or webhook did not come up"
}

function uninstall_celeval_crd() {
  echo ">> Uninstalling CELEval custom task"
  ko delete --ignore-not-found=true -f ${REPO_ROOT_DIR}/celeval/config/

  # Make sure that everything is cleaned up in the current namespace.
  delete_celeval_resources
}

function delete_celeval_resources() {
  for res in run; do
    kubectl delete --ignore-not-found=true ${res}.tekton.dev -l app.kubernetes.io/managed-by=CELEval
  done
  for res in celeval; do
    kubectl delete --ignore-not-found=true ${res}.custom.tekton.dev --all
  done
}
