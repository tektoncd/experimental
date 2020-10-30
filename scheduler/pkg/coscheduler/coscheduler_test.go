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
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	clientsetfake "k8s.io/client-go/kubernetes/fake"
	framework "k8s.io/kubernetes/pkg/scheduler/framework/v1alpha1"
	st "k8s.io/kubernetes/pkg/scheduler/testing"
)

func TestLess(t *testing.T) {
	labels1 := map[string]string{
		PodGroupName:  "pg1",
		PodGroupTotal: "3",
	}
	labels2 := map[string]string{
		PodGroupName:  "pg2",
		PodGroupTotal: "5",
	}

	var lowPriority, highPriority = int32(10), int32(100)
	t1 := time.Now()
	t2 := t1.Add(time.Second)
	for _, tt := range []struct {
		name     string
		p1       *framework.PodInfo
		p2       *framework.PodInfo
		expected bool
	}{
		{
			name: "p1.priority less than p2.priority",
			p1: &framework.PodInfo{
				Pod: &v1.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: "pod1", Namespace: "namespace1"},
					Spec: v1.PodSpec{
						Priority: &lowPriority,
					},
				},
			},
			p2: &framework.PodInfo{
				Pod: &v1.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: "pod2", Namespace: "namespace2"},
					Spec: v1.PodSpec{
						Priority: &highPriority,
					},
				},
			},
			expected: false, // p2 should be ahead of p1 in the queue
		},
		{
			name: "p1.priority greater than p2.priority",
			p1: &framework.PodInfo{
				Pod: &v1.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: "pod1", Namespace: "namespace1"},
					Spec: v1.PodSpec{
						Priority: &highPriority,
					},
				},
			},
			p2: &framework.PodInfo{
				Pod: &v1.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: "pod2", Namespace: "namespace2"},
					Spec: v1.PodSpec{
						Priority: &lowPriority,
					},
				},
			},
			expected: true, // p1 should be ahead of p2 in the queue
		},
		{
			name: "equal priority. p1 is added to schedulingQ earlier than p2",
			p1: &framework.PodInfo{
				Pod: &v1.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: "pod1", Namespace: "namespace1"},
					Spec: v1.PodSpec{
						Priority: &highPriority,
					},
				},
				InitialAttemptTimestamp: t1,
			},
			p2: &framework.PodInfo{
				Pod: &v1.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: "pod2", Namespace: "namespace2"},
					Spec: v1.PodSpec{
						Priority: &highPriority,
					},
				},
				InitialAttemptTimestamp: t2,
			},
			expected: true, // p1 should be ahead of p2 in the queue
		},
		{
			name: "equal priority. p2 is added to schedulingQ earlier than p1",
			p1: &framework.PodInfo{
				Pod: &v1.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: "pod1", Namespace: "namespace1"},
					Spec: v1.PodSpec{
						Priority: &highPriority,
					},
				},
				InitialAttemptTimestamp: t2,
			},
			p2: &framework.PodInfo{
				Pod: &v1.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: "pod2", Namespace: "namespace2"},
					Spec: v1.PodSpec{
						Priority: &highPriority,
					},
				},
				InitialAttemptTimestamp: t1,
			},
			expected: false, // p2 should be ahead of p1 in the queue
		},
		{
			name: "p1.priority less than p2.priority, p1 belongs to podGroup1",
			p1: &framework.PodInfo{
				Pod: &v1.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: "pod1", Namespace: "namespace1", Labels: labels1},
					Spec: v1.PodSpec{
						Priority: &lowPriority,
					},
				},
			},
			p2: &framework.PodInfo{
				Pod: &v1.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: "pod2", Namespace: "namespace2"},
					Spec: v1.PodSpec{
						Priority: &highPriority,
					},
				},
			},
			expected: false, // p2 should be ahead of p1 in the queue
		},
		{
			name: "p1.priority greater than p2.priority, p1 belongs to podGroup1",
			p1: &framework.PodInfo{
				Pod: &v1.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: "pod1", Namespace: "namespace1", Labels: labels1},
					Spec: v1.PodSpec{
						Priority: &highPriority,
					},
				},
			},
			p2: &framework.PodInfo{
				Pod: &v1.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: "pod2", Namespace: "namespace2"},
					Spec: v1.PodSpec{
						Priority: &lowPriority,
					},
				},
			},
			expected: true, // p1 should be ahead of p2 in the queue
		},
		{
			name: "equal priority. p1 is added to schedulingQ earlier than p2, p1 belongs to podGroup1",
			p1: &framework.PodInfo{
				Pod: &v1.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: "pod1", Namespace: "namespace1", Labels: labels1},
					Spec: v1.PodSpec{
						Priority: &highPriority,
					},
				},
				InitialAttemptTimestamp: t1,
			},
			p2: &framework.PodInfo{
				Pod: &v1.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: "pod2", Namespace: "namespace2"},
					Spec: v1.PodSpec{
						Priority: &highPriority,
					},
				},
				InitialAttemptTimestamp: t2,
			},
			expected: true, // p1 should be ahead of p2 in the queue
		},
		{
			name: "equal priority. p2 is added to schedulingQ earlier than p1, p1 belongs to podGroup1",
			p1: &framework.PodInfo{
				Pod: &v1.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: "pod1", Namespace: "namespace1", Labels: labels1},
					Spec: v1.PodSpec{
						Priority: &highPriority,
					},
				},
				InitialAttemptTimestamp: t2,
			},
			p2: &framework.PodInfo{
				Pod: &v1.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: "pod2", Namespace: "namespace2"},
					Spec: v1.PodSpec{
						Priority: &highPriority,
					},
				},
				InitialAttemptTimestamp: t1,
			},
			expected: false, // p2 should be ahead of p1 in the queue
		},

		{
			name: "p1.priority less than p2.priority, p1 belongs to podGroup1 and p2 belongs to podGroup2",
			p1: &framework.PodInfo{
				Pod: &v1.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: "pod1", Namespace: "namespace1", Labels: labels1},
					Spec: v1.PodSpec{
						Priority: &lowPriority,
					},
				},
			},
			p2: &framework.PodInfo{
				Pod: &v1.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: "pod2", Namespace: "namespace2", Labels: labels2},
					Spec: v1.PodSpec{
						Priority: &highPriority,
					},
				},
			},
			expected: false, // p2 should be ahead of p1 in the queue
		},
		{
			name: "p1.priority greater than p2.priority, p1 belongs to podGroup1 and p2 belongs to podGroup2",
			p1: &framework.PodInfo{
				Pod: &v1.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: "pod1", Namespace: "namespace1", Labels: labels1},
					Spec: v1.PodSpec{
						Priority: &highPriority,
					},
				},
			},
			p2: &framework.PodInfo{
				Pod: &v1.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: "pod2", Namespace: "namespace2", Labels: labels2},
					Spec: v1.PodSpec{
						Priority: &lowPriority,
					},
				},
			},
			expected: true, // p1 should be ahead of p2 in the queue
		},
		{
			name: "equal priority. p1 is added to schedulingQ earlier than p2, p1 belongs to podGroup1 and p2 belongs to podGroup2",
			p1: &framework.PodInfo{
				Pod: &v1.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: "pod1", Namespace: "namespace1", Labels: labels1},
					Spec: v1.PodSpec{
						Priority: &highPriority,
					},
				},
				InitialAttemptTimestamp: t1,
			},
			p2: &framework.PodInfo{
				Pod: &v1.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: "pod2", Namespace: "namespace2", Labels: labels2},
					Spec: v1.PodSpec{
						Priority: &highPriority,
					},
				},
				InitialAttemptTimestamp: t2,
			},
			expected: true, // p1 should be ahead of p2 in the queue
		},
		{
			name: "equal priority. p2 is added to schedulingQ earlier than p1, p1 belongs to podGroup1 and p2 belongs to podGroup2",
			p1: &framework.PodInfo{
				Pod: &v1.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: "pod1", Namespace: "namespace1", Labels: labels1},
					Spec: v1.PodSpec{
						Priority: &highPriority,
					},
				},
				InitialAttemptTimestamp: t2,
			},
			p2: &framework.PodInfo{
				Pod: &v1.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: "pod2", Namespace: "namespace2", Labels: labels2},
					Spec: v1.PodSpec{
						Priority: &highPriority,
					},
				},
				InitialAttemptTimestamp: t1,
			},
			expected: false, // p2 should be ahead of p1 in the queue
		},
		{
			name: "equal priority and creation time, p1 belongs to podGroup1 and p2 belongs to podGroup2",
			p1: &framework.PodInfo{
				Pod: &v1.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: "pod1", Namespace: "namespace1", Labels: labels1},
					Spec: v1.PodSpec{
						Priority: &highPriority,
					},
				},
				InitialAttemptTimestamp: t1,
			},
			p2: &framework.PodInfo{
				Pod: &v1.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: "pod2", Namespace: "namespace2", Labels: labels2},
					Spec: v1.PodSpec{
						Priority: &highPriority,
					},
				},
				InitialAttemptTimestamp: t1,
			},
			expected: true, // p1 should be ahead of p2 in the queue
		},
		{
			name: "equal priority and creation time, p2 belong to podGroup2",
			p1: &framework.PodInfo{
				Pod: &v1.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: "pod1", Namespace: "namespace1"},
					Spec: v1.PodSpec{
						Priority: &highPriority,
					},
				},
				InitialAttemptTimestamp: t1,
			},
			p2: &framework.PodInfo{
				Pod: &v1.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: "pod2", Namespace: "namespace2", Labels: labels2},
					Spec: v1.PodSpec{
						Priority: &highPriority,
					},
				},
				InitialAttemptTimestamp: t1,
			},
			expected: true, // p1 should be ahead of p2 in the queue
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			coscheduler := &Coscheduler{}
			if got := coscheduler.Less(tt.p1, tt.p2); got != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}

