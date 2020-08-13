# V1beta1PipelineTaskCondition

PipelineTaskCondition allows a PipelineTask to declare a Condition to be evaluated before the Task is run.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**condition_ref** | **str** | ConditionRef is the name of the Condition to use for the conditionCheck | 
**params** | [**list[V1beta1Param]**](V1beta1Param.md) | Params declare parameters passed to this Condition | [optional] 
**resources** | [**list[V1beta1PipelineTaskInputResource]**](V1beta1PipelineTaskInputResource.md) | Resources declare the resources provided to this Condition as input | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


