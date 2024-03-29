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


class V1beta1WorkspaceBinding(object):
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
        'config_map': 'V1ConfigMapVolumeSource',
        'empty_dir': 'V1EmptyDirVolumeSource',
        'name': 'str',
        'persistent_volume_claim': 'V1PersistentVolumeClaimVolumeSource',
        'secret': 'V1SecretVolumeSource',
        'sub_path': 'str',
        'volume_claim_template': 'V1PersistentVolumeClaim'
    }

    attribute_map = {
        'config_map': 'configMap',
        'empty_dir': 'emptyDir',
        'name': 'name',
        'persistent_volume_claim': 'persistentVolumeClaim',
        'secret': 'secret',
        'sub_path': 'subPath',
        'volume_claim_template': 'volumeClaimTemplate'
    }

    def __init__(self, config_map=None, empty_dir=None, name='', persistent_volume_claim=None, secret=None, sub_path=None, volume_claim_template=None, local_vars_configuration=None):  # noqa: E501
        """V1beta1WorkspaceBinding - a model defined in OpenAPI"""  # noqa: E501
        if local_vars_configuration is None:
            local_vars_configuration = Configuration()
        self.local_vars_configuration = local_vars_configuration

        self._config_map = None
        self._empty_dir = None
        self._name = None
        self._persistent_volume_claim = None
        self._secret = None
        self._sub_path = None
        self._volume_claim_template = None
        self.discriminator = None

        if config_map is not None:
            self.config_map = config_map
        if empty_dir is not None:
            self.empty_dir = empty_dir
        self.name = name
        if persistent_volume_claim is not None:
            self.persistent_volume_claim = persistent_volume_claim
        if secret is not None:
            self.secret = secret
        if sub_path is not None:
            self.sub_path = sub_path
        if volume_claim_template is not None:
            self.volume_claim_template = volume_claim_template

    @property
    def config_map(self):
        """Gets the config_map of this V1beta1WorkspaceBinding.  # noqa: E501


        :return: The config_map of this V1beta1WorkspaceBinding.  # noqa: E501
        :rtype: V1ConfigMapVolumeSource
        """
        return self._config_map

    @config_map.setter
    def config_map(self, config_map):
        """Sets the config_map of this V1beta1WorkspaceBinding.


        :param config_map: The config_map of this V1beta1WorkspaceBinding.  # noqa: E501
        :type: V1ConfigMapVolumeSource
        """

        self._config_map = config_map

    @property
    def empty_dir(self):
        """Gets the empty_dir of this V1beta1WorkspaceBinding.  # noqa: E501


        :return: The empty_dir of this V1beta1WorkspaceBinding.  # noqa: E501
        :rtype: V1EmptyDirVolumeSource
        """
        return self._empty_dir

    @empty_dir.setter
    def empty_dir(self, empty_dir):
        """Sets the empty_dir of this V1beta1WorkspaceBinding.


        :param empty_dir: The empty_dir of this V1beta1WorkspaceBinding.  # noqa: E501
        :type: V1EmptyDirVolumeSource
        """

        self._empty_dir = empty_dir

    @property
    def name(self):
        """Gets the name of this V1beta1WorkspaceBinding.  # noqa: E501

        Name is the name of the workspace populated by the volume.  # noqa: E501

        :return: The name of this V1beta1WorkspaceBinding.  # noqa: E501
        :rtype: str
        """
        return self._name

    @name.setter
    def name(self, name):
        """Sets the name of this V1beta1WorkspaceBinding.

        Name is the name of the workspace populated by the volume.  # noqa: E501

        :param name: The name of this V1beta1WorkspaceBinding.  # noqa: E501
        :type: str
        """
        if self.local_vars_configuration.client_side_validation and name is None:  # noqa: E501
            raise ValueError("Invalid value for `name`, must not be `None`")  # noqa: E501

        self._name = name

    @property
    def persistent_volume_claim(self):
        """Gets the persistent_volume_claim of this V1beta1WorkspaceBinding.  # noqa: E501


        :return: The persistent_volume_claim of this V1beta1WorkspaceBinding.  # noqa: E501
        :rtype: V1PersistentVolumeClaimVolumeSource
        """
        return self._persistent_volume_claim

    @persistent_volume_claim.setter
    def persistent_volume_claim(self, persistent_volume_claim):
        """Sets the persistent_volume_claim of this V1beta1WorkspaceBinding.


        :param persistent_volume_claim: The persistent_volume_claim of this V1beta1WorkspaceBinding.  # noqa: E501
        :type: V1PersistentVolumeClaimVolumeSource
        """

        self._persistent_volume_claim = persistent_volume_claim

    @property
    def secret(self):
        """Gets the secret of this V1beta1WorkspaceBinding.  # noqa: E501


        :return: The secret of this V1beta1WorkspaceBinding.  # noqa: E501
        :rtype: V1SecretVolumeSource
        """
        return self._secret

    @secret.setter
    def secret(self, secret):
        """Sets the secret of this V1beta1WorkspaceBinding.


        :param secret: The secret of this V1beta1WorkspaceBinding.  # noqa: E501
        :type: V1SecretVolumeSource
        """

        self._secret = secret

    @property
    def sub_path(self):
        """Gets the sub_path of this V1beta1WorkspaceBinding.  # noqa: E501

        SubPath is optionally a directory on the volume which should be used for this binding (i.e. the volume will be mounted at this sub directory).  # noqa: E501

        :return: The sub_path of this V1beta1WorkspaceBinding.  # noqa: E501
        :rtype: str
        """
        return self._sub_path

    @sub_path.setter
    def sub_path(self, sub_path):
        """Sets the sub_path of this V1beta1WorkspaceBinding.

        SubPath is optionally a directory on the volume which should be used for this binding (i.e. the volume will be mounted at this sub directory).  # noqa: E501

        :param sub_path: The sub_path of this V1beta1WorkspaceBinding.  # noqa: E501
        :type: str
        """

        self._sub_path = sub_path

    @property
    def volume_claim_template(self):
        """Gets the volume_claim_template of this V1beta1WorkspaceBinding.  # noqa: E501


        :return: The volume_claim_template of this V1beta1WorkspaceBinding.  # noqa: E501
        :rtype: V1PersistentVolumeClaim
        """
        return self._volume_claim_template

    @volume_claim_template.setter
    def volume_claim_template(self, volume_claim_template):
        """Sets the volume_claim_template of this V1beta1WorkspaceBinding.


        :param volume_claim_template: The volume_claim_template of this V1beta1WorkspaceBinding.  # noqa: E501
        :type: V1PersistentVolumeClaim
        """

        self._volume_claim_template = volume_claim_template

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
        if not isinstance(other, V1beta1WorkspaceBinding):
            return False

        return self.to_dict() == other.to_dict()

    def __ne__(self, other):
        """Returns true if both objects are not equal"""
        if not isinstance(other, V1beta1WorkspaceBinding):
            return True

        return self.to_dict() != other.to_dict()
