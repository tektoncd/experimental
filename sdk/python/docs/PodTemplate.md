# PodTemplate

PodTemplate holds pod specific configuration
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**affinity** | [**V1Affinity**](https://github.com/kubernetes-client/python/blob/master/kubernetes/docs/V1Affinity.md) |  | [optional] 
**automount_service_account_token** | **bool** | AutomountServiceAccountToken indicates whether pods running as this service account should have an API token automatically mounted. | [optional] 
**dns_config** | [**V1PodDNSConfig**](https://github.com/kubernetes-client/python/blob/master/kubernetes/docs/V1PodDNSConfig.md) |  | [optional] 
**dns_policy** | **str** | Set DNS policy for the pod. Defaults to \&quot;ClusterFirst\&quot;. Valid values are &#39;ClusterFirst&#39;, &#39;Default&#39; or &#39;None&#39;. DNS parameters given in DNSConfig will be merged with the policy selected with DNSPolicy. | [optional] 
**enable_service_links** | **bool** | EnableServiceLinks indicates whether information about services should be injected into pod&#39;s environment variables, matching the syntax of Docker links. Optional: Defaults to true. | [optional] 
**host_aliases** | [**list[V1HostAlias]**](V1HostAlias.md) | HostAliases is an optional list of hosts and IPs that will be injected into the pod&#39;s hosts file if specified. This is only valid for non-hostNetwork pods. | [optional] 
**host_network** | **bool** | HostNetwork specifies whether the pod may use the node network namespace | [optional] 
**image_pull_secrets** | [**list[V1LocalObjectReference]**](https://github.com/kubernetes-client/python/blob/master/kubernetes/docs/V1LocalObjectReference.md) | ImagePullSecrets gives the name of the secret used by the pod to pull the image if specified | [optional] 
**node_selector** | **dict(str, str)** | NodeSelector is a selector which must be true for the pod to fit on a node. Selector which must match a node&#39;s labels for the pod to be scheduled on that node. More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/ | [optional] 
**priority_class_name** | **str** | If specified, indicates the pod&#39;s priority. \&quot;system-node-critical\&quot; and \&quot;system-cluster-critical\&quot; are two special keywords which indicate the highest priorities with the former being the highest priority. Any other name must be defined by creating a PriorityClass object with that name. If not specified, the pod priority will be default or zero if there is no default. | [optional] 
**runtime_class_name** | **str** | RuntimeClassName refers to a RuntimeClass object in the node.k8s.io group, which should be used to run this pod. If no RuntimeClass resource matches the named class, the pod will not be run. If unset or empty, the \&quot;legacy\&quot; RuntimeClass will be used, which is an implicit class with an empty definition that uses the default runtime handler. More info: https://git.k8s.io/enhancements/keps/sig-node/runtime-class.md This is a beta feature as of Kubernetes v1.14. | [optional] 
**scheduler_name** | **str** | SchedulerName specifies the scheduler to be used to dispatch the Pod | [optional] 
**security_context** | [**V1PodSecurityContext**](https://github.com/kubernetes-client/python/blob/master/kubernetes/docs/V1PodSecurityContext.md) |  | [optional] 
**tolerations** | [**list[V1Toleration]**](https://github.com/kubernetes-client/python/blob/master/kubernetes/docs/V1Toleration.md) | If specified, the pod&#39;s tolerations. | [optional] 
**volumes** | [**list[V1Volume]**](https://github.com/kubernetes-client/python/blob/master/kubernetes/docs/V1Volume.md) | List of volumes that can be mounted by containers belonging to the pod. More info: https://kubernetes.io/docs/concepts/storage/volumes | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


