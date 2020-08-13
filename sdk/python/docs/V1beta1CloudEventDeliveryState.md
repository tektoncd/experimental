# V1beta1CloudEventDeliveryState

CloudEventDeliveryState reports the state of a cloud event to be sent.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**condition** | **str** | Current status | [optional] 
**message** | **str** | Error is the text of error (if any) | 
**retry_count** | **int** | RetryCount is the number of attempts of sending the cloud event | 
**sent_at** | [**V1Time**](V1Time.md) |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


