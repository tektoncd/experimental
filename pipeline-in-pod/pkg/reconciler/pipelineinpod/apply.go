package pipelineinpod

import (
	"context"
	"fmt"
	"path/filepath"

	cprv1alpha1 "github.com/tektoncd/experimental/pipeline-in-pod/pkg/apis/colocatedpipelinerun/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/names"
	"github.com/tektoncd/pipeline/pkg/reconciler/pipelinerun/resources"
	taskresources "github.com/tektoncd/pipeline/pkg/reconciler/taskrun/resources"
	"github.com/tektoncd/pipeline/pkg/reconciler/volumeclaim"
	"github.com/tektoncd/pipeline/pkg/workspace"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	controller "knative.dev/pkg/controller"
)

const (
	volumeNameBase = "ws"
)

func ApplyParametersToPipeline(p *v1beta1.PipelineSpec, cpr *cprv1alpha1.ColocatedPipelineRun) *v1beta1.PipelineSpec {
	// This assumes that the ColocatedPipelineRun inputs have been validated against what the Pipeline requests.

	// stringReplacements is used for standard single-string stringReplacements, while arrayReplacements contains arrays
	// that need to be further processed.
	stringReplacements := map[string]string{}
	arrayReplacements := map[string][]string{}

	patterns := []string{
		"params.%s",
		"params[%q]",
		"params['%s']",
	}

	// Set all the default stringReplacements
	for _, p := range p.Params {
		if p.Default != nil {
			if p.Default.Type == v1beta1.ParamTypeString {
				for _, pattern := range patterns {
					stringReplacements[fmt.Sprintf(pattern, p.Name)] = p.Default.StringVal
				}
			} else {
				for _, pattern := range patterns {
					arrayReplacements[fmt.Sprintf(pattern, p.Name)] = p.Default.ArrayVal
				}
			}
		}
	}
	// Set and overwrite params with the ones from the PipelineRun
	for _, p := range cpr.Spec.Params {
		if p.Value.Type == v1beta1.ParamTypeString {
			for _, pattern := range patterns {
				stringReplacements[fmt.Sprintf(pattern, p.Name)] = p.Value.StringVal
			}
		} else {
			for _, pattern := range patterns {
				arrayReplacements[fmt.Sprintf(pattern, p.Name)] = p.Value.ArrayVal
			}
		}
	}

	return resources.ApplyReplacements(p, stringReplacements, arrayReplacements)
}

func ApplyParametersToTask(spec *v1beta1.TaskSpec, pt *v1beta1.PipelineTask, defaults ...v1beta1.ParamSpec) *v1beta1.TaskSpec {
	// This assumes that the ColocatedPipelineRun inputs have been validated against what the Task requests.

	// stringReplacements is used for standard single-string stringReplacements, while arrayReplacements contains arrays
	// that need to be further processed.
	if len(spec.Params) > 0 {
		defaults = append(defaults, spec.Params...)
	}
	stringReplacements := map[string]string{}
	arrayReplacements := map[string][]string{}

	patterns := []string{
		"params.%s",
		"params[%q]",
		"params['%s']",
	}

	// Set all the default stringReplacements
	for _, p := range defaults {
		if p.Default != nil {
			if p.Default.Type == v1beta1.ParamTypeString {
				for _, pattern := range patterns {
					stringReplacements[fmt.Sprintf(pattern, p.Name)] = p.Default.StringVal
				}
			} else {
				for _, pattern := range patterns {
					arrayReplacements[fmt.Sprintf(pattern, p.Name)] = p.Default.ArrayVal
				}
			}
		}
	}
	// Set and overwrite params with the ones from the Pipeline Task
	for _, p := range pt.Params {
		if p.Value.Type == v1beta1.ParamTypeString {
			for _, pattern := range patterns {
				stringReplacements[fmt.Sprintf(pattern, p.Name)] = p.Value.StringVal
			}
		} else {
			for _, pattern := range patterns {
				arrayReplacements[fmt.Sprintf(pattern, p.Name)] = p.Value.ArrayVal
			}
		}
	}
	return taskresources.ApplyReplacements(spec, stringReplacements, arrayReplacements)
}

// ApplyWorkspacesToPipeline replaces workspace variables in the given pipeline spec with their
// concrete values.
func ApplyWorkspacesToPipeline(p *v1beta1.PipelineSpec, cpr *cprv1alpha1.ColocatedPipelineRun) *v1beta1.PipelineSpec {
	p = p.DeepCopy()
	replacements := map[string]string{}
	for _, declaredWorkspace := range p.Workspaces {
		key := fmt.Sprintf("workspaces.%s.bound", declaredWorkspace.Name)
		replacements[key] = "false"
	}
	for _, boundWorkspace := range cpr.Spec.Workspaces {
		key := fmt.Sprintf("workspaces.%s.bound", boundWorkspace.Name)
		replacements[key] = "true"
	}
	return resources.ApplyReplacements(p, replacements, map[string][]string{})
}

