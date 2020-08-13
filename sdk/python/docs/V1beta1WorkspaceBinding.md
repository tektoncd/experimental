# V1beta1WorkspaceBinding

WorkspaceBinding maps a Task's declared workspace to a Volume.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**config_map** | [**V1ConfigMapVolumeSource**](https://github.com/kubernetes-client/python/blob/master/kubernetes/docs/V1ConfigMapVolumeSource.md) |  | [optional] 
**empty_dir** | [**V1EmptyDirVolumeSource**](https://github.com/kubernetes-client/python/blob/master/kubernetes/docs/V1EmptyDirVolumeSource.md) |  | [optional] 
**name** | **str** | Name is the name of the workspace populated by the volume. | 
**persistent_volume_claim** | [**V1PersistentVolumeClaimVolumeSource**](https://github.com/kubernetes-client/python/blob/master/kubernetes/docs/V1PersistentVolumeClaimVolumeSource.md) |  | [optional] 
**secret** | [**V1SecretVolumeSource**](https://github.com/kubernetes-client/python/blob/master/kubernetes/docs/V1SecretVolumeSource.md) |  | [optional] 
**sub_path** | **str** | SubPath is optionally a directory on the volume which should be used for this binding (i.e. the volume will be mounted at this sub directory). | [optional] 
**volume_claim_template** | [**V1PersistentVolumeClaim**](https://github.com/kubernetes-client/python/blob/master/kubernetes/docs/V1PersistentVolumeClaim.md) |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


