# V1beta1TaskRunStatus

TaskRunStatus defines the observed state of TaskRun
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**annotations** | **dict(str, str)** | Annotations is additional Status fields for the Resource to save some additional State as well as convey more information to the user. This is roughly akin to Annotations on any k8s resource, just the reconciler conveying richer information outwards. | [optional] 
**cloud_events** | [**list[V1beta1CloudEventDelivery]**](V1beta1CloudEventDelivery.md) | CloudEvents describe the state of each cloud event requested via a CloudEventResource. | [optional] 
**completion_time** | [**V1Time**](V1Time.md) |  | [optional] 
**conditions** | [**list[KnativeCondition]**](KnativeCondition.md) | Conditions the latest available observations of a resource&#39;s current state. | [optional] 
**observed_generation** | **int** | ObservedGeneration is the &#39;Generation&#39; of the Service that was last processed by the controller. | [optional] 
**pod_name** | **str** | PodName is the name of the pod responsible for executing this task&#39;s steps. | [default to '']
**resources_result** | [**list[V1beta1PipelineResourceResult]**](V1beta1PipelineResourceResult.md) | Results from Resources built during the taskRun. currently includes the digest of build container images | [optional] 
**retries_status** | [**list[V1beta1TaskRunStatus]**](V1beta1TaskRunStatus.md) | RetriesStatus contains the history of TaskRunStatus in case of a retry in order to keep record of failures. All TaskRunStatus stored in RetriesStatus will have no date within the RetriesStatus as is redundant. | [optional] 
**sidecars** | [**list[V1beta1SidecarState]**](V1beta1SidecarState.md) | The list has one entry per sidecar in the manifest. Each entry is represents the imageid of the corresponding sidecar. | [optional] 
**start_time** | [**V1Time**](V1Time.md) |  | [optional] 
**steps** | [**list[V1beta1StepState]**](V1beta1StepState.md) | Steps describes the state of each build step container. | [optional] 
**task_results** | [**list[V1beta1TaskRunResult]**](V1beta1TaskRunResult.md) | TaskRunResults are the list of results written out by the task&#39;s containers | [optional] 
**task_spec** | [**V1beta1TaskSpec**](V1beta1TaskSpec.md) |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


