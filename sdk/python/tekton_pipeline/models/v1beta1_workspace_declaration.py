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


class V1beta1WorkspaceDeclaration(object):
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
        'description': 'str',
        'mount_path': 'str',
        'name': 'str',
        'optional': 'bool',
        'read_only': 'bool'
    }

    attribute_map = {
        'description': 'description',
        'mount_path': 'mountPath',
        'name': 'name',
        'optional': 'optional',
        'read_only': 'readOnly'
    }

    def __init__(self, description=None, mount_path=None, name='', optional=None, read_only=None, local_vars_configuration=None):  # noqa: E501
        """V1beta1WorkspaceDeclaration - a model defined in OpenAPI"""  # noqa: E501
        if local_vars_configuration is None:
            local_vars_configuration = Configuration()
        self.local_vars_configuration = local_vars_configuration

        self._description = None
        self._mount_path = None
        self._name = None
        self._optional = None
        self._read_only = None
        self.discriminator = None

        if description is not None:
            self.description = description
        if mount_path is not None:
            self.mount_path = mount_path
        self.name = name
        if optional is not None:
            self.optional = optional
        if read_only is not None:
            self.read_only = read_only

    @property
    def description(self):
        """Gets the description of this V1beta1WorkspaceDeclaration.  # noqa: E501

        Description is an optional human readable description of this volume.  # noqa: E501

        :return: The description of this V1beta1WorkspaceDeclaration.  # noqa: E501
        :rtype: str
        """
        return self._description

    @description.setter
    def description(self, description):
        """Sets the description of this V1beta1WorkspaceDeclaration.

        Description is an optional human readable description of this volume.  # noqa: E501

        :param description: The description of this V1beta1WorkspaceDeclaration.  # noqa: E501
        :type: str
        """

        self._description = description

    @property
    def mount_path(self):
        """Gets the mount_path of this V1beta1WorkspaceDeclaration.  # noqa: E501

        MountPath overrides the directory that the volume will be made available at.  # noqa: E501

        :return: The mount_path of this V1beta1WorkspaceDeclaration.  # noqa: E501
        :rtype: str
        """
        return self._mount_path

    @mount_path.setter
    def mount_path(self, mount_path):
        """Sets the mount_path of this V1beta1WorkspaceDeclaration.

        MountPath overrides the directory that the volume will be made available at.  # noqa: E501

        :param mount_path: The mount_path of this V1beta1WorkspaceDeclaration.  # noqa: E501
        :type: str
        """

        self._mount_path = mount_path

    @property
    def name(self):
        """Gets the name of this V1beta1WorkspaceDeclaration.  # noqa: E501

        Name is the name by which you can bind the volume at runtime.  # noqa: E501

        :return: The name of this V1beta1WorkspaceDeclaration.  # noqa: E501
        :rtype: str
        """
        return self._name

    @name.setter
    def name(self, name):
        """Sets the name of this V1beta1WorkspaceDeclaration.

        Name is the name by which you can bind the volume at runtime.  # noqa: E501

        :param name: The name of this V1beta1WorkspaceDeclaration.  # noqa: E501
        :type: str
        """
        if self.local_vars_configuration.client_side_validation and name is None:  # noqa: E501
            raise ValueError("Invalid value for `name`, must not be `None`")  # noqa: E501

        self._name = name

    @property
    def optional(self):
        """Gets the optional of this V1beta1WorkspaceDeclaration.  # noqa: E501

        Optional marks a Workspace as not being required in TaskRuns. By default this field is false and so declared workspaces are required.  # noqa: E501

        :return: The optional of this V1beta1WorkspaceDeclaration.  # noqa: E501
        :rtype: bool
        """
        return self._optional

    @optional.setter
    def optional(self, optional):
        """Sets the optional of this V1beta1WorkspaceDeclaration.

        Optional marks a Workspace as not being required in TaskRuns. By default this field is false and so declared workspaces are required.  # noqa: E501

        :param optional: The optional of this V1beta1WorkspaceDeclaration.  # noqa: E501
        :type: bool
        """

        self._optional = optional

    @property
    def read_only(self):
        """Gets the read_only of this V1beta1WorkspaceDeclaration.  # noqa: E501

        ReadOnly dictates whether a mounted volume is writable. By default this field is false and so mounted volumes are writable.  # noqa: E501

        :return: The read_only of this V1beta1WorkspaceDeclaration.  # noqa: E501
        :rtype: bool
        """
        return self._read_only

    @read_only.setter
    def read_only(self, read_only):
        """Sets the read_only of this V1beta1WorkspaceDeclaration.

        ReadOnly dictates whether a mounted volume is writable. By default this field is false and so mounted volumes are writable.  # noqa: E501

        :param read_only: The read_only of this V1beta1WorkspaceDeclaration.  # noqa: E501
        :type: bool
        """

        self._read_only = read_only

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
        if not isinstance(other, V1beta1WorkspaceDeclaration):
            return False

        return self.to_dict() == other.to_dict()

    def __ne__(self, other):
        """Returns true if both objects are not equal"""
        if not isinstance(other, V1beta1WorkspaceDeclaration):
            return True

        return self.to_dict() != other.to_dict()
