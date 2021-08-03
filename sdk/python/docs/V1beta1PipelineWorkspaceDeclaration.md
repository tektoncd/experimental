# V1beta1PipelineWorkspaceDeclaration

WorkspacePipelineDeclaration creates a named slot in a Pipeline that a PipelineRun is expected to populate with a workspace binding. Deprecated: use PipelineWorkspaceDeclaration type instead
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**description** | **str** | Description is a human readable string describing how the workspace will be used in the Pipeline. It can be useful to include a bit of detail about which tasks are intended to have access to the data on the workspace. | [optional] 
**name** | **str** | Name is the name of a workspace to be provided by a PipelineRun. | [default to '']
**optional** | **bool** | Optional marks a Workspace as not being required in PipelineRuns. By default this field is false and so declared workspaces are required. | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


