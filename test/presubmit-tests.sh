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

source $(dirname $0)/../vendor/github.com/tektoncd/plumbing/scripts/presubmit-tests.sh

IS_TEKTONCD_LISTENER_ONLY=0
IS_WEBHOOKS_EXTENSION_ONLY=0

pr_only_contains "tekton-listener" && IS_TEKTONCD_LISTENER_ONLY=1
pr_only_contains "webhooks-extension" && IS_WEBHOOKS_EXTENSION_ONLY=1

function run() {
    folder=$1
    shift
    pushd $(dirname $0)/../${folder} >/dev/null 2>/dev/null
    ./test/presubmit-tests.sh $@ || exited=1
    popd >/dev/null 2>/dev/null
    return $exited
}

if (( IS_TEKTONCD_LISTENER_ONLY )); then
    header "Only tekton-listener"
    run tekton-listener $@ || exit 1
elif (( IS_WEBHOOKS_EXTENSION_ONLY)); then
    header "Only webhookS-extension"
    run webhook-extension $@ || exit 1
else
    header "All the tests"
    run tekton-listener $@ || exited=1
    run webhook-extension $@ || exited=1
    exit $exited
fi
