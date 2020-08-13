# V1beta1TaskResources

TaskResources allows a Pipeline to declare how its DeclaredPipelineResources should be provided to a Task as its inputs and outputs.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**inputs** | [**list[V1beta1TaskResource]**](V1beta1TaskResource.md) | Inputs holds the mapping from the PipelineResources declared in DeclaredPipelineResources to the input PipelineResources required by the Task. | [optional] 
**outputs** | [**list[V1beta1TaskResource]**](V1beta1TaskResource.md) | Outputs holds the mapping from the PipelineResources declared in DeclaredPipelineResources to the input PipelineResources required by the Task. | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


