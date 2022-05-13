package pipelineinpod

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/test/diff"
	corev1 "k8s.io/api/core/v1"
)

var volumeMount = corev1.VolumeMount{
	Name:      "my-mount",
	MountPath: "/mount/point",
}
var volumeMountSort = cmpopts.SortSlices(func(i, j corev1.VolumeMount) bool { return i.Name < j.Name })

func TestCreateContainersSingleTask(t *testing.T) {
	tasks := []pipelineTaskContainers{{
		pt: &v1beta1.PipelineTask{Name: "pipeline-task"},
		containers: []corev1.Container{{
			Image:   "step-1",
			Command: []string{"cmd"},
			Args:    []string{"arg1", "arg2"},
		}, {
			Image:        "step-2",
			Command:      []string{"cmd1", "cmd2", "cmd3"}, // multiple cmd elements
			Args:         []string{"arg1", "arg2"},
			VolumeMounts: []corev1.VolumeMount{volumeMount}, // pre-existing volumeMount
		}, {
			Image:   "step-3",
			Command: []string{"cmd"},
			Args:    []string{"arg1", "arg2"},
		}},
	}}
	want := []corev1.Container{{
		Image:   "step-1",
		Command: []string{entrypointBinary},
		Args: []string{
			"-wait_file", "/tekton/downward/ready",
			"-post_file", "/tekton/run/pipeline-task/0/out",
			"-termination_path", "/tekton/termination",
			"-step_metadata_dir", "/tekton/steps",
			"-wait_file_content",
			"-entrypoint", "cmd", "--",
			"arg1", "arg2",
		},
		VolumeMounts: []corev1.VolumeMount{downwardMount, {
			Name:      "tekton-internal-run-pipeline-task-0",
			MountPath: "/tekton/run/pipeline-task/0",
			ReadOnly:  false,
		}, {
			Name:      "tekton-internal-run-pipeline-task-1",
			MountPath: "/tekton/run/pipeline-task/1",
			ReadOnly:  true,
		}, {
			Name:      "tekton-internal-run-pipeline-task-2",
			MountPath: "/tekton/run/pipeline-task/2",
			ReadOnly:  true,
		}},
		TerminationMessagePath: "/tekton/termination",
	}, {
		Image:   "step-2",
		Command: []string{entrypointBinary},
		Args: []string{
			"-wait_file", "/tekton/run/pipeline-task/0/out",
			"-post_file", "/tekton/run/pipeline-task/1/out",
			"-termination_path", "/tekton/termination",
			"-step_metadata_dir", "/tekton/steps",
			"-entrypoint", "cmd1", "--",
			"cmd2", "cmd3",
			"arg1", "arg2",
		},
		VolumeMounts: []corev1.VolumeMount{volumeMount, {
			Name:      "tekton-internal-run-pipeline-task-0",
			MountPath: "/tekton/run/pipeline-task/0",
			ReadOnly:  true,
		}, {
			Name:      "tekton-internal-run-pipeline-task-1",
			MountPath: "/tekton/run/pipeline-task/1",
			ReadOnly:  false,
		}, {
			Name:      "tekton-internal-run-pipeline-task-2",
			MountPath: "/tekton/run/pipeline-task/2",
			ReadOnly:  true,
		}},
		TerminationMessagePath: "/tekton/termination",
	}, {
		Image:   "step-3",
		Command: []string{entrypointBinary},
		Args: []string{
			"-wait_file", "/tekton/run/pipeline-task/1/out",
			"-post_file", "/tekton/run/pipeline-task/2/out",
			"-termination_path", "/tekton/termination",
			"-step_metadata_dir", "/tekton/steps",
			"-entrypoint", "cmd", "--",
			"arg1", "arg2",
		},
		VolumeMounts: []corev1.VolumeMount{{
			Name:      "tekton-internal-run-pipeline-task-0",
			MountPath: "/tekton/run/pipeline-task/0",
			ReadOnly:  true,
		}, {
			Name:      "tekton-internal-run-pipeline-task-1",
			MountPath: "/tekton/run/pipeline-task/1",
			ReadOnly:  true,
		}, {
			Name:      "tekton-internal-run-pipeline-task-2",
			MountPath: "/tekton/run/pipeline-task/2",
			ReadOnly:  false,
		}},
		TerminationMessagePath: "/tekton/termination",
	}}
	gotContainers, _, err := createContainers([]string{}, tasks, nil)
	if err != nil {
		t.Fatalf("createContainers: %v", err)
	}
	if d := cmp.Diff(want, gotContainers); d != "" {
		t.Errorf("Diff %s", diff.PrintWantGot(d))
	}
}