// ApplyWorkspacesToTasks creates volumes for workspaces, replaces workspace path variables,
// and returns a mapping of workspace names (as specified in the pipeline) to volumes
// and a mapping of pipeline task name to volume mounts.
func ApplyWorkspacesToTasks(ctx context.Context, runMeta metav1.ObjectMeta, cpr *cprv1alpha1.ColocatedPipelineRun) (map[string]corev1.Volume, map[string][]corev1.VolumeMount, error) {
	// Get the randomized volume names assigned to workspace bindings
	workspaceVolumes := createVolumes(cpr.Spec.Workspaces)
	workspaceBindings := make(map[string][]v1beta1.WorkspaceBinding)
	if cpr.Status.PipelineSpec == nil {
		return nil, nil, fmt.Errorf("no pipeline spec")
	}
	for _, pt := range cpr.Status.PipelineSpec.Tasks {
		wbs, _, err := getWorkspaceBindingsForPipelineTask(runMeta, cpr, pt)
		if err != nil {
			return nil, nil, err
		}
		workspaceBindings[pt.Name] = wbs
	}
	err := applyWorkspaceSubstitutionsToTasks(ctx, workspaceVolumes, workspaceBindings, cpr)
	if err != nil {
		cpr.Status.MarkFailed(ReasonInvalidWorkspaceBinding, err.Error())
		return nil, nil, controller.NewPermanentError(err)
	}
	volumeMounts, err := getWorkspaceVolumeMounts(workspaceVolumes, cpr)
	if err != nil {
		cpr.Status.MarkFailed(ReasonInvalidWorkspaceBinding, err.Error())
		return nil, nil, controller.NewPermanentError(err)
	}
	return workspaceVolumes, volumeMounts, nil
}

func applyWorkspaceSubstitutionsToTasks(ctx context.Context, workspaceVolumes map[string]corev1.Volume, workspaceBindings map[string][]v1beta1.WorkspaceBinding,
	cpr *cprv1alpha1.ColocatedPipelineRun) error {
	for i, status := range cpr.Status.ChildStatuses {
		taskSpec := status.Spec
		wbs, ok := workspaceBindings[status.PipelineTaskName]
		if !ok {
			return fmt.Errorf("no workspace bindings for pipeline task %s", status.PipelineTaskName)
		}
		if err := workspace.ValidateBindings(taskSpec.Workspaces, wbs); err != nil {
			return err
		}
		taskSpec = taskresources.ApplyWorkspaces(ctx, taskSpec, taskSpec.Workspaces, wbs, workspaceVolumes)
		taskSpec, err := workspace.Apply(ctx, *taskSpec, wbs, workspaceVolumes)
		if err != nil {
			return err
		}
		cpr.Status.ChildStatuses[i].Spec = taskSpec
	}
	return nil
}

// getWorkspaceVolumeMounts takes a mapping of workspace name to volumes and
// returns a mapping of Pipeline Task name to volume mounts.
func getWorkspaceVolumeMounts(
	volumes map[string]corev1.Volume, cpr *cprv1alpha1.ColocatedPipelineRun) (map[string][]corev1.VolumeMount, error) {
	volumeMounts := make(map[string][]corev1.VolumeMount)
	cprWorkspaces := make(map[string]v1beta1.WorkspaceBinding)
	for _, binding := range cpr.Spec.Workspaces {
		cprWorkspaces[binding.Name] = binding
	}
	taskSpecs := make(map[string]v1beta1.TaskSpec)
	for _, status := range cpr.Status.ChildStatuses {
		taskSpecs[status.PipelineTaskName] = *status.Spec
	}

	for _, pt := range cpr.Status.PipelineSpec.Tasks {
		var ptMounts []corev1.VolumeMount
		for _, ptWorkspaceBinding := range pt.Workspaces {
			pipelineWorkspaceName := ptWorkspaceBinding.Workspace
			vv := volumes[pipelineWorkspaceName]
			wb := cprWorkspaces[pipelineWorkspaceName]
			ts := taskSpecs[pt.Name]
			w, err := getDeclaredWorkspace(ptWorkspaceBinding.Name, ts.Workspaces)
			if err != nil {
				return nil, fmt.Errorf("error with workspaces for pipeline task %s: %s", pt.Name, err)
			}
			volumeMount := corev1.VolumeMount{
				Name:      vv.Name,
				MountPath: w.GetMountPath(),
				SubPath:   wb.SubPath,
				ReadOnly:  w.ReadOnly,
			}
			ptMounts = append(ptMounts, volumeMount)
		}
		volumeMounts[pt.Name] = ptMounts
	}
	return volumeMounts, nil
}

