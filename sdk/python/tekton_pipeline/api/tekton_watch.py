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
from kubernetes import client
from kubernetes import watch as k8s_watch
from table_logger import TableLogger

from tekton_pipeline.constants import constants
from tekton_pipeline.utils import utils

def watch(name, plural, namespace=None, timeout_seconds=600, version=constants.TEKTON_VERSION):
    """Watch the created or patched tekton objects in the specified namespace"""

    if namespace is None:
        namespace = utils.get_default_target_namespace()

    tbl = TableLogger(
        columns='NAME,SUCCEEDED,REASON,STARTED,COMPLETED',
        colwidth={'NAME': 20, 'SUCCEEDED': 20, 'REASON': 20, 'STARTED': 20, 'COMPLETED': 20},
        border=False)

    stream = k8s_watch.Watch().stream(
        client.CustomObjectsApi().list_namespaced_custom_object,
        constants.TEKTON_GROUP,
        version,
        namespace,
        plural,
        timeout_seconds=timeout_seconds)

    for event in stream:
        tekton = event['object']
        tekton_name = tekton['metadata']['name']
        if name and name != tekton_name:
            continue
        else:
            if tekton.get('status', ''):
                status = ''
                reason = ''
                startTime = tekton['status'].get('startTime','')
                completionTime = tekton['status'].get('completionTime','')
                for condition in tekton['status'].get('conditions', {}):
                    status = condition.get('status', '')
                    reason = condition.get('reason', '')
                tbl(tekton_name, status, reason, startTime, completionTime)
            else:
                tbl(tekton_name, '', '', '', '')
                # Sleep 2 to avoid status section is not generated within a very short time.
                time.sleep(2)
                continue

            if name == tekton_name and status != 'Unknown':
                break
