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

import time
from kubernetes import client, config

from tekton_pipeline.constants import constants
from tekton_pipeline.utils import utils
from tekton_pipeline.api.tekton_watch import watch as tekton_watch


class TektonClient(object):

    def __init__(self, config_file=None, context=None,
                 client_configuration=None, persist_config=True):
        """
        Tekton client constructor
        :param config_file: kubeconfig file, defaults to ~/.kube/config
        :param context: kubernetes context
        :param client_configuration: kubernetes configuration object
        :param persist_config:
        """
        if config_file or not utils.is_running_in_k8s():
            config.load_kube_config(
                config_file=config_file,
                context=context,
                client_configuration=client_configuration,
                persist_config=persist_config)
        else:
            config.load_incluster_config()
        self.core_api = client.CoreV1Api()
        self.app_api = client.AppsV1Api()
        self.api_instance = client.CustomObjectsApi()

    def create(self, entity, body, namespace=None):
        """
        Create the Tekton entity
        :param entity: the tekton entity,  currently supported values: ['task', 'taskrun', 'pipeline', 'pipelinerun', 'clustertask']. 
        :param body: Tekton entity body
        :param namespace: defaults to current or default namespace
        :return: created Tekton entity
        """
        utils.check_entity(entity)

        if namespace is None:
            namespace = utils.get_tekton_namespace(body)

        plural = str(entity).lower() + "s"

        try:
            outputs = self.api_instance.create_namespaced_custom_object(
                group=constants.TEKTON_GROUP,
                version=constants.PIPELINERESOURCE_VERSION if plural == 'pipelineresource' else constants.TEKTON_VERSION,
                namespace=namespace,
                plural=plural,
                body=body)
        except client.rest.ApiException as e:
            raise RuntimeError(
                "Exception when calling CustomObjectsApi->create_namespaced_custom_object:\
                 %s\n" % e)

        return outputs

    def get(self, entity, name, namespace=None, watch=False, timeout_seconds=600):
        """
        Get the Tekton objects
        :param entity: the tekton entity, currently supported values: ['task', 'taskrun', 'pipeline', 'pipelinerun'].
        :param name: existing Tekton objects
        :param namespace: defaults to current or default namespace
        :return: Tekton objects
        """

        utils.check_entity(entity)

        if namespace is None:
            namespace = utils.get_default_target_namespace()

        plural = str(entity).lower() + "s"

        if watch:
            tekton_watch(
                name=name,
                plural=plural,
                namespace=namespace,
                timeout_seconds=timeout_seconds)
        else:
            try:
                return self.api_instance.get_namespaced_custom_object(
                    group=constants.TEKTON_GROUP,
                    version=constants.PIPELINERESOURCE_VERSION if plural == 'pipelineresource' else constants.TEKTON_VERSION,
                    namespace=namespace,
                    plural=plural,
                    name=name)
            except client.rest.ApiException as e:
                raise RuntimeError(
                    "Exception when calling CustomObjectsApi->get_namespaced_custom_object:\
                    %s\n" % e)

    def patch(self, entity, name, body, namespace=None):
        """
        Patch existing tekton object
        :param entity: the tekton entity, currently supported values: ['task', 'taskrun', 'pipeline', 'pipelinerun', 'clustertask'].
        :param name: existing tekton object name
        :param body: patched tekton object
        :param namespace: defaults to current or default namespace
        :return: patched tekton object
        """

        utils.check_entity(entity)

        if namespace is None:
            namespace = utils.get_tekton_namespace(body)

        plural = str(entity).lower() + "s"

        try:
            return self.api_instance.patch_namespaced_custom_object(
                group=constants.TEKTON_GROUP,
                version=constants.PIPELINERESOURCE_VERSION if plural == 'pipelineresource' else constants.TEKTON_VERSION,
                namespace=namespace,
                plural=plural,
                name=name,
                body=body)
        except client.rest.ApiException as e:
            raise RuntimeError(
                "Exception when calling CustomObjectsApi->patch_namespaced_custom_object:\
                 %s\n" % e)

    def delete(self, entity, name, namespace=None):
        """
        Delete the Tekton objects
        :param entity: the tekton entity, currently supported values: ['task', 'taskrun', 'pipeline', 'pipelinerun', 'clustertask'].
        :param name: Tekton object's name
        :param namespace: defaults to current or default namespace
        :return:
        """
        utils.check_entity(entity)

        if namespace is None:
            namespace = utils.get_default_target_namespace()

        plural = str(entity).lower() + "s"

        try:
            return self.api_instance.delete_namespaced_custom_object(
                group=constants.TEKTON_GROUP,
                version=constants.PIPELINERESOURCE_VERSION if plural == 'pipelineresource' else constants.TEKTON_VERSION,
                namespace=namespace,
                plural=plural,
                name=name)
        except client.rest.ApiException as e:
            raise RuntimeError(
                "Exception when calling CustomObjectsApi->delete_namespaced_custom_object:\
                 %s\n" % e)
