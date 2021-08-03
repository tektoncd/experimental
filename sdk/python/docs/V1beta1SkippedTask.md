# V1beta1SkippedTask

SkippedTask is used to describe the Tasks that were skipped due to their When Expressions evaluating to False. This is a struct because we are looking into including more details about the When Expressions that caused this Task to be skipped.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**name** | **str** | Name is the Pipeline Task name | [default to '']
**when_expressions** | [**list[V1beta1WhenExpression]**](V1beta1WhenExpression.md) | WhenExpressions is the list of checks guarding the execution of the PipelineTask | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