func TestCreateContainersSequentialTasks(t *testing.T) {
	tasks := []pipelineTaskContainers{{
		pt: &v1beta1.PipelineTask{
			Name: "pipeline-task-1",
		},
		containers: []corev1.Container{{
			Image:   "step-1",
			Command: []string{"cmd"},
			Args:    []string{"arg1", "arg2"},
		}, {
			Image:   "step-2",
			Command: []string{"cmd"},
			Args:    []string{"arg1", "arg2"},
		}},
	}, {
		pt: &v1beta1.PipelineTask{
			Name:     "pipeline-task-2",
			RunAfter: []string{"pipeline-task-1"},
		},
		containers: []corev1.Container{{
			Image:   "step-1",
			Command: []string{"cmd"},
			Args:    []string{"arg1", "arg2"},
		}},
	}}

	want := []corev1.Container{{
		Image:   "step-1",
		Command: []string{entrypointBinary},
		Args: []string{
			"-wait_file", "/tekton/downward/ready",
			"-post_file", "/tekton/run/pipeline-task-1/0/out",
			"-termination_path", "/tekton/termination",
			"-step_metadata_dir", "/tekton/steps",
			"-wait_file_content",
			"-entrypoint", "cmd", "--",
			"arg1", "arg2",
		},
		VolumeMounts: []corev1.VolumeMount{downwardMount, {
			Name:      "tekton-internal-run-pipeline-task-1-0",
			MountPath: "/tekton/run/pipeline-task-1/0",
			ReadOnly:  false,
		}, {
			Name:      "tekton-internal-run-pipeline-task-1-1",
			MountPath: "/tekton/run/pipeline-task-1/1",
			ReadOnly:  true,
		}},
		TerminationMessagePath: "/tekton/termination",
	}, {
		Image:   "step-2",
		Command: []string{entrypointBinary},
		Args: []string{
			"-wait_file", "/tekton/run/pipeline-task-1/0/out",
			"-post_file", "/tekton/run/pipeline-task-1/1/out",
			"-termination_path", "/tekton/termination",
			"-step_metadata_dir", "/tekton/steps",
			"-entrypoint", "cmd", "--",
			"arg1", "arg2",
		},
		VolumeMounts: []corev1.VolumeMount{{
			Name:      "tekton-internal-run-pipeline-task-1-0",
			MountPath: "/tekton/run/pipeline-task-1/0",
			ReadOnly:  true,
		}, {
			Name:      "tekton-internal-run-pipeline-task-1-1",
			MountPath: "/tekton/run/pipeline-task-1/1",
			ReadOnly:  false,
		}},
		TerminationMessagePath: "/tekton/termination",
	}, {
		Image:   "step-1",
		Command: []string{entrypointBinary},
		Args: []string{
			"-wait_file", "/tekton/run/pipeline-task-1/1/out",
			"-post_file", "/tekton/run/pipeline-task-2/0/out",
			"-termination_path", "/tekton/termination",
			"-step_metadata_dir", "/tekton/steps",
			"-entrypoint", "cmd", "--",
			"arg1", "arg2",
		},
		VolumeMounts: []corev1.VolumeMount{{
			Name:      "tekton-internal-run-pipeline-task-1-1",
			MountPath: "/tekton/run/pipeline-task-1/1",
			ReadOnly:  true,
		}, {
			Name:      "tekton-internal-run-pipeline-task-2-0",
			MountPath: "/tekton/run/pipeline-task-2/0",
			ReadOnly:  false,
		}},
		TerminationMessagePath: "/tekton/termination",
	}}
	gotContainers, _, err := createContainers([]string{}, tasks, nil)
	if err != nil {
		t.Fatalf("createContainers: %v", err)
	}
	if d := cmp.Diff(want, gotContainers, volumeMountSort); d != "" {
		t.Errorf("Diff %s", diff.PrintWantGot(d))
	}
}

