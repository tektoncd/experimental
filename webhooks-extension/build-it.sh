#!/bin/bash

if ! [ -x "$(command -v yq)" ]; then
  echo 'Error: yq is not installed (hint: brew install yq)' >&2
  exit 1
fi

npm uninstall --save-dev react

if [[ $1 ]]
then
  rollup -c --dir $1
else 
  rollup -c
fi

npm install --save-dev react

thePath=""

kodataPath="cmd/extension/kodata"
distPath="dist"

foundIt=false
if [[ -e $kodataPath ]] ; then
  thePath=$kodataPath
  foundIt=true
elif [[ -e $distPath ]] ; then
  thePath=$distPath
  foundIt=true
fi

if [[ ! foundIt ]] ; then
  echo "Didn't find the bundle in either $kodataPath or $distPath, can't update anything"
  exit -1
fi

serviceFile="base/300-extension-service.yaml"
newestFileInFolder=$(ls -Art $thePath | tail -n 1)
echo "Thinking the newest file (your build?) is $newestFileInFolder"
foundHash=$(echo $thePath/$newestFileInFolder | awk -F "." '{print $2}')
echo "Thinking the hash to use is $foundHash"
toUse="web/extension.$foundHash.js"
yq w -i $serviceFile metadata.annotations.tekton-dashboard-bundle-location $toUse
echo "Done!"