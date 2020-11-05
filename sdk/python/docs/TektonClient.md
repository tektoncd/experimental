# TektonClient

> TektonClient(config_file=None, context=None, client_configuration=None, persist_config=True)

User can loads authentication and cluster information from kube-config file and stores them in kubernetes.client.configuration. Parameters are as following:

parameter |  Description
------------ | -------------
config_file | Name of the kube-config file. Defaults to `~/.kube/config`. Note that for the case that the SDK is running in cluster and you want to operate Tekton object in another remote cluster, user must set `config_file` to load kube-config file explicitly, e.g. `TektonClient(config_file="~/.kube/config")`. |
context |Set the active context. If is set to None, current_context from config file will be used.|
client_configuration | The kubernetes.client.Configuration to set configs to.|
persist_config | If True, config file will be updated when changed (e.g GCP token refresh).|


The APIs for TektonClient are as following:

Class | Method |  Description
------------ | ------------- | -------------
TektonClient | [create](#create) | Create tekton object, such as task, taskrun, pipeline etc...|
TektonClient | [get](#get)    | Get or watch the specified tekton object in the namespace |
TektonClient | [patch](#patch) | Patch the specified tekton object in the namespace |
TektonClient | [delete](#delete) | Delete the specified tekton object |


## create
> create(entity, body, namespace=None)

Create the provided Tekton object in the specified namespace

### Example

```python
from kubernetes import client as k8s_client
from tekton_pipeline import V1beta1Task
from tekton_pipeline import V1beta1TaskSpec
from tekton_pipeline import V1beta1Step

# Define the task
task = V1beta1Task(api_version='tekton.dev/v1beta1',
                   kind='TaskRun',
                   metadata=k8s_client.V1ObjectMeta(name='sdk-sample-task'),
                   spec=V1beta1TaskSpec(
                       steps=[V1beta1Step(name='default',
                              image='ubuntu',
                              script='sleep 30;echo "This is a sdk demo."')]
                   ))

# Submit the task to cluster
tekton_client.create(entity='task', body=task, namespace='default')
```


### Parameters
Name | Type |  Description | Notes
------------ | ------------- | ------------- | -------------
entity  | str | Tekton entity, valid value: ['task', 'taskrun', 'pipeline', 'pipelinerun']| Required |
namespace | str | Namespace for tekton object deploying to. If the `namespace` is not defined, will align with tekton object definition, or use current or default namespace if namespace is not specified in tekton object definition.  | Optional |

### Return type
object

## get
> get(entity, name, namespace=None, watch=False, timeout_seconds=600)

Get the created tekton object in the specified namespace

### Example

```python
from tekton_pipeline import TektonClient

tekton_client = TektonClient()

tekton_client.get(entity='task', name='sdk-sample-task', namespace='default')

# Or watch the taskrun or pipeline run as below
tekton_client.get(entity='taskrun', name='sdk-sample-taskrun', namespace='default', watch=True)

```


### Parameters
Name | Type |  Description | Notes
------------ | ------------- | ------------- | -------------
entity  | str | Tekton entity, valid value: ['task', 'taskrun', 'pipeline', 'pipelinerun']| Required |
name  | str | tekton object name| |
namespace | str | The tekton object's namespace. Defaults to current or default namespace.| Optional |
watch | bool | Watch the created Tekton object if `True`, otherwise will return the created Tekton object. Stop watching if reaches the optional specified `timeout_seconds` or once the status `Succeeded` or `Failed`. | Optional |
timeout_seconds | int | Timeout seconds for watching. Defaults to 600. | Optional |

### Return type
object


## patch
> patch(entity, name, body, namespace=None)

Patch the provided Tekton object in the specified namespace

### Example

```python
# Update the task defination
task = V1beta1Task(api_version='tekton.dev/v1beta1',
                   kind='TaskRun',
                   metadata=k8s_client.V1ObjectMeta(name='sdk-sample-task'),
                   spec=V1beta1TaskSpec(
                       steps=[V1beta1Step(name='default',
                              image='ubuntu',
                              script='sleep 30;echo "This is a sdk patch demo."')]
                   ))

# Patch the task
tekton_client.patch(entity='task', name='sdk-sample-task', body=task, namespace='default')
```


### Parameters
Name | Type |  Description | Notes
------------ | ------------- | ------------- | -------------
entity  | str | Tekton entity, valid value: ['task', 'taskrun', 'pipeline', 'pipelinerun']| Required |
name  | str | tekton object name| |
namespace | str | Namespace for tekton object deploying to. If the `namespace` is not defined, will align with tekton object definition, or use current or default namespace if namespace is not specified in tekton object definition.  | Optional |


## delete
> delete(entity, name, namespace=None)

Delete the created tekton object in the specified namespace

### Example

```python

from tekton_pipeline import TektonClient

tekton_client = TektonClient()

tekton_client.delete(entity='task', name='sdk-sample-task', namespace='default')

```

### Parameters
Name | Type |  Description | Notes
------------ | ------------- | ------------- | -------------
entity  | str | Tekton entity, valid value: ['task', 'taskrun', 'pipeline', 'pipelinerun']| Required |
name  | str | tekton object name| |
namespace | str | The tekton object's namespace. Defaults to current or default namespace. | Optional|

### Return type
object
