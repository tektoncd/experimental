# V1beta1PipelineSpec

PipelineSpec defines the desired state of Pipeline.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**description** | **str** | Description is a user-facing description of the pipeline that may be used to populate a UI. | [optional] 
**_finally** | [**list[V1beta1PipelineTask]**](V1beta1PipelineTask.md) | Finally declares the list of Tasks that execute just before leaving the Pipeline i.e. either after all Tasks are finished executing successfully or after a failure which would result in ending the Pipeline | [optional] 
**params** | [**list[V1beta1ParamSpec]**](V1beta1ParamSpec.md) | Params declares a list of input parameters that must be supplied when this Pipeline is run. | [optional] 
**resources** | [**list[V1beta1PipelineDeclaredResource]**](V1beta1PipelineDeclaredResource.md) | Resources declares the names and types of the resources given to the Pipeline&#39;s tasks as inputs and outputs. | [optional] 
**results** | [**list[V1beta1PipelineResult]**](V1beta1PipelineResult.md) | Results are values that this pipeline can output once run | [optional] 
**tasks** | [**list[V1beta1PipelineTask]**](V1beta1PipelineTask.md) | Tasks declares the graph of Tasks that execute when this Pipeline is run. | [optional] 
**workspaces** | [**list[V1beta1PipelineWorkspaceDeclaration]**](V1beta1PipelineWorkspaceDeclaration.md) | Workspaces declares a set of named workspaces that are expected to be provided by a PipelineRun. | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


