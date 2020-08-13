# V1beta1PipelineTaskResources

PipelineTaskResources allows a Pipeline to declare how its DeclaredPipelineResources should be provided to a Task as its inputs and outputs.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**inputs** | [**list[V1beta1PipelineTaskInputResource]**](V1beta1PipelineTaskInputResource.md) | Inputs holds the mapping from the PipelineResources declared in DeclaredPipelineResources to the input PipelineResources required by the Task. | [optional] 
**outputs** | [**list[V1beta1PipelineTaskOutputResource]**](V1beta1PipelineTaskOutputResource.md) | Outputs holds the mapping from the PipelineResources declared in DeclaredPipelineResources to the input PipelineResources required by the Task. | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


