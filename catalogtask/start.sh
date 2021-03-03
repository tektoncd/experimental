#!/usr/bin/env sh
set -xe
clear
CATALOG_GIT_URL="https://github.com/tektoncd/catalog.git" 	\
	 METRICS_DOMAIN="tekton.dev/pipelines" 			\
	 SYSTEM_NAMESPACE="tekton-pipelines" 			\
	 go run ./cmd/controller --				\
		 -kubeconfig "~/.kube/config"