func TestCreateContainersParallelTasks(t *testing.T) {
	tasks := []pipelineTaskContainers{{
		pt: &v1beta1.PipelineTask{
			Name: "pipeline-task-1",
		},
		containers: []corev1.Container{{
			Image:   "step-1",
			Command: []string{"cmd"},
			Args:    []string{"arg1", "arg2"},
		}, {
			Image:   "step-2",
			Command: []string{"cmd"},
			Args:    []string{"arg1", "arg2"},
		}},
	}, {
		pt: &v1beta1.PipelineTask{
			Name: "pipeline-task-2",
		},
		containers: []corev1.Container{{
			Image:   "step-1",
			Command: []string{"cmd"},
			Args:    []string{"arg1", "arg2"},
		}},
	}}

	want := []corev1.Container{{
		Image:   "step-1",
		Command: []string{entrypointBinary},
		Args: []string{
			"-wait_file", "/tekton/downward/ready",
			"-post_file", "/tekton/run/pipeline-task-1/0/out",
			"-termination_path", "/tekton/termination",
			"-step_metadata_dir", "/tekton/steps",
			"-wait_file_content",
			"-entrypoint", "cmd", "--",
			"arg1", "arg2",
		},
		VolumeMounts: []corev1.VolumeMount{downwardMount, {
			Name:      "tekton-internal-run-pipeline-task-1-0",
			MountPath: "/tekton/run/pipeline-task-1/0",
			ReadOnly:  false,
		}, {
			Name:      "tekton-internal-run-pipeline-task-1-1",
			MountPath: "/tekton/run/pipeline-task-1/1",
			ReadOnly:  true,
		}},
		TerminationMessagePath: "/tekton/termination",
	}, {
		Image:   "step-2",
		Command: []string{entrypointBinary},
		Args: []string{
			"-wait_file", "/tekton/run/pipeline-task-1/0/out",
			"-post_file", "/tekton/run/pipeline-task-1/1/out",
			"-termination_path", "/tekton/termination",
			"-step_metadata_dir", "/tekton/steps",
			"-entrypoint", "cmd", "--",
			"arg1", "arg2",
		},
		VolumeMounts: []corev1.VolumeMount{{
			Name:      "tekton-internal-run-pipeline-task-1-0",
			MountPath: "/tekton/run/pipeline-task-1/0",
			ReadOnly:  true,
		}, {
			Name:      "tekton-internal-run-pipeline-task-1-1",
			MountPath: "/tekton/run/pipeline-task-1/1",
			ReadOnly:  false,
		}},
		TerminationMessagePath: "/tekton/termination",
	}, {
		Image:   "step-1",
		Command: []string{entrypointBinary},
		Args: []string{
			"-wait_file", "/tekton/downward/ready",
			"-post_file", "/tekton/run/pipeline-task-2/0/out",
			"-termination_path", "/tekton/termination",
			"-step_metadata_dir", "/tekton/steps",
			"-wait_file_content",
			"-entrypoint", "cmd", "--",
			"arg1", "arg2",
		},
		VolumeMounts: []corev1.VolumeMount{downwardMount, {
			Name:      "tekton-internal-run-pipeline-task-2-0",
			MountPath: "/tekton/run/pipeline-task-2/0",
			ReadOnly:  false,
		}},
		TerminationMessagePath: "/tekton/termination",
	}}
	gotContainers, _, err := createContainers([]string{}, tasks, nil)
	if err != nil {
		t.Fatalf("createContainers: %v", err)
	}
	if d := cmp.Diff(want, gotContainers, volumeMountSort); d != "" {
		t.Errorf("Diff %s", diff.PrintWantGot(d))
	}
}
