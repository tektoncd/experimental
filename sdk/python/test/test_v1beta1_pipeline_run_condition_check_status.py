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


from __future__ import absolute_import

import unittest
import datetime

import tekton_pipeline
from tekton_pipeline.models.v1beta1_pipeline_run_condition_check_status import V1beta1PipelineRunConditionCheckStatus  # noqa: E501
from tekton_pipeline.rest import ApiException

class TestV1beta1PipelineRunConditionCheckStatus(unittest.TestCase):
    """V1beta1PipelineRunConditionCheckStatus unit test stubs"""

    def setUp(self):
        pass

    def tearDown(self):
        pass

    def make_instance(self, include_optional):
        """Test V1beta1PipelineRunConditionCheckStatus
            include_option is a boolean, when False only required
            params are included, when True both required and
            optional params are included """
        # model = tekton_pipeline.models.v1beta1_pipeline_run_condition_check_status.V1beta1PipelineRunConditionCheckStatus()  # noqa: E501
        if include_optional :
            return V1beta1PipelineRunConditionCheckStatus(
                condition_name = '0', 
                status = tekton_pipeline.models.v1beta1/condition_check_status.v1beta1.ConditionCheckStatus(
                    annotations = {
                        'key' : '0'
                        }, 
                    check = None, 
                    completion_time = None, 
                    conditions = [
                        None
                        ], 
                    observed_generation = 56, 
                    pod_name = '0', 
                    start_time = None, )
            )
        else :
            return V1beta1PipelineRunConditionCheckStatus(
        )

    def testV1beta1PipelineRunConditionCheckStatus(self):
        """Test V1beta1PipelineRunConditionCheckStatus"""
        inst_req_only = self.make_instance(include_optional=False)
        inst_req_and_optional = self.make_instance(include_optional=True)


if __name__ == '__main__':
    unittest.main()
