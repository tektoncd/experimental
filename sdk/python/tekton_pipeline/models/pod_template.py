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

# coding: utf-8

"""
    Tekton

    Tekton Pipeline  # noqa: E501

    The version of the OpenAPI document: v0.17.2
    Generated by: https://openapi-generator.tech
"""


import pprint
import re  # noqa: F401

import six

from tekton_pipeline.configuration import Configuration


class PodTemplate(object):
    """NOTE: This class is auto generated by OpenAPI Generator.
    Ref: https://openapi-generator.tech

    Do not edit the class manually.
    """

    """
    Attributes:
      openapi_types (dict): The key is attribute name
                            and the value is attribute type.
      attribute_map (dict): The key is attribute name
                            and the value is json key in definition.
    """
    openapi_types = {
        'affinity': 'V1Affinity',
        'automount_service_account_token': 'bool',
        'dns_config': 'V1PodDNSConfig',
        'dns_policy': 'str',
        'enable_service_links': 'bool',
        'host_aliases': 'list[V1HostAlias]',
        'host_network': 'bool',
        'image_pull_secrets': 'list[V1LocalObjectReference]',
        'node_selector': 'dict(str, str)',
        'priority_class_name': 'str',
        'runtime_class_name': 'str',
        'scheduler_name': 'str',
        'security_context': 'V1PodSecurityContext',
        'tolerations': 'list[V1Toleration]',
        'volumes': 'list[V1Volume]'
    }

    attribute_map = {
        'affinity': 'affinity',
        'automount_service_account_token': 'automountServiceAccountToken',
        'dns_config': 'dnsConfig',
        'dns_policy': 'dnsPolicy',
        'enable_service_links': 'enableServiceLinks',
        'host_aliases': 'hostAliases',
        'host_network': 'hostNetwork',
        'image_pull_secrets': 'imagePullSecrets',
        'node_selector': 'nodeSelector',
        'priority_class_name': 'priorityClassName',
        'runtime_class_name': 'runtimeClassName',
        'scheduler_name': 'schedulerName',
        'security_context': 'securityContext',
        'tolerations': 'tolerations',
        'volumes': 'volumes'
    }

    def __init__(self, affinity=None, automount_service_account_token=None, dns_config=None, dns_policy=None, enable_service_links=None, host_aliases=None, host_network=None, image_pull_secrets=None, node_selector=None, priority_class_name=None, runtime_class_name=None, scheduler_name=None, security_context=None, tolerations=None, volumes=None, local_vars_configuration=None):  # noqa: E501
        """PodTemplate - a model defined in OpenAPI"""  # noqa: E501
        if local_vars_configuration is None:
            local_vars_configuration = Configuration()
        self.local_vars_configuration = local_vars_configuration

        self._affinity = None
        self._automount_service_account_token = None
        self._dns_config = None
        self._dns_policy = None
        self._enable_service_links = None
        self._host_aliases = None
        self._host_network = None
        self._image_pull_secrets = None
        self._node_selector = None
        self._priority_class_name = None
        self._runtime_class_name = None
        self._scheduler_name = None
        self._security_context = None
        self._tolerations = None
        self._volumes = None
        self.discriminator = None

        if affinity is not None:
            self.affinity = affinity
        if automount_service_account_token is not None:
            self.automount_service_account_token = automount_service_account_token
        if dns_config is not None:
            self.dns_config = dns_config
        if dns_policy is not None:
            self.dns_policy = dns_policy
        if enable_service_links is not None:
            self.enable_service_links = enable_service_links
        if host_aliases is not None:
            self.host_aliases = host_aliases
        if host_network is not None:
            self.host_network = host_network
        if image_pull_secrets is not None:
            self.image_pull_secrets = image_pull_secrets
        if node_selector is not None:
            self.node_selector = node_selector
        if priority_class_name is not None:
            self.priority_class_name = priority_class_name
        if runtime_class_name is not None:
            self.runtime_class_name = runtime_class_name
        if scheduler_name is not None:
            self.scheduler_name = scheduler_name
        if security_context is not None:
            self.security_context = security_context
        if tolerations is not None:
            self.tolerations = tolerations
        if volumes is not None:
            self.volumes = volumes

    @property
    def affinity(self):
        """Gets the affinity of this PodTemplate.  # noqa: E501


        :return: The affinity of this PodTemplate.  # noqa: E501
        :rtype: V1Affinity
        """
        return self._affinity

    @affinity.setter
    def affinity(self, affinity):
        """Sets the affinity of this PodTemplate.


        :param affinity: The affinity of this PodTemplate.  # noqa: E501
        :type: V1Affinity
        """

        self._affinity = affinity

    @property
    def automount_service_account_token(self):
        """Gets the automount_service_account_token of this PodTemplate.  # noqa: E501

        AutomountServiceAccountToken indicates whether pods running as this service account should have an API token automatically mounted.  # noqa: E501

        :return: The automount_service_account_token of this PodTemplate.  # noqa: E501
        :rtype: bool
        """
        return self._automount_service_account_token

    @automount_service_account_token.setter
    def automount_service_account_token(self, automount_service_account_token):
        """Sets the automount_service_account_token of this PodTemplate.

        AutomountServiceAccountToken indicates whether pods running as this service account should have an API token automatically mounted.  # noqa: E501

        :param automount_service_account_token: The automount_service_account_token of this PodTemplate.  # noqa: E501
        :type: bool
        """

        self._automount_service_account_token = automount_service_account_token

    @property
    def dns_config(self):
        """Gets the dns_config of this PodTemplate.  # noqa: E501


        :return: The dns_config of this PodTemplate.  # noqa: E501
        :rtype: V1PodDNSConfig
        """
        return self._dns_config

    @dns_config.setter
    def dns_config(self, dns_config):
        """Sets the dns_config of this PodTemplate.


        :param dns_config: The dns_config of this PodTemplate.  # noqa: E501
        :type: V1PodDNSConfig
        """

        self._dns_config = dns_config

    @property
    def dns_policy(self):
        """Gets the dns_policy of this PodTemplate.  # noqa: E501

        Set DNS policy for the pod. Defaults to \"ClusterFirst\". Valid values are 'ClusterFirst', 'Default' or 'None'. DNS parameters given in DNSConfig will be merged with the policy selected with DNSPolicy.  # noqa: E501

        :return: The dns_policy of this PodTemplate.  # noqa: E501
        :rtype: str
        """
        return self._dns_policy

    @dns_policy.setter
    def dns_policy(self, dns_policy):
        """Sets the dns_policy of this PodTemplate.

        Set DNS policy for the pod. Defaults to \"ClusterFirst\". Valid values are 'ClusterFirst', 'Default' or 'None'. DNS parameters given in DNSConfig will be merged with the policy selected with DNSPolicy.  # noqa: E501

        :param dns_policy: The dns_policy of this PodTemplate.  # noqa: E501
        :type: str
        """

        self._dns_policy = dns_policy

    @property
    def enable_service_links(self):
        """Gets the enable_service_links of this PodTemplate.  # noqa: E501

        EnableServiceLinks indicates whether information about services should be injected into pod's environment variables, matching the syntax of Docker links. Optional: Defaults to true.  # noqa: E501

        :return: The enable_service_links of this PodTemplate.  # noqa: E501
        :rtype: bool
        """
        return self._enable_service_links

    @enable_service_links.setter
    def enable_service_links(self, enable_service_links):
        """Sets the enable_service_links of this PodTemplate.

        EnableServiceLinks indicates whether information about services should be injected into pod's environment variables, matching the syntax of Docker links. Optional: Defaults to true.  # noqa: E501

        :param enable_service_links: The enable_service_links of this PodTemplate.  # noqa: E501
        :type: bool
        """

        self._enable_service_links = enable_service_links

    @property
    def host_aliases(self):
        """Gets the host_aliases of this PodTemplate.  # noqa: E501

        HostAliases is an optional list of hosts and IPs that will be injected into the pod's hosts file if specified. This is only valid for non-hostNetwork pods.  # noqa: E501

        :return: The host_aliases of this PodTemplate.  # noqa: E501
        :rtype: list[V1HostAlias]
        """
        return self._host_aliases

    @host_aliases.setter
    def host_aliases(self, host_aliases):
        """Sets the host_aliases of this PodTemplate.

        HostAliases is an optional list of hosts and IPs that will be injected into the pod's hosts file if specified. This is only valid for non-hostNetwork pods.  # noqa: E501

        :param host_aliases: The host_aliases of this PodTemplate.  # noqa: E501
        :type: list[V1HostAlias]
        """

        self._host_aliases = host_aliases

    @property
    def host_network(self):
        """Gets the host_network of this PodTemplate.  # noqa: E501

        HostNetwork specifies whether the pod may use the node network namespace  # noqa: E501

        :return: The host_network of this PodTemplate.  # noqa: E501
        :rtype: bool
        """
        return self._host_network

    @host_network.setter
    def host_network(self, host_network):
        """Sets the host_network of this PodTemplate.

        HostNetwork specifies whether the pod may use the node network namespace  # noqa: E501

        :param host_network: The host_network of this PodTemplate.  # noqa: E501
        :type: bool
        """

        self._host_network = host_network

    @property
    def image_pull_secrets(self):
        """Gets the image_pull_secrets of this PodTemplate.  # noqa: E501

        ImagePullSecrets gives the name of the secret used by the pod to pull the image if specified  # noqa: E501

        :return: The image_pull_secrets of this PodTemplate.  # noqa: E501
        :rtype: list[V1LocalObjectReference]
        """
        return self._image_pull_secrets

    @image_pull_secrets.setter
    def image_pull_secrets(self, image_pull_secrets):
        """Sets the image_pull_secrets of this PodTemplate.

        ImagePullSecrets gives the name of the secret used by the pod to pull the image if specified  # noqa: E501

        :param image_pull_secrets: The image_pull_secrets of this PodTemplate.  # noqa: E501
        :type: list[V1LocalObjectReference]
        """

        self._image_pull_secrets = image_pull_secrets

    @property
    def node_selector(self):
        """Gets the node_selector of this PodTemplate.  # noqa: E501

        NodeSelector is a selector which must be true for the pod to fit on a node. Selector which must match a node's labels for the pod to be scheduled on that node. More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/  # noqa: E501

        :return: The node_selector of this PodTemplate.  # noqa: E501
        :rtype: dict(str, str)
        """
        return self._node_selector

    @node_selector.setter
    def node_selector(self, node_selector):
        """Sets the node_selector of this PodTemplate.

        NodeSelector is a selector which must be true for the pod to fit on a node. Selector which must match a node's labels for the pod to be scheduled on that node. More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/  # noqa: E501

        :param node_selector: The node_selector of this PodTemplate.  # noqa: E501
        :type: dict(str, str)
        """

        self._node_selector = node_selector

    @property
    def priority_class_name(self):
        """Gets the priority_class_name of this PodTemplate.  # noqa: E501

        If specified, indicates the pod's priority. \"system-node-critical\" and \"system-cluster-critical\" are two special keywords which indicate the highest priorities with the former being the highest priority. Any other name must be defined by creating a PriorityClass object with that name. If not specified, the pod priority will be default or zero if there is no default.  # noqa: E501

        :return: The priority_class_name of this PodTemplate.  # noqa: E501
        :rtype: str
        """
        return self._priority_class_name

    @priority_class_name.setter
    def priority_class_name(self, priority_class_name):
        """Sets the priority_class_name of this PodTemplate.

        If specified, indicates the pod's priority. \"system-node-critical\" and \"system-cluster-critical\" are two special keywords which indicate the highest priorities with the former being the highest priority. Any other name must be defined by creating a PriorityClass object with that name. If not specified, the pod priority will be default or zero if there is no default.  # noqa: E501

        :param priority_class_name: The priority_class_name of this PodTemplate.  # noqa: E501
        :type: str
        """

        self._priority_class_name = priority_class_name

    @property
    def runtime_class_name(self):
        """Gets the runtime_class_name of this PodTemplate.  # noqa: E501

        RuntimeClassName refers to a RuntimeClass object in the node.k8s.io group, which should be used to run this pod. If no RuntimeClass resource matches the named class, the pod will not be run. If unset or empty, the \"legacy\" RuntimeClass will be used, which is an implicit class with an empty definition that uses the default runtime handler. More info: https://git.k8s.io/enhancements/keps/sig-node/runtime-class.md This is a beta feature as of Kubernetes v1.14.  # noqa: E501

        :return: The runtime_class_name of this PodTemplate.  # noqa: E501
        :rtype: str
        """
        return self._runtime_class_name

    @runtime_class_name.setter
    def runtime_class_name(self, runtime_class_name):
        """Sets the runtime_class_name of this PodTemplate.

        RuntimeClassName refers to a RuntimeClass object in the node.k8s.io group, which should be used to run this pod. If no RuntimeClass resource matches the named class, the pod will not be run. If unset or empty, the \"legacy\" RuntimeClass will be used, which is an implicit class with an empty definition that uses the default runtime handler. More info: https://git.k8s.io/enhancements/keps/sig-node/runtime-class.md This is a beta feature as of Kubernetes v1.14.  # noqa: E501

        :param runtime_class_name: The runtime_class_name of this PodTemplate.  # noqa: E501
        :type: str
        """

        self._runtime_class_name = runtime_class_name

    @property
    def scheduler_name(self):
        """Gets the scheduler_name of this PodTemplate.  # noqa: E501

        SchedulerName specifies the scheduler to be used to dispatch the Pod  # noqa: E501

        :return: The scheduler_name of this PodTemplate.  # noqa: E501
        :rtype: str
        """
        return self._scheduler_name

    @scheduler_name.setter
    def scheduler_name(self, scheduler_name):
        """Sets the scheduler_name of this PodTemplate.

        SchedulerName specifies the scheduler to be used to dispatch the Pod  # noqa: E501

        :param scheduler_name: The scheduler_name of this PodTemplate.  # noqa: E501
        :type: str
        """

        self._scheduler_name = scheduler_name

    @property
    def security_context(self):
        """Gets the security_context of this PodTemplate.  # noqa: E501


        :return: The security_context of this PodTemplate.  # noqa: E501
        :rtype: V1PodSecurityContext
        """
        return self._security_context

    @security_context.setter
    def security_context(self, security_context):
        """Sets the security_context of this PodTemplate.


        :param security_context: The security_context of this PodTemplate.  # noqa: E501
        :type: V1PodSecurityContext
        """

        self._security_context = security_context

    @property
    def tolerations(self):
        """Gets the tolerations of this PodTemplate.  # noqa: E501

        If specified, the pod's tolerations.  # noqa: E501

        :return: The tolerations of this PodTemplate.  # noqa: E501
        :rtype: list[V1Toleration]
        """
        return self._tolerations

    @tolerations.setter
    def tolerations(self, tolerations):
        """Sets the tolerations of this PodTemplate.

        If specified, the pod's tolerations.  # noqa: E501

        :param tolerations: The tolerations of this PodTemplate.  # noqa: E501
        :type: list[V1Toleration]
        """

        self._tolerations = tolerations

    @property
    def volumes(self):
        """Gets the volumes of this PodTemplate.  # noqa: E501

        List of volumes that can be mounted by containers belonging to the pod. More info: https://kubernetes.io/docs/concepts/storage/volumes  # noqa: E501

        :return: The volumes of this PodTemplate.  # noqa: E501
        :rtype: list[V1Volume]
        """
        return self._volumes

    @volumes.setter
    def volumes(self, volumes):
        """Sets the volumes of this PodTemplate.

        List of volumes that can be mounted by containers belonging to the pod. More info: https://kubernetes.io/docs/concepts/storage/volumes  # noqa: E501

        :param volumes: The volumes of this PodTemplate.  # noqa: E501
        :type: list[V1Volume]
        """

        self._volumes = volumes

    def to_dict(self):
        """Returns the model properties as a dict"""
        result = {}

        for attr, _ in six.iteritems(self.openapi_types):
            value = getattr(self, attr)
            if isinstance(value, list):
                result[attr] = list(map(
                    lambda x: x.to_dict() if hasattr(x, "to_dict") else x,
                    value
                ))
            elif hasattr(value, "to_dict"):
                result[attr] = value.to_dict()
            elif isinstance(value, dict):
                result[attr] = dict(map(
                    lambda item: (item[0], item[1].to_dict())
                    if hasattr(item[1], "to_dict") else item,
                    value.items()
                ))
            else:
                result[attr] = value

        return result

    def to_str(self):
        """Returns the string representation of the model"""
        return pprint.pformat(self.to_dict())

    def __repr__(self):
        """For `print` and `pprint`"""
        return self.to_str()

    def __eq__(self, other):
        """Returns true if both objects are equal"""
        if not isinstance(other, PodTemplate):
            return False

        return self.to_dict() == other.to_dict()

    def __ne__(self, other):
        """Returns true if both objects are not equal"""
        if not isinstance(other, PodTemplate):
            return True

        return self.to_dict() != other.to_dict()
