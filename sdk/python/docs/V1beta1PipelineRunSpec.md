# V1beta1PipelineRunSpec

PipelineRunSpec defines the desired state of PipelineRun
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**params** | [**list[V1beta1Param]**](V1beta1Param.md) | Params is a list of parameter names and values. | [optional] 
**pipeline_ref** | [**V1beta1PipelineRef**](V1beta1PipelineRef.md) |  | [optional] 
**pipeline_spec** | [**V1beta1PipelineSpec**](V1beta1PipelineSpec.md) |  | [optional] 
**pod_template** | [**PodTemplate**](PodTemplate.md) |  | [optional] 
**resources** | [**list[V1beta1PipelineResourceBinding]**](V1beta1PipelineResourceBinding.md) | Resources is a list of bindings specifying which actual instances of PipelineResources to use for the resources the Pipeline has declared it needs. | [optional] 
**service_account_name** | **str** |  | [optional] 
**service_account_names** | [**list[V1beta1PipelineRunSpecServiceAccountName]**](V1beta1PipelineRunSpecServiceAccountName.md) | Deprecated: use taskRunSpecs.ServiceAccountName instead | [optional] 
**status** | **str** | Used for cancelling a pipelinerun (and maybe more later on) | [optional] 
**task_run_specs** | [**list[V1beta1PipelineTaskRunSpec]**](V1beta1PipelineTaskRunSpec.md) | TaskRunSpecs holds a set of runtime specs | [optional] 
**timeout** | [**V1Duration**](V1Duration.md) |  | [optional] 
**timeouts** | [**V1beta1TimeoutFields**](V1beta1TimeoutFields.md) |  | [optional] 
**workspaces** | [**list[V1beta1WorkspaceBinding]**](V1beta1WorkspaceBinding.md) | Workspaces holds a set of workspace bindings that must match names with those declared in the pipeline. | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


