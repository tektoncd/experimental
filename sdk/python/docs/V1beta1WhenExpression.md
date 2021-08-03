# V1beta1WhenExpression

WhenExpression allows a PipelineTask to declare expressions to be evaluated before the Task is run to determine whether the Task should be executed or skipped
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**input** | **str** | Input is the string for guard checking which can be a static input or an output from a parent Task | [default to '']
**operator** | **str** | Operator that represents an Input&#39;s relationship to the values | [default to '']
**values** | **list[str]** | Values is an array of strings, which is compared against the input, for guard checking It must be non-empty | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


