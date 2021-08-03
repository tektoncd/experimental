# Copyright 2021 The Tekton Authors
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

    def create(self, tekton, plural=None, namespace=None):
        """
        Create the Tekton objects
        :param tekton: Tekton objects
        :param plural: the custom object's plural name. 
        :param namespace: defaults to current or default namespace
        :return: created Tekton objects
        """

        if namespace is None:
            namespace = utils.get_tekton_namespace(tekton)

        if plural is None:
            plural = utils.get_tekton_plural(tekton)

        try:
            outputs = self.api_instance.create_namespaced_custom_object(
                constants.TEKTON_GROUP,
                constants.TEKTON_VERSION,
                namespace,
                plural,
                tekton)
        except client.rest.ApiException as e:
            raise RuntimeError(
                "Exception when calling CustomObjectsApi->create_namespaced_custom_object:\
                 %s\n" % e)

        return outputs

    def get(self, name, plural, namespace=None):
        """
        Get the Tekton objects
        :param name: existing Tekton objects
        :param plural: the custom object's plural name. 
        :param namespace: defaults to current or default namespace
        :return: Tekton objects
        """
        if namespace is None:
            namespace = utils.get_default_target_namespace()

        try:
            return self.api_instance.get_namespaced_custom_object(
                constants.TEKTON_GROUP,
                constants.TEKTON_VERSION,
                namespace,
                plural,
                name)
        except client.rest.ApiException as e:
            raise RuntimeError(
                "Exception when calling CustomObjectsApi->get_namespaced_custom_object:\
                %s\n" % e)


    def delete(self, name, plural, namespace=None):
        """
        Delete the Tekton objects
        :param name: Tekton object's name
        :param plural: the custom object's plural name. 
        :param namespace: defaults to current or default namespace
        :return:
        """
        if namespace is None:
            namespace = utils.get_default_target_namespace()

        try:
            return self.api_instance.delete_namespaced_custom_object(
                constants.TEKTON_GROUP,
                constants.TEKTON_VERSION,
                namespace,
                plural,
                name)
        except client.rest.ApiException as e:
            raise RuntimeError(
                "Exception when calling CustomObjectsApi->delete_namespaced_custom_object:\
                 %s\n" % e)
