# V1beta1EmbeddedTask

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**api_version** | **str** |  | [optional] 
**description** | **str** | Description is a user-facing description of the task that may be used to populate a UI. | [optional] 
**kind** | **str** |  | [optional] 
**metadata** | [**V1beta1PipelineTaskMetadata**](V1beta1PipelineTaskMetadata.md) |  | [optional] 
**params** | [**list[V1beta1ParamSpec]**](V1beta1ParamSpec.md) | Params is a list of input parameters required to run the task. Params must be supplied as inputs in TaskRuns unless they declare a default value. | [optional] 
**resources** | [**V1beta1TaskResources**](V1beta1TaskResources.md) |  | [optional] 
**results** | [**list[V1beta1TaskResult]**](V1beta1TaskResult.md) | Results are values that this Task can output | [optional] 
**sidecars** | [**list[V1beta1Sidecar]**](V1beta1Sidecar.md) | Sidecars are run alongside the Task&#39;s step containers. They begin before the steps start and end after the steps complete. | [optional] 
**spec** | [**K8sIoApimachineryPkgRuntimeRawExtension**](K8sIoApimachineryPkgRuntimeRawExtension.md) |  | [optional] 
**step_template** | [**V1Container**](https://github.com/kubernetes-client/python/blob/master/kubernetes/docs/V1Container.md) |  | [optional] 
**steps** | [**list[V1beta1Step]**](V1beta1Step.md) | Steps are the steps of the build; each step is run sequentially with the source mounted into /workspace. | [optional] 
**volumes** | [**list[V1Volume]**](https://github.com/kubernetes-client/python/blob/master/kubernetes/docs/V1Volume.md) | Volumes is a collection of volumes that are available to mount into the steps of the build. | [optional] 
**workspaces** | [**list[V1beta1WorkspaceDeclaration]**](V1beta1WorkspaceDeclaration.md) | Workspaces are the volumes that this Task requires. | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


