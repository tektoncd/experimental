# V1beta1TaskResourceBinding

TaskResourceBinding points to the PipelineResource that will be used for the Task input or output called Name.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**name** | **str** | Name is the name of the PipelineResource in the Pipeline&#39;s declaration | [optional] 
**paths** | **list[str]** | Paths will probably be removed in #1284, and then PipelineResourceBinding can be used instead. The optional Path field corresponds to a path on disk at which the Resource can be found (used when providing the resource via mounted volume, overriding the default logic to fetch the Resource). | [optional] 
**resource_ref** | [**V1beta1PipelineResourceRef**](V1beta1PipelineResourceRef.md) |  | [optional] 
**resource_spec** | [**V1alpha1PipelineResourceSpec**](V1alpha1PipelineResourceSpec.md) |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


