# V1beta1PipelineRunRunStatus

PipelineRunRunStatus contains the name of the PipelineTask for this Run and the Run's Status
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**pipeline_task_name** | **str** | PipelineTaskName is the name of the PipelineTask. | [optional] 
**status** | [**GithubComTektoncdPipelinePkgApisRunV1alpha1RunStatus**](GithubComTektoncdPipelinePkgApisRunV1alpha1RunStatus.md) |  | [optional] 
**when_expressions** | [**list[V1beta1WhenExpression]**](V1beta1WhenExpression.md) | WhenExpressions is the list of checks guarding the execution of the PipelineTask | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


