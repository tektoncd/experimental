# V1beta1WorkspacePipelineTaskBinding

WorkspacePipelineTaskBinding describes how a workspace passed into the pipeline should be mapped to a task's declared workspace.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**name** | **str** | Name is the name of the workspace as declared by the task | 
**sub_path** | **str** | SubPath is optionally a directory on the volume which should be used for this binding (i.e. the volume will be mounted at this sub directory). | [optional] 
**workspace** | **str** | Workspace is the name of the workspace declared by the pipeline | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


