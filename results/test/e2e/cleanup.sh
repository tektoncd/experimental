#!/bin/bash

export DOCKER_IN_DOCKER_ENABLED="true"
export KIND_CLUSTER_NAME=${KIND_CLUSTER_NAME:-"tekton-results"}
kind delete cluster --name=tekton-results