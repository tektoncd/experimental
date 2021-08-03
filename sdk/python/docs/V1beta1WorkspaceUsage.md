# V1beta1WorkspaceUsage

WorkspaceUsage is used by a Step or Sidecar to declare that it wants isolated access to a Workspace defined in a Task.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**mount_path** | **str** | MountPath is the path that the workspace should be mounted to inside the Step or Sidecar, overriding any MountPath specified in the Task&#39;s WorkspaceDeclaration. | [default to '']
**name** | **str** | Name is the name of the workspace this Step or Sidecar wants access to. | [default to '']

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


