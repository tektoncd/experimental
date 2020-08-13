# V1beta1ConditionCheckStatus

ConditionCheckStatus defines the observed state of ConditionCheck
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**annotations** | **dict(str, str)** | Annotations is additional Status fields for the Resource to save some additional State as well as convey more information to the user. This is roughly akin to Annotations on any k8s resource, just the reconciler conveying richer information outwards. | [optional] 
**check** | [**V1ContainerState**](https://github.com/kubernetes-client/python/blob/master/kubernetes/docs/V1ContainerState.md) |  | [optional] 
**completion_time** | [**V1Time**](V1Time.md) |  | [optional] 
**conditions** | [**list[KnativeCondition]**](KnativeCondition.md) | Conditions the latest available observations of a resource&#39;s current state. | [optional] 
**observed_generation** | **int** | ObservedGeneration is the &#39;Generation&#39; of the Service that was last processed by the controller. | [optional] 
**pod_name** | **str** | PodName is the name of the pod responsible for executing this condition check. | 
**start_time** | [**V1Time**](V1Time.md) |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


