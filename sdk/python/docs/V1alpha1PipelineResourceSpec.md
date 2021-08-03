# V1alpha1PipelineResourceSpec

PipelineResourceSpec defines  an individual resources used in the pipeline.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**description** | **str** | Description is a user-facing description of the resource that may be used to populate a UI. | [optional] 
**params** | [**list[V1alpha1ResourceParam]**](V1alpha1ResourceParam.md) |  | 
**secrets** | [**list[V1alpha1SecretParam]**](V1alpha1SecretParam.md) | Secrets to fetch to populate some of resource fields | [optional] 
**type** | **str** |  | [default to '']

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


