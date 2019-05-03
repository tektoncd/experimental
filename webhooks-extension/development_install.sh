#!/bin/bash

# Helper function to exit script
function fail() {
    echo "Error: $1."
    exit 1
}

# Set your docker registry and install namespace here so you do not have to
# enter them every time you run this script
DOCKER_REPO=""
NAMESPACE=""

if [ -z $DOCKER_REPO ]; then
    read -p "Enter your docker registry to build and push the images to. For example, your public Dockerhub ID: " DOCKER_REPO
fi
if [ -z $NAMESPACE ]; then
    read -p "Enter the namespace in which to install this webhook extension. This must be the same namespace that the Tekton Dashboard is installed into: " NAMESPACE
fi
echo "DOCKER_REPO: $DOCKER_REPO"
echo "NAMESPACE: $NAMESPACE"
kubectl get namespace "$NAMESPACE" &> /dev/null || fail "the namespace you specified ($NAMESPACE) does not exist"


# Build and push the extension and sink images
docker login

echo "build cmd/extension ..."
docker build -t ${DOCKER_REPO}/extension:latest -f cmd/extension/Dockerfile . || fail "extension docker build failed"
docker push ${DOCKER_REPO}/extension:latest || fail "extension docker push failed"

echo "build cmd/sink ..."
docker build -t ${DOCKER_REPO}/sink:latest -f cmd/sink/Dockerfile . || fail "sink docker build failed"
docker push ${DOCKER_REPO}/sink:latest || fail "sink docker push failed"


# Copy and replace the config/ yaml files into install/
mkdir -p install
cp -r config/ install/
for file in install/*.yaml; do
    sed -i '' 's|github.com/tektoncd/experimental/webhooks-extension/cmd/extension|'"$DOCKER_REPO"'/extension:latest|g' "${file}"
    sed -i '' 's|github.com/tektoncd/experimental/webhooks-extension/cmd/sink|'"$DOCKER_REPO"'/sink:latest|g' "${file}"
done


# Install into the NAMESPACE
echo "install in namespace ${NAMESPACE}"
kubectl apply -f install/ -n ${NAMESPACE} || fail "kubectl apply failed"

echo "the extension and sink yaml files were successfully applied into the $NAMESPACE namespace"
