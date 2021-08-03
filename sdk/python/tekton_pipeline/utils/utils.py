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

import os


def is_running_in_k8s():
    return os.path.isdir('/var/run/secrets/kubernetes.io/')


def get_current_k8s_namespace():
    with open('/var/run/secrets/kubernetes.io/serviceaccount/namespace', 'r') as f:
        return f.readline()


def get_default_target_namespace():
    if not is_running_in_k8s():
        return 'default'
    return get_current_k8s_namespace()


def get_tekton_namespace(tekton):
    tekton_namespace = tekton.metadata.namespace
    namespace = tekton_namespace or get_default_target_namespace()
    return namespace


def get_tekton_plural(tekton):
    tekton_plural = str(tekton.kind).lower() + "s"
    return tekton_plural

def check_entity(entity):
    valid_entities = ['task', 'taskrun', 'pipeline', 'pipelinerun']
    if entity not in valid_entities:
        raise RuntimeError("The entity %s is not support, currently supported entities: %s" %
                           (entity, valid_entities))
