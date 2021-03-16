#!/usr/bin/env bash

# Copyright 2019 The Tekton Authors
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

function run() {
    folder=$1
    header "${folder}"
    if should_test_folder $folder ; then
      shift
      pushd $(dirname $0)/../${folder} || return 1
      if [[ -f ./test/presubmit-tests.sh ]]; then
          ./test/presubmit-tests.sh $@ || exited=1
      else
          echo "Skip due to no './test/presubmit-tests.sh' file"
      fi
      popd >/dev/null
      return $exited
    fi
    echo "Skip - no files changed"
    return 0
}

function should_test_folder() {
    # If initialize_environment failed to identify the changed files, fall back to testing everything.
    if [[ -z "$(cat ${CHANGED_FILES})" ]]; then
        echo "Cannot determine changed files.  Testing all projects." && return 0
    fi
    for file in $(cat "${CHANGED_FILES}"); do
        # If changed file is in the folder then test that folder.
        echo $file | grep -q "^$1/.*" && echo "$file is modified so testing project $1" && return 0
        # If changed file is outside all project folders then test every folder.
        echo $file | grep -q -v $inanyprojectregex && echo "$file is modified so testing all projects" && return 0
    done
    return 1
}

# Get list of changed files
initialize_environment

projects="catalogs cel commit-status-tracker generators helm hub oci pipeline/cleanup  pipelines-in-pipelines tekdoc task-loops webhooks-extension"
inanyprojectregex=$(echo "^${projects// /\/.* ^}\/.*" | sed 's/ /\\|/g')

for proj in $projects; do
    run $proj $@ || exit 1
done
