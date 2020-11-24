#!/bin/bash
# Copyright 2020 The Tekton Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -e

export DOCKER_IN_DOCKER_ENABLED="true"
export KIND_CLUSTER_NAME=${KIND_CLUSTER_NAME:-"tekton-results"}
export KO_DOCKER_REPO=${KO_DOCKER_REPO:-"kind.local"}

ROOT="$(git rev-parse --show-toplevel)/results"

echo "Using kubectl context: $(kubectl config current-context)"

echo "Installing Tekton Pipelines..."
TEKTON_PIPELINE_CONFIG=${TEKTON_PIPELINE_CONFIG:-"https://storage.googleapis.com/tekton-releases/pipeline/latest/release.yaml"}
kubectl apply --filename ${TEKTON_PIPELINE_CONFIG}

echo "Generating DB secret..."
# Don't fail if the secret isn't created - this can happen if the secret already exists.
kubectl create secret generic tekton-results-mysql --namespace="tekton-pipelines" --from-literal=user=root --from-literal=password=$(openssl rand -base64 20) || echo "continuing anyway..."

echo "Generating DB init config..."
kubectl create configmap mysql-initdb-config --from-file="${ROOT}/schema/results.sql" --namespace="tekton-pipelines" || echo "continuing anyway..."

echo "Installing Tekton Results..."
ko apply --filename="${ROOT}/config/"

echo "Waiting for deployments to be ready..."
kubectl wait deployment "tekton-results-mysql" --namespace="tekton-pipelines" --for="condition=available" --timeout="60s"
kubectl wait deployment "tekton-results-api" --namespace="tekton-pipelines" --for="condition=available" --timeout="60s"
kubectl wait deployment "tekton-results-watcher" --namespace="tekton-pipelines" --for="condition=available" --timeout="60s"