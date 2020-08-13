# V1beta1PipelineTask

PipelineTask defines a task in a Pipeline, passing inputs from both Params and from the output of previous tasks.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**conditions** | [**list[V1beta1PipelineTaskCondition]**](V1beta1PipelineTaskCondition.md) | Conditions is a list of conditions that need to be true for the task to run Conditions are deprecated, use WhenExpressions instead | [optional] 
**name** | **str** | Name is the name of this task within the context of a Pipeline. Name is used as a coordinate with the &#x60;from&#x60; and &#x60;runAfter&#x60; fields to establish the execution order of tasks relative to one another. | [optional] 
**params** | [**list[V1beta1Param]**](V1beta1Param.md) | Parameters declares parameters passed to this task. | [optional] 
**resources** | [**V1beta1PipelineTaskResources**](V1beta1PipelineTaskResources.md) |  | [optional] 
**retries** | **int** | Retries represents how many times this task should be retried in case of task failure: ConditionSucceeded set to False | [optional] 
**run_after** | **list[str]** | RunAfter is the list of PipelineTask names that should be executed before this Task executes. (Used to force a specific ordering in graph execution.) | [optional] 
**task_ref** | [**V1beta1TaskRef**](V1beta1TaskRef.md) |  | [optional] 
**task_spec** | [**V1beta1EmbeddedTask**](V1beta1EmbeddedTask.md) |  | [optional] 
**timeout** | [**V1Duration**](V1Duration.md) |  | [optional] 
**when** | [**list[V1beta1WhenExpression]**](V1beta1WhenExpression.md) | WhenExpressions is a list of when expressions that need to be true for the task to run | [optional] 
**workspaces** | [**list[V1beta1WorkspacePipelineTaskBinding]**](V1beta1WorkspacePipelineTaskBinding.md) | Workspaces maps workspaces from the pipeline spec to the workspaces declared in the Task. | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


