# V1beta1StepState

StepState reports the results of running a step in a Task.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**container** | **str** |  | [optional] 
**image_id** | **str** |  | [optional] 
**name** | **str** |  | [optional] 
**running** | [**V1ContainerStateRunning**](https://github.com/kubernetes-client/python/blob/master/kubernetes/docs/V1ContainerStateRunning.md) |  | [optional] 
**terminated** | [**V1ContainerStateTerminated**](https://github.com/kubernetes-client/python/blob/master/kubernetes/docs/V1ContainerStateTerminated.md) |  | [optional] 
**waiting** | [**V1ContainerStateWaiting**](https://github.com/kubernetes-client/python/blob/master/kubernetes/docs/V1ContainerStateWaiting.md) |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


