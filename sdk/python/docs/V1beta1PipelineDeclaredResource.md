# V1beta1PipelineDeclaredResource

PipelineDeclaredResource is used by a Pipeline to declare the types of the PipelineResources that it will required to run and names which can be used to refer to these PipelineResources in PipelineTaskResourceBindings.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**name** | **str** | Name is the name that will be used by the Pipeline to refer to this resource. It does not directly correspond to the name of any PipelineResources Task inputs or outputs, and it does not correspond to the actual names of the PipelineResources that will be bound in the PipelineRun. | [default to '']
**optional** | **bool** | Optional declares the resource as optional. optional: true - the resource is considered optional optional: false - the resource is considered required (default/equivalent of not specifying it) | [optional] 
**type** | **str** | Type is the type of the PipelineResource. | [default to '']

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


