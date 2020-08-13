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

from tekton.configuration import Configuration


class V1beta1WhenExpression(object):
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
        'input': 'str',
        'operator': 'str',
        'values': 'list[str]',
        'input': 'str',
        'operator': 'str',
        'values': 'list[str]'
    }

    attribute_map = {
        'input': 'Input',
        'operator': 'Operator',
        'values': 'Values',
        'input': 'input',
        'operator': 'operator',
        'values': 'values'
    }

    def __init__(self, input=None, operator=None, values=None, input=None, operator=None, values=None, local_vars_configuration=None):  # noqa: E501
        """V1beta1WhenExpression - a model defined in OpenAPI"""  # noqa: E501
        if local_vars_configuration is None:
            local_vars_configuration = Configuration()
        self.local_vars_configuration = local_vars_configuration

        self._input = None
        self._operator = None
        self._values = None
        self._input = None
        self._operator = None
        self._values = None
        self.discriminator = None

        if input is not None:
            self.input = input
        if operator is not None:
            self.operator = operator
        if values is not None:
            self.values = values
        self.input = input
        self.operator = operator
        self.values = values

    @property
    def input(self):
        """Gets the input of this V1beta1WhenExpression.  # noqa: E501

        DeprecatedInput for backwards compatibility with <v0.17 it is the string for guard checking which can be a static input or an output from a parent Task  # noqa: E501

        :return: The input of this V1beta1WhenExpression.  # noqa: E501
        :rtype: str
        """
        return self._input

    @input.setter
    def input(self, input):
        """Sets the input of this V1beta1WhenExpression.

        DeprecatedInput for backwards compatibility with <v0.17 it is the string for guard checking which can be a static input or an output from a parent Task  # noqa: E501

        :param input: The input of this V1beta1WhenExpression.  # noqa: E501
        :type: str
        """

        self._input = input

    @property
    def operator(self):
        """Gets the operator of this V1beta1WhenExpression.  # noqa: E501

        DeprecatedOperator for backwards compatibility with <v0.17 it represents a DeprecatedInput's relationship to the DeprecatedValues  # noqa: E501

        :return: The operator of this V1beta1WhenExpression.  # noqa: E501
        :rtype: str
        """
        return self._operator

    @operator.setter
    def operator(self, operator):
        """Sets the operator of this V1beta1WhenExpression.

        DeprecatedOperator for backwards compatibility with <v0.17 it represents a DeprecatedInput's relationship to the DeprecatedValues  # noqa: E501

        :param operator: The operator of this V1beta1WhenExpression.  # noqa: E501
        :type: str
        """

        self._operator = operator

    @property
    def values(self):
        """Gets the values of this V1beta1WhenExpression.  # noqa: E501

        DeprecatedValues for backwards compatibility with <v0.17 it represents a DeprecatedInput's relationship to the DeprecatedValues  # noqa: E501

        :return: The values of this V1beta1WhenExpression.  # noqa: E501
        :rtype: list[str]
        """
        return self._values

    @values.setter
    def values(self, values):
        """Sets the values of this V1beta1WhenExpression.

        DeprecatedValues for backwards compatibility with <v0.17 it represents a DeprecatedInput's relationship to the DeprecatedValues  # noqa: E501

        :param values: The values of this V1beta1WhenExpression.  # noqa: E501
        :type: list[str]
        """

        self._values = values

    @property
    def input(self):
        """Gets the input of this V1beta1WhenExpression.  # noqa: E501

        Input is the string for guard checking which can be a static input or an output from a parent Task  # noqa: E501

        :return: The input of this V1beta1WhenExpression.  # noqa: E501
        :rtype: str
        """
        return self._input

    @input.setter
    def input(self, input):
        """Sets the input of this V1beta1WhenExpression.

        Input is the string for guard checking which can be a static input or an output from a parent Task  # noqa: E501

        :param input: The input of this V1beta1WhenExpression.  # noqa: E501
        :type: str
        """
        if self.local_vars_configuration.client_side_validation and input is None:  # noqa: E501
            raise ValueError("Invalid value for `input`, must not be `None`")  # noqa: E501

        self._input = input

    @property
    def operator(self):
        """Gets the operator of this V1beta1WhenExpression.  # noqa: E501

        Operator that represents an Input's relationship to the values  # noqa: E501

        :return: The operator of this V1beta1WhenExpression.  # noqa: E501
        :rtype: str
        """
        return self._operator

    @operator.setter
    def operator(self, operator):
        """Sets the operator of this V1beta1WhenExpression.

        Operator that represents an Input's relationship to the values  # noqa: E501

        :param operator: The operator of this V1beta1WhenExpression.  # noqa: E501
        :type: str
        """
        if self.local_vars_configuration.client_side_validation and operator is None:  # noqa: E501
            raise ValueError("Invalid value for `operator`, must not be `None`")  # noqa: E501

        self._operator = operator

    @property
    def values(self):
        """Gets the values of this V1beta1WhenExpression.  # noqa: E501

        Values is an array of strings, which is compared against the input, for guard checking It must be non-empty  # noqa: E501

        :return: The values of this V1beta1WhenExpression.  # noqa: E501
        :rtype: list[str]
        """
        return self._values

    @values.setter
    def values(self, values):
        """Sets the values of this V1beta1WhenExpression.

        Values is an array of strings, which is compared against the input, for guard checking It must be non-empty  # noqa: E501

        :param values: The values of this V1beta1WhenExpression.  # noqa: E501
        :type: list[str]
        """
        if self.local_vars_configuration.client_side_validation and values is None:  # noqa: E501
            raise ValueError("Invalid value for `values`, must not be `None`")  # noqa: E501

        self._values = values

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
        if not isinstance(other, V1beta1WhenExpression):
            return False

        return self.to_dict() == other.to_dict()

    def __ne__(self, other):
        """Returns true if both objects are not equal"""
        if not isinstance(other, V1beta1WhenExpression):
            return True

        return self.to_dict() != other.to_dict()
