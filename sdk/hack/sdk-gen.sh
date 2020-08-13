#!/usr/bin/env bash

# Copyright 2020 The Tekton Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -o errexit
set -o nounset

OPENAPI_GEN_URL="https://repo1.maven.org/maven2/org/openapitools/openapi-generator-cli/4.3.1/openapi-generator-cli-4.3.1.jar"
OPENAPI_GEN_JAR="sdk/hack/openapi-generator-cli.jar"
SWAGGER_CODEGEN_CONF="sdk/hack/swagger_config.json"
SWAGGER_CODEGEN_FILE="sdk/hack/swagger.json"
SWAGGER_CODEGEN_SOURCE="https://github.com/tektoncd/pipeline/tree/master/pkg/apis/pipeline/v1beta1/swagger.json"
SDK_OUTPUT_PATH="./sdk/python"

echo "Check the swagger.json file ..."
if [ ! -f ${SWAGGER_CODEGEN_FILE} ]
then
    wget -O ${SWAGGER_CODEGEN_FILE} ${SWAGGER_CODEGEN_SOURCE}
fi

echo "Downloading the swagger-codegen JAR package ..."
if [ ! -f ${OPENAPI_GEN_JAR} ]
then
    wget -O ${OPENAPI_GEN_JAR} ${OPENAPI_GEN_URL}
fi

echo "Generating Python SDK for Tekton Pipeline ..."
java -jar ${OPENAPI_GEN_JAR} generate -i ${SWAGGER_CODEGEN_FILE} -g python -o ${SDK_OUTPUT_PATH} -c ${SWAGGER_CODEGEN_CONF}

echo "Adding Python boilerplate message ..."
for i in $(find ./sdk/python -name *.py)
do
  if ! grep -q Copyright $i
  then
    cat sdk/hack/boilerplate.python.txt $i >$i.new && mv $i.new $i
  fi
done

echo "Replace Kubernetes document link ..."
MAPPING_LIST=`grep V1 ${SWAGGER_CODEGEN_CONF} |awk -F '"' '{print $2}'`
K8S_URL='https://github.com/kubernetes-client/python/blob/master/kubernetes/docs'
for map in ${MAPPING_LIST}
do
   sed -i'.bak' -e "s@($map.md)@($K8S_URL/$map.md)@g" ./sdk/python/docs/*
   rm -rf ./sdk/python/docs/*.bak
done

echo "Update some specify files ..."
git checkout ${SDK_OUTPUT_PATH}/setup.py
git checkout ${SDK_OUTPUT_PATH}/requirements.txt
# Better to merge README file munally.
#git checkout ${SDK_OUTPUT_PATH}/README.md

if ! grep -q "TektonClient" ${SDK_OUTPUT_PATH}/tekton/__init__.py
then
echo "from tekton.api.tekton_client import TektonClient" >> ${SDK_OUTPUT_PATH}/tekton/__init__.py
fi

if ! grep -q "constants" ${SDK_OUTPUT_PATH}/tekton/__init__.py
then
echo "from tekton.constants import constants" >> ${SDK_OUTPUT_PATH}/tekton/__init__.py
fi

echo "Tekton Pipeline Python SDK is generated successfully to folder ${SDK_OUTPUT_PATH}/."


