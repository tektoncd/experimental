# V1beta1TaskRunSpec

TaskRunSpec defines the desired state of TaskRun
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**debug** | [**V1beta1TaskRunDebug**](V1beta1TaskRunDebug.md) |  | [optional] 
**params** | [**list[V1beta1Param]**](V1beta1Param.md) |  | [optional] 
**pod_template** | [**PodTemplate**](PodTemplate.md) |  | [optional] 
**resources** | [**V1beta1TaskRunResources**](V1beta1TaskRunResources.md) |  | [optional] 
**service_account_name** | **str** |  | [optional] [default to '']
**status** | **str** | Used for cancelling a taskrun (and maybe more later on) | [optional] 
**task_ref** | [**V1beta1TaskRef**](V1beta1TaskRef.md) |  | [optional] 
**task_spec** | [**V1beta1TaskSpec**](V1beta1TaskSpec.md) |  | [optional] 
**timeout** | [**V1Duration**](V1Duration.md) |  | [optional] 
**workspaces** | [**list[V1beta1WorkspaceBinding]**](V1beta1WorkspaceBinding.md) | Workspaces is a list of WorkspaceBindings from volumes to workspaces. | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