func getDeclaredWorkspace(name string, w []v1beta1.WorkspaceDeclaration) (*v1beta1.WorkspaceDeclaration, error) {
	for _, workspace := range w {
		if workspace.Name == name {
			return &workspace, nil
		}
	}
	return nil, fmt.Errorf("even though validation should have caught it, bound workspace %s did not exist in declared workspaces", name)
}

func createVolumes(wbs []v1beta1.WorkspaceBinding) map[string]corev1.Volume {
	volumes := workspace.CreateVolumes(wbs)
	for _, wb := range wbs {
		_, ok := volumes[wb.Name]
		if !ok {
			name := names.SimpleNameGenerator.RestrictLengthWithRandomSuffix(volumeNameBase)
			ed := corev1.EmptyDirVolumeSource{}
			volumes[wb.Name] = corev1.Volume{Name: name, VolumeSource: corev1.VolumeSource{EmptyDir: &ed}}
		}
	}
	return volumes
}

func getWorkspaceBindingsForPipelineTask(runMeta metav1.ObjectMeta, cpr *cprv1alpha1.ColocatedPipelineRun, pt v1beta1.PipelineTask) ([]v1beta1.WorkspaceBinding, string, error) {
	var workspaces []v1beta1.WorkspaceBinding
	var pipelinePVCWorkspaceName string
	cprWorkspaces := make(map[string]v1beta1.WorkspaceBinding)
	for _, binding := range cpr.Spec.Workspaces {
		cprWorkspaces[binding.Name] = binding
	}
	for _, ws := range pt.Workspaces {
		taskWorkspaceName, pipelineTaskSubPath, pipelineWorkspaceName := ws.Name, ws.SubPath, ws.Workspace
		if b, hasBinding := cprWorkspaces[pipelineWorkspaceName]; hasBinding {
			if b.PersistentVolumeClaim != nil || b.VolumeClaimTemplate != nil {
				pipelinePVCWorkspaceName = pipelineWorkspaceName
			}
			ownerRef := *metav1.NewControllerRef(&v1alpha1.Run{ObjectMeta: metav1.ObjectMeta{Name: runMeta.Name, UID: types.UID(runMeta.UID)}}, groupVersionKind)
			workspaces = append(workspaces, taskWorkspaceByWorkspaceVolumeSource(b, taskWorkspaceName, pipelineTaskSubPath, ownerRef))
		} else {
			return nil, "", fmt.Errorf("expected workspace %q to be provided by colocatedpipelinerun for pipeline task %q", pipelineWorkspaceName, pt.Name)
		}
	}
	return workspaces, pipelinePVCWorkspaceName, nil
}

// taskWorkspaceByWorkspaceVolumeSource is returning the WorkspaceBinding with the TaskRun specified name.
// If the volume source is a volumeClaimTemplate, the template is applied and passed to TaskRun as a persistentVolumeClaim
func taskWorkspaceByWorkspaceVolumeSource(wb v1beta1.WorkspaceBinding, taskWorkspaceName string, pipelineTaskSubPath string, owner metav1.OwnerReference) v1beta1.WorkspaceBinding {
	if wb.VolumeClaimTemplate == nil {
		binding := *wb.DeepCopy()
		binding.Name = taskWorkspaceName
		binding.SubPath = combinedSubPath(wb.SubPath, pipelineTaskSubPath)
		return binding
	}

	// apply template
	binding := v1beta1.WorkspaceBinding{
		SubPath: combinedSubPath(wb.SubPath, pipelineTaskSubPath),
		PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
			ClaimName: volumeclaim.GetPersistentVolumeClaimName(wb.VolumeClaimTemplate, wb, owner),
		},
	}
	binding.Name = taskWorkspaceName
	return binding
}

// combinedSubPath returns the combined value of the optional subPath from workspaceBinding and the optional
// subPath from pipelineTask. If both is set, they are joined with a slash.
func combinedSubPath(workspaceSubPath string, pipelineTaskSubPath string) string {
	if workspaceSubPath == "" {
		return pipelineTaskSubPath
	} else if pipelineTaskSubPath == "" {
		return workspaceSubPath
	}
	return filepath.Join(workspaceSubPath, pipelineTaskSubPath)
}