func TestPreFilter(t *testing.T) {
	tests := []struct {
		name     string
		pod      *v1.Pod
		pods     []*v1.Pod
		podInfos []PodGroupInfo
		expected framework.Code
	}{
		{
			name: "pod does not belong to any podGroup",
			pod:  st.MakePod().Name("p").UID("p").Namespace("ns1").Obj(),
			pods: []*v1.Pod{
				st.MakePod().Name("pg1-1").UID("pg1-1").Namespace("ns1").Label(PodGroupName, "pg1").Obj(),
				st.MakePod().Name("pg2-1").UID("pg2-1").Namespace("ns1").Label(PodGroupName, "pg2").Obj(),
			},
			expected: framework.Success,
		},
		{
			name: "pod belongs to podGroup1 and its PodGroupTotal does not match the group's",
			pod:  st.MakePod().Name("p").UID("p").Namespace("ns1").Label(PodGroupName, "pg1").Label(PodGroupTotal, "2").Obj(),
			pods: []*v1.Pod{
				st.MakePod().Name("pg1-1").UID("pg1-1").Namespace("ns1").Label(PodGroupName, "pg1").Label(PodGroupTotal, "3").Obj(),
			},
			expected: framework.Unschedulable,
		},
		{
			name: "pod belongs to podGroup1 and its priority does not match the group's",
			pod:  st.MakePod().Name("p").UID("p").Namespace("ns1").Priority(20).Label(PodGroupName, "pg1").Label(PodGroupTotal, "2").Obj(),
			pods: []*v1.Pod{
				st.MakePod().Name("pg1-1").UID("pg1-1").Namespace("ns1").Priority(10).Label(PodGroupName, "pg1").Label(PodGroupTotal, "2").Obj(),
			},
			expected: framework.Unschedulable,
		},
		{
			name: "pod belongs to podGroup1, the number of total pods is less than PodGroupTotal",
			pod:  st.MakePod().Name("p").UID("p").Namespace("ns1").Label(PodGroupName, "pg1").Label(PodGroupTotal, "3").Obj(),
			pods: []*v1.Pod{
				st.MakePod().Name("pg1-1").UID("pg1-1").Namespace("ns1").Label(PodGroupName, "pg1").Label(PodGroupTotal, "2").Obj(),
				st.MakePod().Name("pg2-1").UID("pg2-1").Namespace("ns1").Label(PodGroupName, "pg2").Label(PodGroupTotal, "1").Obj(),
			},
			expected: framework.Unschedulable,
		},
		{
			name: "pod belongs to podGroup2, the number of total pods is more than PodGroupTotal",
			pod:  st.MakePod().Name("p").UID("p").Namespace("ns1").Label(PodGroupName, "pg2").Label(PodGroupTotal, "2").Obj(),
			pods: []*v1.Pod{
				st.MakePod().Name("pg2-1").UID("pg2-1").Namespace("ns1").Label(PodGroupName, "pg2").Label(PodGroupTotal, "2").Obj(),
				st.MakePod().Name("pg2-2").UID("pg2-2").Namespace("ns1").Label(PodGroupName, "pg2").Label(PodGroupTotal, "2").Obj(),
				st.MakePod().Name("pg2-2").UID("pg2-3").Namespace("ns1").Label(PodGroupName, "pg2").Label(PodGroupTotal, "2").Obj(),
				st.MakePod().Name("pg1-1").UID("pg1-1").Namespace("ns1").Label(PodGroupName, "pg1").Label(PodGroupTotal, "1").Obj(),
			},
			expected: framework.Unschedulable,
		},
		{
			name: "pod belongs to podGroup1, one of the pod in podGroup1 has been binded",
			pod:  st.MakePod().Name("p").UID("p").Namespace("ns1").Label(PodGroupName, "pg1").Label(PodGroupTotal, "2").Obj(),
			pods: []*v1.Pod{
				st.MakePod().Name("pg1-1").UID("pg1-1").Namespace("ns1").Label(PodGroupName, "pg1").Label(PodGroupTotal, "2").Obj(),
				st.MakePod().Name("pg2-1").UID("pg2-1").Namespace("ns1").Label(PodGroupName, "pg2").Label(PodGroupTotal, "1").Obj(),
			},
			podInfos: []PodGroupInfo{
				{
					key:      "ns1/pg1",
					nodeName: "yes",
				},
			},
			expected: framework.Success,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := clientsetfake.NewSimpleClientset()
			informerFactory := informers.NewSharedInformerFactory(cs, 0)
			podInformer := informerFactory.Core().V1().Pods()
			coscheduler := &Coscheduler{podLister: podInformer.Lister()}
			for _, p := range tt.pods {
				coscheduler.getOrCreatePodGroupInfo(p, time.Now())
				podInformer.Informer().GetStore().Add(p)
			}

			podInformer.Informer().GetStore().Add(tt.pod)
			cycleState := framework.NewCycleState()
			if got := coscheduler.PreFilter(nil, cycleState, tt.pod); got.Code() != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, got.Code())
			}
		})
	}
}
