#!/bin/bash

# Install https://github.com/tektoncd/experimental/tree/master/webhooks-extension

if [ -z "$1" ]; then
  echo "Usage ERROR for script: install_webhooks_extension [target-namespace]"
  echo "Missing [namespace]"
  exit 1
fi 
namespace=$1

npm install
npm rebuild node-sass
npm run build_ko
dep ensure -v

sed -i "" "/value:/ s/$/$namespace/" config/sink-kservice.yaml
ko apply -f config -n $namespace
kubectl get pods -n $namespace
sed -i "" -e "25s/$namespace//" config/sink-kservice.yaml

# Docker desktop: cluster IP = host IP. Not the case for other types of cluster. 
echo ip=$(ifconfig | grep netmask | sed -n 2p | cut -d ' ' -f2)
ipCorrect=true
read -p "Continue with this cluster IP? (y/n)?" choice
case "$choice" in 
  y|Y ) echo "Continuing...";;
  n|N )
    echo "Please set ip manually before continuing"
    ipCorrect=false
    ;;
  * ) echo "invalid input";;
esac

if [ ! $ipCorrect ]
  then 
    read -p "Press y when ip has been set" choice
    case "$choice" in 
      y|Y ) echo "Continuing...";;
      * )
        echo "exiting script"
        exit
        ;;
    esac
fi

kubectl patch configmap config-domain --namespace knative-serving --type='json' \
  --patch '[{"op": "add", "path": "/data/'"${ip}.nip.io"'", "value": ""}]'
