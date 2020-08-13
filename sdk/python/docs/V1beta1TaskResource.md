# V1beta1TaskResource

TaskResource defines an input or output Resource declared as a requirement by a Task. The Name field will be used to refer to these Resources within the Task definition, and when provided as an Input, the Name will be the path to the volume mounted containing this Resource as an input (e.g. an input Resource named `workspace` will be mounted at `/workspace`).
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**description** | **str** | Description is a user-facing description of the declared resource that may be used to populate a UI. | [optional] 
**name** | **str** | Name declares the name by which a resource is referenced in the definition. Resources may be referenced by name in the definition of a Task&#39;s steps. | 
**optional** | **bool** | Optional declares the resource as optional. By default optional is set to false which makes a resource required. optional: true - the resource is considered optional optional: false - the resource is considered required (equivalent of not specifying it) | [optional] 
**target_path** | **str** | TargetPath is the path in workspace directory where the resource will be copied. | [optional] 
**type** | **str** | Type is the type of this resource; | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


