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

// Package convert provides a method to convert v1beta1 API objects to Results
// API proto objects.
package convert

import (
	"fmt"
	"log"
	"runtime/debug"

	"github.com/golang/protobuf/ptypes"
	durpb "github.com/golang/protobuf/ptypes/duration"
	tspb "github.com/golang/protobuf/ptypes/timestamp"
	pb "github.com/tektoncd/experimental/results/proto/proto"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
)

// ToProto converts a v1beta1.TaskRun object to the equivalent Results API
// proto message.
func ToProto(tr *v1beta1.TaskRun) (p *pb.TaskRun, err error) {
	defer func() {
		if r := recover(); r != nil && err == nil {
			log.Printf("Recovered panic in ToProto: %s", debug.Stack())
			err = fmt.Errorf("recovered: %v", r)
		}
	}()

	return &pb.TaskRun{
		ApiVersion: tr.APIVersion,
		Kind:       tr.Kind,
		Metadata: &pb.ObjectMeta{
			Name:              tr.ObjectMeta.Name,
			GenerateName:      tr.ObjectMeta.GenerateName,
			Namespace:         tr.ObjectMeta.Namespace,
			Uid:               string(tr.ObjectMeta.UID),
			Generation:        tr.ObjectMeta.Generation,
			CreationTimestamp: timestamp(&tr.ObjectMeta.CreationTimestamp),
			DeletionTimestamp: timestamp(tr.ObjectMeta.DeletionTimestamp),
			Labels:            tr.ObjectMeta.Labels,
			Annotations:       tr.ObjectMeta.Annotations,
		},
		Spec: &pb.TaskRunSpec{
			Timeout:  duration(tr.Spec.Timeout),
			TaskSpec: taskSpec(tr.Spec.TaskSpec),
		},
		Status: &pb.TaskRunStatus{
			Conditions:         conditions(tr.Status.Conditions),
			ObservedGeneration: tr.Status.ObservedGeneration,
			PodName:            tr.Status.PodName,
			StartTime:          timestamp(tr.Status.StartTime),
			CompletionTime:     timestamp(tr.Status.CompletionTime),
			Steps:              stepStates(tr.Status.Steps),
			TaskSpec:           taskSpec(tr.Status.TaskSpec),
		},
	}, nil
}

func stepStates(ss []v1beta1.StepState) []*pb.StepState {
	var out []*pb.StepState
	for _, s := range ss {
		o := &pb.StepState{
			Name:          s.Name,
			ContainerName: s.ContainerName,
			ImageId:       s.ImageID,
		}

		switch {
		case s.Waiting != nil:
			o.Status = &pb.StepState_Waiting{Waiting: &pb.ContainerStateWaiting{
				Reason:  s.Waiting.Reason,
				Message: s.Waiting.Message,
			}}
		case s.Running != nil:
			o.Status = &pb.StepState_Running{Running: &pb.ContainerStateRunning{
				StartedAt: timestamp(&s.Running.StartedAt),
			}}
		case s.Terminated != nil:
			o.Status = &pb.StepState_Terminated{Terminated: &pb.ContainerStateTerminated{
				ExitCode:    s.Terminated.ExitCode,
				Signal:      s.Terminated.Signal,
				Reason:      s.Terminated.Reason,
				Message:     s.Terminated.Message,
				StartedAt:   timestamp(&s.Terminated.StartedAt),
				FinishedAt:  timestamp(&s.Terminated.FinishedAt),
				ContainerId: s.Terminated.ContainerID,
			}}
		}

		out = append(out, o)
	}
	return out
}

func taskSpec(ts *v1beta1.TaskSpec) *pb.TaskSpec {
	if ts == nil {
		return nil
	}
	return &pb.TaskSpec{
		Steps:    steps(ts.Steps),
		Volumes:  volumes(ts.Volumes),
		Sidecars: steps(ts.Sidecars),
	}
}

func conditions(cs []apis.Condition) []*pb.Condition {
	var out []*pb.Condition
	for _, c := range cs {
		out = append(out, &pb.Condition{
			Type:               string(c.Type),
			Status:             string(c.Status),
			Severity:           string(c.Severity),
			LastTransitionTime: timestamp(&c.LastTransitionTime.Inner),
			Reason:             c.Reason,
			Message:            c.Message,
		})
	}
	return out
}

func volumes(vs []corev1.Volume) []*pb.Volume {
	var out []*pb.Volume
	for _, v := range vs {
		out = append(out, &pb.Volume{
			Name:   v.Name,
			Source: &pb.Volume_EmptyDir{EmptyDir: &pb.EmptyDir{}},
		})
	}
	return out
}

func steps(steps []v1beta1.Step) []*pb.Step {
	var out []*pb.Step
	for _, s := range steps {
		out = append(out, &pb.Step{
			Name:         s.Name,
			Image:        s.Image,
			Command:      s.Command,
			Args:         s.Args,
			WorkingDir:   s.WorkingDir,
			Env:          envVars(s.Container.Env),
			VolumeMounts: volumeMounts(s.VolumeMounts),
			Script:       s.Script,
		})
	}
	return out
}

func envVars(vars []corev1.EnvVar) []*pb.EnvVar {
	var out []*pb.EnvVar
	for _, v := range vars {
		out = append(out, &pb.EnvVar{
			Name:  v.Name,
			Value: v.Value,
		})
	}
	return out
}

func volumeMounts(vms []corev1.VolumeMount) []*pb.VolumeMount {
	var out []*pb.VolumeMount
	for _, vm := range vms {
		out = append(out, &pb.VolumeMount{
			Name:      vm.Name,
			MountPath: vm.MountPath,
			ReadOnly:  vm.ReadOnly,
			SubPath:   vm.SubPath,
		})
	}
	return out
}

func timestamp(t *metav1.Time) *tspb.Timestamp {
	if t == nil {
		return nil
	}
	if t.Time.IsZero() {
		return nil
	}
	p, err := ptypes.TimestampProto(t.Time)
	if err != nil {
		panic(err.Error())
	}
	return p
}

func duration(d *metav1.Duration) *durpb.Duration {
	if d == nil {
		return nil
	}
	return ptypes.DurationProto(d.Duration)
}
