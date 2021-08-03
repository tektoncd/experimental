# V1beta1PipelineRunStatusFields

PipelineRunStatusFields holds the fields of PipelineRunStatus' status. This is defined separately and inlined so that other types can readily consume these fields via duck typing.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**completion_time** | [**V1Time**](V1Time.md) |  | [optional] 
**pipeline_results** | [**list[V1beta1PipelineRunResult]**](V1beta1PipelineRunResult.md) | PipelineResults are the list of results written out by the pipeline task&#39;s containers | [optional] 
**pipeline_spec** | [**V1beta1PipelineSpec**](V1beta1PipelineSpec.md) |  | [optional] 
**runs** | [**dict(str, V1beta1PipelineRunRunStatus)**](V1beta1PipelineRunRunStatus.md) | map of PipelineRunRunStatus with the run name as the key | [optional] 
**skipped_tasks** | [**list[V1beta1SkippedTask]**](V1beta1SkippedTask.md) | list of tasks that were skipped due to when expressions evaluating to false | [optional] 
**start_time** | [**V1Time**](V1Time.md) |  | [optional] 
**task_runs** | [**dict(str, V1beta1PipelineRunTaskRunStatus)**](V1beta1PipelineRunTaskRunStatus.md) | map of PipelineRunTaskRunStatus with the taskRun name as the key | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


