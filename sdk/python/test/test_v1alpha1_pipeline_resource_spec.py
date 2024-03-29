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


from __future__ import absolute_import

import unittest
import datetime

import tekton_pipeline
from tekton_pipeline.models.v1alpha1_pipeline_resource_spec import V1alpha1PipelineResourceSpec  # noqa: E501
from tekton_pipeline.rest import ApiException

class TestV1alpha1PipelineResourceSpec(unittest.TestCase):
    """V1alpha1PipelineResourceSpec unit test stubs"""

    def setUp(self):
        pass

    def tearDown(self):
        pass

    def make_instance(self, include_optional):
        """Test V1alpha1PipelineResourceSpec
            include_option is a boolean, when False only required
            params are included, when True both required and
            optional params are included """
        # model = tekton_pipeline.models.v1alpha1_pipeline_resource_spec.V1alpha1PipelineResourceSpec()  # noqa: E501
        if include_optional :
            return V1alpha1PipelineResourceSpec(
                description = '0', 
                params = [
                    tekton_pipeline.models.v1alpha1/resource_param.v1alpha1.ResourceParam(
                        name = '0', 
                        value = '0', )
                    ], 
                secrets = [
                    tekton_pipeline.models.v1alpha1/secret_param.v1alpha1.SecretParam(
                        field_name = '0', 
                        secret_key = '0', 
                        secret_name = '0', )
                    ], 
                type = '0'
            )
        else :
            return V1alpha1PipelineResourceSpec(
                params = [
                    tekton_pipeline.models.v1alpha1/resource_param.v1alpha1.ResourceParam(
                        name = '0', 
                        value = '0', )
                    ],
                type = '0',
        )

    def testV1alpha1PipelineResourceSpec(self):
        """Test V1alpha1PipelineResourceSpec"""
        inst_req_only = self.make_instance(include_optional=False)
        inst_req_and_optional = self.make_instance(include_optional=True)


if __name__ == '__main__':
    unittest.main()
