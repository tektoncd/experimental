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

# This script runs at the postsubmit phase; it is started by prow when a push event
# happens on master, via a PR merge for example.
set -e
source $(dirname $0)/../vendor/github.com/tektoncd/plumbing/scripts/presubmit-tests.sh

for p in webhooks-extension tekton-listener; do
    header "Publish image for ${p}"
    pushd $(dirname $0)/../${p} > /dev/null
    set +e
    ./test/publish-images.sh
    set -e
    popd > /dev/null
done
