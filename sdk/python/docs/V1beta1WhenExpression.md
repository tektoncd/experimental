# V1beta1WhenExpression

WhenExpression allows a PipelineTask to declare expressions to be evaluated before the Task is run to determine whether the Task should be executed or skipped
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**input** | **str** | DeprecatedInput for backwards compatibility with &lt;v0.17 it is the string for guard checking which can be a static input or an output from a parent Task | [optional] 
**operator** | **str** | DeprecatedOperator for backwards compatibility with &lt;v0.17 it represents a DeprecatedInput&#39;s relationship to the DeprecatedValues | [optional] 
**values** | **list[str]** | DeprecatedValues for backwards compatibility with &lt;v0.17 it represents a DeprecatedInput&#39;s relationship to the DeprecatedValues | [optional] 
**input** | **str** | Input is the string for guard checking which can be a static input or an output from a parent Task | 
**operator** | **str** | Operator that represents an Input&#39;s relationship to the values | 
**values** | **list[str]** | Values is an array of strings, which is compared against the input, for guard checking It must be non-empty | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


