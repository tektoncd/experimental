#!/bin/bash

export tekton_repo_dir=$(git rev-parse --show-toplevel)
export test_dir="${tekton_repo_dir}/webhooks-extension/test"

source ${test_dir}/config.sh
source ${test_dir}/util.sh

# This script should be run after install_prereqs.sh and install_dashboard_and_extension.sh
# 
# Set up a webhook to trigger a pipeline suitable for manual testing via git push. 
# All secrets and tekton resources will be created in the same namespace as the dashboard for the purpose of this test. 

# Create secrets
# 1. We need a host:ip for the dashboard to curl commands to
#    Typically we port-forward the dashboard, but in this first version we're going to create a NodePort service
#    and talk to that via localhost:31001. 

check GOPATH
if [ ! -f $GOPATH/src/github.com/tektoncd/experimental/webhooks-extension/test/credentials.sh ]; then 
  echo "${GOPATH}/src/github.com/tektoncd/experimental/webhooks-extension/test must exist and contain adequate config. "
  exit 1
fi
pushd $GOPATH/src/github.com/tektoncd/experimental/webhooks-extension/test
source credentials.sh

kubectl apply -f dashboard-service.yaml -n ${DASHBOARD_INSTALL_NS} 

# Cleanup from the previous time we ran this script. Could use curl against dashboard APIs for extra testing.
kubectl delete secret docker-push -n ${DASHBOARD_INSTALL_NS}
kubectl delete secret github-repo-access-secret -n ${DASHBOARD_INSTALL_NS}
kubectl delete secret github-secret -n ${DASHBOARD_INSTALL_NS}
kubectl delete githubsource knative-demo-test -n ${DASHBOARD_INSTALL_NS}
kubectl delete task build-push -n ${DASHBOARD_INSTALL_NS}
kubectl delete task deploy-simple-kubectl-task -n ${DASHBOARD_INSTALL_NS}
kubectl delete pipeline simple-pipeline -n ${DASHBOARD_INSTALL_NS}
kubectl delete pipelineruns --all -n ${DASHBOARD_INSTALL_NS} 
rm -rf example-pipelines

# github-secret is used to created webhooks
# TODO: implement secretToken support
kubectl create secret generic github-secret \
  --from-literal=accessToken=$GITHUB_TOKEN \
  --from-literal=secretToken=$(cat /dev/urandom | LC_CTYPE=C tr -dc a-zA-Z0-9 | fold -w 32 | head -n 1) \
  --namespace $DASHBOARD_INSTALL_NS

# github-repo-access-secret is used to check code out of github
# TODO: This ought to work with just an accesstoken but currently we only have it working for user/pass with pass=token
post_data='{
  "name": "github-repo-access-secret",
  "type": "userpass",
  "username": "'"${GITHUB_USERNAME}"'",
  "password": "'"${GITHUB_TOKEN}"'",
  "url": {"tekton.dev/git-0": "'"${GITHUB_URL}"'"},
  "serviceaccount": "tekton-dashboard"
}'
curl -X POST --header Content-Type:application/json -d "$post_data" http://localhost:31001/v1/namespaces/${DASHBOARD_INSTALL_NS}/credentials
echo 'created github-repo-access-secret'

## docker-push secret used to push images to dockerhub
post_data='{
  "name": "docker-push",
  "type": "userpass",
  "username": "'"${DOCKERHUB_USERNAME}"'",
  "password": "'"${DOCKERHUB_PASSWORD}"'",
  "url": {"tekton.dev/docker-0": "https://index.docker.io/v1/"},
  "serviceaccount": "tekton-dashboard"
}'
curl -X POST --header Content-Type:application/json -d "$post_data" http://localhost:31001/v1/namespaces/${DASHBOARD_INSTALL_NS}/credentials
echo 'created docker-push'

## Install pipelines. This first test uses our simplest pipeline: docker build/tag/push, kubectl apply -f config 
git clone https://github.com/pipeline-hotel/example-pipelines.git
kubectl apply -f example-pipelines/config/deployment-condition.yaml -n ${DASHBOARD_INSTALL_NS}
kubectl apply -f example-pipelines/config/build-task.yaml -n ${DASHBOARD_INSTALL_NS}
kubectl apply -f example-pipelines/config/deploy-task.yaml -n ${DASHBOARD_INSTALL_NS}
kubectl apply -f example-pipelines/config/pipeline.yaml -n ${DASHBOARD_INSTALL_NS}

## Set up webhook
post_data='{
  "name": "knative-demo-test",
  "gitrepositoryurl": "'"${GITHUB_REPO}"'",
  "accesstoken": "github-secret",
  "pipeline": "simple-pipeline",
  "dockerregistry": "'"${DOCKERHUB_USERNAME}"'",
  "namespace": "'"${DASHBOARD_INSTALL_NS}"'"
}'
curl -X POST --header Content-Type:application/json -d "$post_data" http://localhost:31001/v1/extensions/webhooks-extension/webhooks

popd