# V1beta1PipelineTaskInputResource

PipelineTaskInputResource maps the name of a declared PipelineResource input dependency in a Task to the resource in the Pipeline's DeclaredPipelineResources that should be used. This input may come from a previous task.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**_from** | **list[str]** | From is the list of PipelineTask names that the resource has to come from. (Implies an ordering in the execution graph.) | [optional] 
**name** | **str** | Name is the name of the PipelineResource as declared by the Task. | 
**resource** | **str** | Resource is the name of the DeclaredPipelineResource to use. | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


