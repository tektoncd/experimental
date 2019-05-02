#!/bin/bash

npm uninstall --save-dev react
if [[ $1 ]]
then
  rollup -c --dir $1
else 
  rollup -c 
fi
npm install --save-dev react
