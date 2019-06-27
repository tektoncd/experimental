#!/bin/bash

# Workaround since with React as imports, we get no bundle from rollup.
exit_code=0
npm install --save-dev react

if [[ $# -eq 1 ]] ; then
  echo "Arg provided to test: ${1}"
  if [[ ${1} == "ci" ]] ; then
    jest
    exit_code=$?
  else 
    jest ${1}
    exit_code=$?
  fi
else
  jest --watchAll
fi

npm uninstall --save-dev react
exit $exit_code 