#!/bin/bash

ko delete -f config

rm -rf ./dist
rm -rf ./cmd/extension/kodata

npm run build
npm run build_ko

echo "Modify your yaml now: sleeping for 10 seconds"
sleep 10

ko apply -f config

kubectl delete pod -l app=tekton-dashboard
echo "Now port-forward, hints below"

echo "kubectl port-forward $(kubectl get pod -l app=webhooks-extension -o name) 8080:8080"
echo "kubectl port-forward $(kubectl get pod -l app=tekton-dashboard -o name) 9097:9097"