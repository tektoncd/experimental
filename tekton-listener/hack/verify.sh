#!/usr/bin/env bash

# Copyright 2018 The Knative Authors
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

set -o errexit
set -o nounset
set -o pipefail

source $(dirname $0)/../vendor/github.com/knative/test-infra/scripts/library.sh

readonly TMP_DIFFROOT="$(mktemp -d ${REPO_ROOT_DIR}/tmpdiffroot.XXXXXX)"

cleanup() {
  rm -rf "${TMP_DIFFROOT}"
}

trap "cleanup" EXIT SIGINT

cleanup

# Save working tree state
mkdir -p "${TMP_DIFFROOT}/tekton-listener/pkg"
cp -aR "${REPO_ROOT_DIR}/tekton-listener/Gopkg.lock" "${REPO_ROOT_DIR}/tekton-listener/pkg" "${REPO_ROOT_DIR}/tekton-listener/vendor" "${TMP_DIFFROOT}/tekton-listener"

"${REPO_ROOT_DIR}/tekton-listener/hack/generate.sh"
echo "Diffing ${REPO_ROOT_DIR}/tekton-listener against freshly generated codegen"
ret=0
diff -Naupr "${REPO_ROOT_DIR}/tekton-listener/pkg" "${TMP_DIFFROOT}/tekton-listener/pkg" || ret=1
diff -Naupr --no-dereference "${REPO_ROOT_DIR}/tekton-listener/vendor" "${TMP_DIFFROOT}/tekton-listener/vendor" || ret=1

# Restore working tree state
rm -fr "${REPO_ROOT_DIR}/tekton-listener/Gopkg.lock" "${REPO_ROOT_DIR}/tekton-listener/pkg" "${REPO_ROOT_DIR}/tekton-listener/vendor"
cp -aR "${TMP_DIFFROOT}"/tekton-listener/* "${REPO_ROOT_DIR}"/tekton-listener

if [[ $ret -eq 0 ]]
then
  echo "${REPO_ROOT_DIR}/tekton-listener up to date."
else
  echo "${REPO_ROOT_DIR}tekton-listener/ is out of date. Please run hack/update-codegen.sh"
  exit 1
fi
