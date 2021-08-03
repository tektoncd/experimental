# V1beta1WorkspaceDeclaration

WorkspaceDeclaration is a declaration of a volume that a Task requires.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**description** | **str** | Description is an optional human readable description of this volume. | [optional] 
**mount_path** | **str** | MountPath overrides the directory that the volume will be made available at. | [optional] 
**name** | **str** | Name is the name by which you can bind the volume at runtime. | [default to '']
**optional** | **bool** | Optional marks a Workspace as not being required in TaskRuns. By default this field is false and so declared workspaces are required. | [optional] 
**read_only** | **bool** | ReadOnly dictates whether a mounted volume is writable. By default this field is false and so mounted volumes are writable. | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


