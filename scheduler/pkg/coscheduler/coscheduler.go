/*
Copyright 2020 The Tekton Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package coscheduler

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"
	podutil "k8s.io/kubernetes/pkg/api/v1/pod"
	v1helper "k8s.io/kubernetes/pkg/apis/core/v1/helper"
	"k8s.io/kubernetes/pkg/features"
	framework "k8s.io/kubernetes/pkg/scheduler/framework/v1alpha1"
	schedulernodeinfo "k8s.io/kubernetes/pkg/scheduler/nodeinfo"
)

var _ framework.QueueSortPlugin = &Coscheduler{}
var _ framework.PreFilterPlugin = &Coscheduler{}
var _ framework.FilterPlugin = &Coscheduler{}
var _ framework.PostBindPlugin = &Coscheduler{}

const (
	// CoschedulerName is the name of the plugin used in the plugin registry and configurations.
	CoschedulerName = "CoschedulerSamenode"
	// PodGroupName is the name of a pod group that mark pods which should be scheduler to same node.
	PodGroupName = "pod-group.scheduling.sigs.k8s.io/name"
	// PodGroupTotal is the number of pods in the pod group
	PodGroupTotal = "pod-group.scheduling.sigs.k8s.io/total"
	// preFilterStateKey is the key in CycleState to CoschedulerSamenode pre-computed data.
	// Using the name of the plugin will likely help us avoid collisions with other plugins.
	preFilterStateKey = "PreFilter" + CoschedulerName
)

// Coscheduler is a plugin that checks if a node has sufficient resources.
type Coscheduler struct {
	frameworkHandle framework.FrameworkHandle
	podLister       corelisters.PodLister
	// key is <namespace>/<PodGroup name> and value is *PodGroupInfo.
	podGroupInfos    sync.Map
	ignoredResources sets.String
}

// PodGroupInfo is a wrapper to a PodGroup with additional information.
// A PodGroup's priority, timestamp are set according to
// the values of the PodGroup's first pod that is added to the scheduling queue.
type PodGroupInfo struct {
	// key is a unique PodGroup ID and currently implemented as <namespace>/<PodGroup name>.
	key string
	// name is the PodGroup name and defined through a Pod label.
	// The PodGroup name of a regular pod is empty.
	name string
	// priority is the priority of pods in a PodGroup.
	// All pods in a PodGroup should have the same priority.
	priority int32
	// timestamp stores the initialization timestamp of a PodGroup.
	timestamp time.Time
	// nodename stores the node name of pods will bind to.
	nodeName string
	// total is the total number of pod in this pod group.
	total int
	// count is the count of pod which has been binded, when reach total, the pod group will be removed.
	count int
}

// preFilterState computed at PreFilter and used at Filter.
type preFilterState struct {
	schedulernodeinfo.Resource
}

// Clone the prefilter state.
func (s *preFilterState) Clone() framework.StateData {
	return s
}

// Name returns name of the plugin. It is used in logs, etc.
func (c *Coscheduler) Name() string {
	return CoschedulerName
}

// NewCoscheduler initializes a new plugin and returns it.
func NewCoscheduler(_ *runtime.Unknown, handle framework.FrameworkHandle) (framework.Plugin, error) {
	podLister := handle.SharedInformerFactory().Core().V1().Pods().Lister()

	c := &Coscheduler{frameworkHandle: handle,
		podLister: podLister,
	}

	podInformer := handle.SharedInformerFactory().Core().V1().Pods().Informer()
	podInformer.AddEventHandler(
		cache.FilteringResourceEventHandler{
			FilterFunc: func(obj interface{}) bool {
				switch t := obj.(type) {
				case *v1.Pod:
					return responsibleForPod(t)
				case cache.DeletedFinalStateUnknown:
					if pod, ok := t.Obj.(*v1.Pod); ok {
						return responsibleForPod(pod)
					}
					return false
				default:
					return false
				}
			},
			Handler: cache.ResourceEventHandlerFuncs{
				DeleteFunc: c.deletePodGroup,
			},
		},
	)

	return c, nil
}

// Less is used to sort pods in the scheduling queue.
// 1. Compare the priorities of Pods.
// 2. Compare the initialization timestamps of PodGroups/Pods.
// 3. Compare the keys of PodGroups/Pods, i.e., if two pods are tied at priority and creation time, the one without podGroup will go ahead of the one with podGroup.
func (c *Coscheduler) Less(podInfo1, podInfo2 *framework.PodInfo) bool {
	pgInfo1, _ := c.getOrCreatePodGroupInfo(podInfo1.Pod, podInfo1.InitialAttemptTimestamp)
	pgInfo2, _ := c.getOrCreatePodGroupInfo(podInfo2.Pod, podInfo2.InitialAttemptTimestamp)

	priority1 := pgInfo1.priority
	priority2 := pgInfo2.priority

	if priority1 != priority2 {
		return priority1 > priority2
	}

	time1 := pgInfo1.timestamp
	time2 := pgInfo2.timestamp

	if !time1.Equal(time2) {
		return time1.Before(time2)
	}

	return pgInfo1.key < pgInfo2.key
}

// getOrCreatePodGroupInfo returns the existing PodGroup in PodGroupInfos if present.
// Otherwise, it creates a PodGroup and returns the value, It stores
// the created PodGroup in PodGroupInfo if the pod defines a PodGroup.
func (c *Coscheduler) getOrCreatePodGroupInfo(pod *v1.Pod, ts time.Time) (*PodGroupInfo, int) {
	podGroupName, podGroupTotal, _ := getPodGroupLabels(pod)

	var pgKey string
	if len(podGroupName) > 0 && podGroupTotal > 0 {
		pgKey = fmt.Sprintf("%v/%v", pod.Namespace, podGroupName)
	}

	// If it is a PodGroup and present in PodGroupInfos, return it.
	if len(pgKey) != 0 {
		pgInfo, exist := c.podGroupInfos.Load(pgKey)
		if exist {
			return pgInfo.(*PodGroupInfo), podGroupTotal
		}
	}

	// If the PodGroup is not present in PodGroupInfos or the pod is a regular pod,
	// create a PodGroup for the Pod and store it in PodGroupInfos if it's not a regular pod.
	pgInfo := &PodGroupInfo{
		name:      podGroupName,
		key:       pgKey,
		priority:  podutil.GetPodPriority(pod),
		timestamp: ts,
		nodeName:  "",
		total:     podGroupTotal,
		count:     0,
	}

	// If it's not a regular Pod, store the PodGroup in PodGroupInfos
	if len(pgKey) > 0 {
		c.podGroupInfos.Store(pgKey, pgInfo)
	}
	return pgInfo, podGroupTotal
}

// getPodGroupLabels checks if the pod belongs to a PodGroup. If so, it will return the
// podGroupName of the PodGroup. If not, it will return "".
func getPodGroupLabels(pod *v1.Pod) (string, int, error) {
	podGroupName, exist := pod.Labels[PodGroupName]
	if !exist || len(podGroupName) == 0 {
		return "", 0, nil
	}

	podGroupTotal, exist := pod.Labels[PodGroupTotal]
	if !exist || len(podGroupTotal) == 0 {
		return "", 0, nil
	}

	total, err := strconv.Atoi(podGroupTotal)
	if err != nil {
		klog.Errorf("PodGroup %v/%v : PodGroupTotal %v is invalid", pod.Namespace, pod.Name, total)
		return "", 0, err
	}
	if total < 1 {
		klog.Errorf("PodGroup %v/%v : PodGroupTotal %v is less than 1", pod.Namespace, pod.Name, total)
		return "", 0, err
	}
	return podGroupName, total, nil
}

// PreFilter invoked at the prefilter extension point.
func (c *Coscheduler) PreFilter(ctx context.Context, cycleState *framework.CycleState, pod *v1.Pod) *framework.Status {
	pgInfo, podTotal := c.getOrCreatePodGroupInfo(pod, time.Now())
	pgKey := pgInfo.key
	if len(pgKey) == 0 {
		return framework.NewStatus(framework.Success, "")
	}

	// Check if the priorities are the same.
	pgPriority := pgInfo.priority
	podPriority := podutil.GetPodPriority(pod)
	if pgPriority != podPriority {
		klog.V(3).Infof("Pod %v has a different priority (%v) as the PodGroup %v (%v)", pod.Name, podPriority, pgKey, pgPriority)
		return framework.NewStatus(framework.Unschedulable, "Priorities do not match")
	}

	// Check if the total are the same.
	pgTotal := pgInfo.total
	if podTotal != pgTotal {
		klog.V(3).Infof("Pod %v has a different total (%v) as the PodGroup %v (%v)", pod.Name, podTotal, pgKey, pgTotal)
		return framework.NewStatus(framework.Unschedulable, "Total do not match")
	}

	if pgInfo.nodeName != "" {
		return framework.NewStatus(framework.Success, "")
	}

	// time.Sleep(time.Duration(5) * time.Second)
	pods, err := c.getGroupPods(pgInfo.name, pod.Namespace)
	if len(pods) != pgInfo.total {
		klog.V(3).Infof("Count of pod: %v not equeal to total: %v in PodGroup %v", len(pods), pgInfo.total, pgKey)
		return framework.NewStatus(framework.Unschedulable, "Count of pod not match total")
	}

	if err != nil || len(pods) == 0 {
		return framework.NewStatus(framework.Unschedulable, "List pods failed")
	}

	cycleState.Write(preFilterStateKey, computePodResourceRequest(pods))
	return framework.NewStatus(framework.Success, "")
}

// computePodResourceRequest returns a framework.Resource that covers the largest
// width in each resource dimension. Because init-containers run sequentially, we collect
// the max in each dimension iteratively. In contrast, we sum the resource vectors for
// regular containers since they run simultaneously.
//
// If Pod Overhead is specified and the feature gate is set, the resources defined for Overhead
// are added to the calculated Resource request sum
//
// Example:
//
// Pod:
//   InitContainers
//     IC1:
//       CPU: 2
//       Memory: 1G
//     IC2:
//       CPU: 2
//       Memory: 3G
//   Containers
//     C1:
//       CPU: 2
//       Memory: 1G
//     C2:
//       CPU: 1
//       Memory: 1G
//
// Result: CPU: 3, Memory: 3G
func computePodResourceRequest(pods []*v1.Pod) *preFilterState {
	result := &preFilterState{}
	resultInitContiner := schedulernodeinfo.Resource{}
	for _, pod := range pods {
		tempResultInitContiner := schedulernodeinfo.Resource{}

		for _, container := range pod.Spec.Containers {
			result.Add(container.Resources.Requests)
		}

		// take max_resource(sum_pod, any_init_container)
		for _, container := range pod.Spec.InitContainers {
			tempResultInitContiner.SetMaxResource(container.Resources.Requests)
		}
		resultInitContiner.Add(tempResultInitContiner.ResourceList())
		// If Overhead is being utilized, add to the total requests for the pod
		if pod.Spec.Overhead != nil && utilfeature.DefaultFeatureGate.Enabled(features.PodOverhead) {
			result.Add(pod.Spec.Overhead)
		}
	}
	result.SetMaxResource(resultInitContiner.ResourceList())

	return result
}

func (c *Coscheduler) getGroupPods(podGroupName, namespace string) ([]*v1.Pod, error) {
	// TODO get the pods from the scheduler cache and queue instead of the hack manner.
	selector := labels.Set{PodGroupName: podGroupName}.AsSelector()
	pods, err := c.podLister.Pods(namespace).List(selector)
	if err != nil {
		klog.Error(err)
		return nil, err
	}
	return pods, nil
}

// PreFilterExtensions returns prefilter extensions, pod add and remove.
func (c *Coscheduler) PreFilterExtensions() framework.PreFilterExtensions {
	return nil
}

func getPreFilterState(cycleState *framework.CycleState) (*preFilterState, error) {
	c, err := cycleState.Read(preFilterStateKey)
	if err != nil {
		// preFilterState doesn't exist, likely PreFilter wasn't invoked.
		return nil, fmt.Errorf("error reading %q from cycleState: %v", preFilterStateKey, err)
	}

	s, ok := c.(*preFilterState)
	if !ok {
		return nil, fmt.Errorf("%+v  convert to NodeResourcesFit.preFilterState error", c)
	}
	return s, nil
}

// Filter invoked at the filter extension point.
// Checks if a node has sufficient resources, such as cpu, memory, gpu, opaque int resources etc to run a pod.
// It returns a list of insufficient resources, if empty, then the node has all the resources requested by the pod.
func (c *Coscheduler) Filter(ctx context.Context, cycleState *framework.CycleState, pod *v1.Pod, nodeInfo *schedulernodeinfo.NodeInfo) *framework.Status {
	pgInfo, _ := c.getOrCreatePodGroupInfo(pod, time.Now())
	pgKey := pgInfo.key
	if len(pgKey) == 0 {
		return framework.NewStatus(framework.Success, "")
	}

	if pgInfo.nodeName == nodeInfo.Node().Name {
		return nil
	}

	s, err := getPreFilterState(cycleState)
	if err != nil {
		return framework.NewStatus(framework.Error, err.Error())
	}

	insufficientResources := fitsRequest(s, nodeInfo, c.ignoredResources)

	if len(insufficientResources) != 0 {
		// We will keep all failure reasons.
		failureReasons := make([]string, 0, len(insufficientResources))
		for _, r := range insufficientResources {
			failureReasons = append(failureReasons, r.Reason)
		}
		return framework.NewStatus(framework.Unschedulable, failureReasons...)
	}

	pgInfo.nodeName = nodeInfo.Node().Name
	return nil
}

// InsufficientResource describes what kind of resource limit is hit and caused the pod to not fit the node.
type InsufficientResource struct {
	ResourceName v1.ResourceName
	// We explicitly have a parameter for reason to avoid formatting a message on the fly
	// for common resources, which is expensive for cluster autoscaler simulations.
	Reason    string
	Requested int64
	Used      int64
	Capacity  int64
}

func fitsRequest(podRequest *preFilterState, nodeInfo *schedulernodeinfo.NodeInfo, ignoredExtendedResources sets.String) []InsufficientResource {
	insufficientResources := make([]InsufficientResource, 0, 4)

	allowedPodNumber := nodeInfo.AllowedPodNumber()
	if len(nodeInfo.Pods())+1 > allowedPodNumber {
		insufficientResources = append(insufficientResources, InsufficientResource{
			v1.ResourcePods,
			"Too many pods",
			1,
			int64(len(nodeInfo.Pods())),
			int64(allowedPodNumber),
		})
	}

	if ignoredExtendedResources == nil {
		ignoredExtendedResources = sets.NewString()
	}

	if podRequest.MilliCPU == 0 &&
		podRequest.Memory == 0 &&
		podRequest.EphemeralStorage == 0 &&
		len(podRequest.ScalarResources) == 0 {
		return insufficientResources
	}

	allocatable := nodeInfo.AllocatableResource()
	if allocatable.MilliCPU < podRequest.MilliCPU+nodeInfo.RequestedResource().MilliCPU {
		insufficientResources = append(insufficientResources, InsufficientResource{
			v1.ResourceCPU,
			"Insufficient cpu",
			podRequest.MilliCPU,
			nodeInfo.RequestedResource().MilliCPU,
			allocatable.MilliCPU,
		})
	}
	if allocatable.Memory < podRequest.Memory+nodeInfo.RequestedResource().Memory {
		insufficientResources = append(insufficientResources, InsufficientResource{
			v1.ResourceMemory,
			"Insufficient memory",
			podRequest.Memory,
			nodeInfo.RequestedResource().Memory,
			allocatable.Memory,
		})
	}
	if allocatable.EphemeralStorage < podRequest.EphemeralStorage+nodeInfo.RequestedResource().EphemeralStorage {
		insufficientResources = append(insufficientResources, InsufficientResource{
			v1.ResourceEphemeralStorage,
			"Insufficient ephemeral-storage",
			podRequest.EphemeralStorage,
			nodeInfo.RequestedResource().EphemeralStorage,
			allocatable.EphemeralStorage,
		})
	}

	for rName, rQuant := range podRequest.ScalarResources {
		if v1helper.IsExtendedResourceName(rName) {
			// If this resource is one of the extended resources that should be
			// ignored, we will skip checking it.
			if ignoredExtendedResources.Has(string(rName)) {
				continue
			}
		}
		if allocatable.ScalarResources[rName] < rQuant+nodeInfo.RequestedResource().ScalarResources[rName] {
			insufficientResources = append(insufficientResources, InsufficientResource{
				rName,
				fmt.Sprintf("Insufficient %v", rName),
				podRequest.ScalarResources[rName],
				nodeInfo.RequestedResource().ScalarResources[rName],
				allocatable.ScalarResources[rName],
			})
		}
	}

	return insufficientResources
}

// PostBind is to clear Pginfo when every pod in the group is binded.
func (c *Coscheduler) PostBind(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string) {
	pgInfo, _ := c.getOrCreatePodGroupInfo(pod, time.Now())
	pgKey := pgInfo.key
	if len(pgKey) == 0 {
		return
	}

	if pgInfo.nodeName == nodeName {
		pgInfo.count++
		if pgInfo.count == pgInfo.total {
			c.podGroupInfos.Delete(pgKey)
		}
	}
}

// responsibleForPod selects pod that belongs to a PodGroup.
func responsibleForPod(pod *v1.Pod) bool {
	podGroupName, podGroupTotal, _ := getPodGroupLabels(pod)
	if len(podGroupName) == 0 || podGroupTotal == 0 {
		return false
	}
	return true
}

// markPodGroupAsExpired set the deletionTimestamp of PodGroup to mark PodGroup as expired.
func (c *Coscheduler) deletePodGroup(obj interface{}) {
	pod := obj.(*v1.Pod)
	podGroupName, podGroupTotal, _ := getPodGroupLabels(pod)
	if len(podGroupName) == 0 || podGroupTotal == 0 {
		return
	}

	pgKey := fmt.Sprintf("%v/%v", pod.Namespace, podGroupName)
	// If it's a PodGroup and present in PodGroupInfos, set its deletionTimestamp.
	_, exist := c.podGroupInfos.Load(pgKey)
	if !exist {
		return
	}

	c.podGroupInfos.Delete(pgKey)
}
