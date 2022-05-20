package pipelineinpod

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	cprv1alpha1 "github.com/tektoncd/experimental/pipeline-in-pod/pkg/apis/colocatedpipelinerun/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/test/names"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestApplyWorkspacesToTasks(t *testing.T) {
	names.TestingSeed()
	tcs := []struct {
		name             string
		cpr              cprv1alpha1.ColocatedPipelineRun
		wantVolumes      map[string]corev1.Volume
		wantVolumeMounts map[string][]corev1.VolumeMount
	}{{
		name: "workspace used in sequential tasks",
		cpr: cprv1alpha1.ColocatedPipelineRun{
			Spec: cprv1alpha1.ColocatedPipelineRunSpec{
				Workspaces: []v1beta1.WorkspaceBinding{{
					Name:     "source-code",
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				}},
				PipelineSpec: &v1beta1.PipelineSpec{
					Workspaces: []v1beta1.PipelineWorkspaceDeclaration{{Name: "source-code"}},
					Tasks: []v1beta1.PipelineTask{{
						Name: "clone",
						Workspaces: []v1beta1.WorkspacePipelineTaskBinding{{
							Name:      "output",
							Workspace: "source-code",
						}},
					}, {
						Name: "build",
						Workspaces: []v1beta1.WorkspacePipelineTaskBinding{{
							Name:      "input",
							Workspace: "source-code",
						}},
					}, {
						Name: "unrelated-task",
					}}},
			},
			Status: cprv1alpha1.ColocatedPipelineRunStatus{
				ColocatedPipelineRunStatusFields: cprv1alpha1.ColocatedPipelineRunStatusFields{
					PipelineSpec: &v1beta1.PipelineSpec{
						Workspaces: []v1beta1.PipelineWorkspaceDeclaration{{Name: "source-code"}},
						Tasks: []v1beta1.PipelineTask{{
							Name: "clone",
							Workspaces: []v1beta1.WorkspacePipelineTaskBinding{{
								Name:      "output",
								Workspace: "source-code",
							}},
						}, {
							Name: "build",
							Workspaces: []v1beta1.WorkspacePipelineTaskBinding{{
								Name:      "input",
								Workspace: "source-code",
							}},
						}, {
							Name: "unrelated-task",
						}}},
					ChildStatuses: []cprv1alpha1.ChildStatus{{
						PipelineTaskName: "clone",
						Spec:             &v1beta1.TaskSpec{Workspaces: []v1beta1.WorkspaceDeclaration{{Name: "output"}}},
					}, {
						PipelineTaskName: "build",
						Spec:             &v1beta1.TaskSpec{Workspaces: []v1beta1.WorkspaceDeclaration{{Name: "input"}}},
					}, {
						PipelineTaskName: "unrelated-task",
						Spec:             &v1beta1.TaskSpec{},
					}},
				},
			},
		},
		wantVolumes: map[string]corev1.Volume{"source-code": {Name: "ws-9l9zj", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}}},
		wantVolumeMounts: map[string][]corev1.VolumeMount{"clone": {{
			Name: "ws-9l9zj", MountPath: "/workspace/output",
		}}, "build": {{
			Name: "ws-9l9zj", MountPath: "/workspace/input",
		}}, "unrelated-task": nil},
	}}
	runMeta := metav1.ObjectMeta{Name: "foo", Namespace: "default"}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			gotVolumes, gotVolumeMounts, err := ApplyWorkspacesToTasks(context.Background(), runMeta, &tc.cpr)
			if err != nil {
				t.Errorf("unexpected error applying workspaces to tasks: %s", err)
			}
			if d := cmp.Diff(tc.wantVolumes, gotVolumes); d != "" {
				t.Errorf("Wrong volumes: %s", d)
			}
			if d := cmp.Diff(tc.wantVolumeMounts, gotVolumeMounts); d != "" {
				t.Errorf("Wrong volume mounts: %s", d)
			}
		})
	}
}
