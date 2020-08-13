# V1beta1PipelineRunStatus

PipelineRunStatus defines the observed state of PipelineRun
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**annotations** | **dict(str, str)** | Annotations is additional Status fields for the Resource to save some additional State as well as convey more information to the user. This is roughly akin to Annotations on any k8s resource, just the reconciler conveying richer information outwards. | [optional] 
**completion_time** | [**V1Time**](V1Time.md) |  | [optional] 
**conditions** | [**list[KnativeCondition]**](KnativeCondition.md) | Conditions the latest available observations of a resource&#39;s current state. | [optional] 
**observed_generation** | **int** | ObservedGeneration is the &#39;Generation&#39; of the Service that was last processed by the controller. | [optional] 
**pipeline_results** | [**list[V1beta1PipelineRunResult]**](V1beta1PipelineRunResult.md) | PipelineResults are the list of results written out by the pipeline task&#39;s containers | [optional] 
**pipeline_spec** | [**V1beta1PipelineSpec**](V1beta1PipelineSpec.md) |  | [optional] 
**skipped_tasks** | [**list[V1beta1SkippedTask]**](V1beta1SkippedTask.md) | list of tasks that were skipped due to when expressions evaluating to false | [optional] 
**start_time** | [**V1Time**](V1Time.md) |  | [optional] 
**task_runs** | [**dict(str, V1beta1PipelineRunTaskRunStatus)**](V1beta1PipelineRunTaskRunStatus.md) | map of PipelineRunTaskRunStatus with the taskRun name as the key | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


