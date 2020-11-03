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
TektonClient | [delete](#delete) | Delete the specified tekton object |


## create
> create(tekton, plural=None, namespace=None)

Create the provided Tekton object in the specified namespace

### Example

```python
from kubernetes import client

from tekton_pipeline import TektonClient
from tekton_pipeline import V1beta1TaskRun
from tekton_pipeline import V1beta1TaskRunSpec
from tekton_pipeline import V1beta1TaskSpec
from tekton_pipeline import V1beta1Step

tekton_client = TektonClient()

taskrun = V1beta1TaskRun(
    api_version='tekton.dev/v1beta1',
    kind='TaskRun',
    metadata=client.V1ObjectMeta(name='sdk-sample-taskrun'),
    spec=V1beta1TaskRunSpec(
        task_spec=V1beta1TaskSpec(
            steps=[V1beta1Step(name='default',
                            image='ubuntu',
                            script='sleep 30;echo "This is a sdk demo.')]
        )))

tekton_client.create(taskrun, namespace='default')
```


### Parameters
Name | Type |  Description | Notes
------------ | ------------- | ------------- | -------------
tekton  | tekton object | tekton object defination| Required |
namespace | str | Namespace for tekton object deploying to. If the `namespace` is not defined, will align with tekton object definition, or use current or default namespace if namespace is not specified in tekton object definition.  | Optional |
plural | tekton object plural | tekton object plural | Optional |

### Return type
object

## get
> get(self, name, plural, namespace=None)

Get the created tekton object in the specified namespace

### Example

```python
from tekton_pipeline import TektonClient

tekton_client = TektonClient()

tekton_client.get(name='sdk-sample-taskrun', plural='taskruns', namespace='default')

```


### Parameters
Name | Type |  Description | Notes
------------ | ------------- | ------------- | -------------
name  | str | tekton object name. | Required. |
plural | tekton object plural | tekton object plural | Required |
namespace | str | The tekton object's namespace. Defaults to current or default namespace.| Optional |

### Return type
object


## delete
> delete(self, name, plural, namespace=None)

Delete the created tekton object in the specified namespace

### Example

```python

from tekton_pipeline import TektonClient

tekton_client = TektonClient()

tekton_client.delete(name='sdk-sample-taskrun', plural='taskruns', namespace='default')

```

### Parameters
Name | Type |  Description | Notes
------------ | ------------- | ------------- | -------------
name  | str | tekton object name| |
plural | tekton object plural | tekton object plural | Required |
namespace | str | The tekton object's namespace. Defaults to current or default namespace. | Optional|

### Return type
object
