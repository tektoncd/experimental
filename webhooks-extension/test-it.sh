#!/bin/bash

# Workaround since with React as imports, we get no bundle from rollup.
npm install --save-dev react


if [[ $# -eq 1 ]] ; then
  echo "Arg provided to test: ${1}"
  jest ${1}
else
  jest --watchAll
fi

npm uninstall --save-dev react