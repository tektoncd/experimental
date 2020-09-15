/*
Copyright 2020 The Tekton Authors

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

package convert

import (
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes"
	durpb "github.com/golang/protobuf/ptypes/duration"
	tspb "github.com/golang/protobuf/ptypes/timestamp"
	"github.com/google/go-cmp/cmp"
	pb "github.com/tektoncd/experimental/results/proto/proto"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"google.golang.org/protobuf/testing/protocmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"
)

func TestToProto(t *testing.T) {
	n := time.Now()
	nextTime := func() metav1.Time {
		n = n.Add(time.Hour)
		return metav1.Time{Time: n}
	}
	create, delete, start, finish := nextTime(), nextTime(), nextTime(), nextTime()

	got, err := ToProto(&v1beta1.TaskRun{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "api-version",
			Kind:       "kind",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:              "name",
			GenerateName:      "generate-name",
			Namespace:         "namespace",
			UID:               "uid",
			Generation:        12345,
			CreationTimestamp: create,
			DeletionTimestamp: &delete,
			Labels: map[string]string{
				"label-one": "one",
				"label-two": "two",
			},
			Annotations: map[string]string{
				"annotation-one": "one",
				"annotation-two": "two",
			},
		},
		Spec: v1beta1.TaskRunSpec{
			Timeout: &metav1.Duration{Duration: time.Hour},
			TaskSpec: &v1beta1.TaskSpec{
				Steps: []v1beta1.Step{{
					Script: "script",
					Container: corev1.Container{
						Name:       "name",
						Image:      "image",
						Command:    []string{"cmd1", "cmd2"},
						Args:       []string{"arg1", "arg2"},
						WorkingDir: "workingdir",
						Env: []corev1.EnvVar{{
							Name:  "env1",
							Value: "ENV1",
						}, {
							Name:  "env2",
							Value: "ENV2",
						}},
						VolumeMounts: []corev1.VolumeMount{{
							Name:      "vm1",
							MountPath: "path1",
							ReadOnly:  false,
							SubPath:   "subpath1",
						}, {
							Name:      "vm2",
							MountPath: "path2",
							ReadOnly:  true,
							SubPath:   "subpath2",
						}},
					},
				}, {
					Container: corev1.Container{Name: "step2"},
				}},
				Sidecars: []v1beta1.Step{{
					Container: corev1.Container{Name: "sidecar1"},
				}, {
					Container: corev1.Container{Name: "sidecar2"},
				}},
				Volumes: []corev1.Volume{{
					Name:         "volname1",
					VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
				}, {
					Name:         "volname2",
					VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
				}},
			},
		},
		Status: v1beta1.TaskRunStatus{
			Status: duckv1beta1.Status{
				ObservedGeneration: 23456,
				Conditions: []apis.Condition{{
					Type:               "type",
					Status:             "status",
					Severity:           "omgbad",
					LastTransitionTime: apis.VolatileTime{Inner: finish},
					Reason:             "reason",
					Message:            "message",
				}, {
					Type: "another condition",
				}},
			},
			TaskRunStatusFields: v1beta1.TaskRunStatusFields{
				PodName:        "podname",
				StartTime:      &start,
				CompletionTime: &finish,
				Steps: []v1beta1.StepState{{
					ContainerState: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{
							ExitCode:    123,
							Signal:      456,
							Reason:      "reason",
							Message:     "message",
							StartedAt:   start,
							FinishedAt:  finish,
							ContainerID: "containerid",
						},
					},
					Name:          "name",
					ContainerName: "containername",
					ImageID:       "imageid",
				}, {
					Name: "another state",
				}},
			},
		},
	})
	if err != nil {
		t.Fatalf("ToProto: %v", err)
	}

	want := &pb.TaskRun{
		ApiVersion: "api-version",
		Kind:       "kind",
		Metadata: &pb.ObjectMeta{
			Name:              "name",
			GenerateName:      "generate-name",
			Namespace:         "namespace",
			Uid:               "uid",
			Generation:        12345,
			CreationTimestamp: timestamp(&create),
			DeletionTimestamp: timestamp(&delete),
			Labels: map[string]string{
				"label-one": "one",
				"label-two": "two",
			},
			Annotations: map[string]string{
				"annotation-one": "one",
				"annotation-two": "two",
			},
		},
		Spec: &pb.TaskRunSpec{
			Timeout: &durpb.Duration{Seconds: 3600},
			TaskSpec: &pb.TaskSpec{
				Steps: []*pb.Step{{
					Script:     "script",
					Name:       "name",
					Image:      "image",
					Command:    []string{"cmd1", "cmd2"},
					Args:       []string{"arg1", "arg2"},
					WorkingDir: "workingdir",
					Env: []*pb.EnvVar{{
						Name:  "env1",
						Value: "ENV1",
					}, {
						Name:  "env2",
						Value: "ENV2",
					}},
					VolumeMounts: []*pb.VolumeMount{{
						Name:      "vm1",
						MountPath: "path1",
						ReadOnly:  false,
						SubPath:   "subpath1",
					}, {
						Name:      "vm2",
						MountPath: "path2",
						ReadOnly:  true,
						SubPath:   "subpath2",
					}},
				}, {
					Name: "step2",
				}},
				Sidecars: []*pb.Step{{
					Name: "sidecar1",
				}, {
					Name: "sidecar2",
				}},
				Volumes: []*pb.Volume{{
					Name:   "volname1",
					Source: &pb.Volume_EmptyDir{EmptyDir: &pb.EmptyDir{}},
				}, {
					Name:   "volname2",
					Source: &pb.Volume_EmptyDir{EmptyDir: &pb.EmptyDir{}},
				}},
			},
		},
		Status: &pb.TaskRunStatus{
			Conditions: []*pb.Condition{{
				Type:               "type",
				Status:             "status",
				Severity:           "omgbad",
				LastTransitionTime: timestamp(&finish),
				Reason:             "reason",
				Message:            "message",
			}, {
				Type: "another condition",
			}},
			ObservedGeneration: 23456,
			PodName:            "podname",
			StartTime:          timestamp(&start),
			CompletionTime:     timestamp(&finish),
			Steps: []*pb.StepState{{
				Status: &pb.StepState_Terminated{Terminated: &pb.ContainerStateTerminated{
					ExitCode:    123,
					Signal:      456,
					Reason:      "reason",
					Message:     "message",
					StartedAt:   timestamp(&start),
					FinishedAt:  timestamp(&finish),
					ContainerId: "containerid",
				}},
				Name:          "name",
				ContainerName: "containername",
				ImageId:       "imageid",
			}, {
				Name: "another state",
			}},
		},
	}

	if d := cmp.Diff(want, got, protocmp.Transform()); d != "" {
		t.Errorf("Diff(-want,+got): %s", d)
	}
}

func timestamp(t *metav1.Time) *tspb.Timestamp {
	if t == nil {
		return nil
	}
	if t.Time.IsZero() {
		return nil
	}
	p, err := ptypes.TimestampProto(t.Time.Truncate(time.Second))
	if err != nil {
		panic(err.Error())
	}
	return p
}
