# V1beta1PipelineRunTaskRunStatus

PipelineRunTaskRunStatus contains the name of the PipelineTask for this TaskRun and the TaskRun's Status
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**condition_checks** | [**dict(str, V1beta1PipelineRunConditionCheckStatus)**](V1beta1PipelineRunConditionCheckStatus.md) | ConditionChecks maps the name of a condition check to its Status | [optional] 
**pipeline_task_name** | **str** | PipelineTaskName is the name of the PipelineTask. | [optional] 
**status** | [**V1beta1TaskRunStatus**](V1beta1TaskRunStatus.md) |  | [optional] 
**when_expressions** | [**list[V1beta1WhenExpression]**](V1beta1WhenExpression.md) | WhenExpressions is the list of checks guarding the execution of the PipelineTask | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


