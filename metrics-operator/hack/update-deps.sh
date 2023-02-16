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
set -o errexit
set -o nounset
set -o pipefail

source $(git rev-parse --show-toplevel)/vendor/github.com/tektoncd/plumbing/scripts/library.sh

# Copied from scripts.sh
# Update licenses.
# Parameters: $1 - output file, relative to repo root dir.
#             $2...$n - directories and files to inspect.
function update_licenses() {
  cd ${REPO_ROOT_DIR}/metrics-operator || return 1
  local dst=$1
  shift
  go-licenses save ./... --save_path=${dst} --force
  # Hack to make sure directories retain write permissions after save. This
  # can happen if the directory being copied is a Go module.
  # See https://github.com/google/go-licenses/issues/11
   chmod +w $(find ${dst} -type d)
}

# HACK: Most plumbing scripts assume everything is relative to ${REPO_ROO_DIR}
# This does not work for experimental since there are multiple projects within
# subfolders. We also cannot set ${REPO_ROOT_DIR} since it is a readonly variable

cd ${REPO_ROOT_DIR}/metrics-operator

# Update when you want to pin knative.dev/pkg
VERSION="master"

# The list of dependencies that we track at HEAD and periodically
# float forward in this repository.
FLOATING_DEPS=(
  "knative.dev/pkg@${VERSION}"
)

# Parse flags to determine any we should pass to dep.
GO_GET=0
while [[ $# -ne 0 ]]; do
  parameter=$1
  case ${parameter} in
    --upgrade) GO_GET=1 ;;
    *) abort "unknown option ${parameter}" ;;
  esac
  shift
done
readonly GO_GET

if (( GO_GET )); then
  go get -d ${FLOATING_DEPS[@]}
fi


# Prune modules.
go mod tidy
go mod vendor
update_licenses third_party/
