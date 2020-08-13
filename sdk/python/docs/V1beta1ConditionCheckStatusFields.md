# V1beta1ConditionCheckStatusFields

ConditionCheckStatusFields holds the fields of ConfigurationCheck's status. This is defined separately and inlined so that other types can readily consume these fields via duck typing.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**check** | [**V1ContainerState**](https://github.com/kubernetes-client/python/blob/master/kubernetes/docs/V1ContainerState.md) |  | [optional] 
**completion_time** | [**V1Time**](V1Time.md) |  | [optional] 
**pod_name** | **str** | PodName is the name of the pod responsible for executing this condition check. | 
**start_time** | [**V1Time**](V1Time.md) |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


