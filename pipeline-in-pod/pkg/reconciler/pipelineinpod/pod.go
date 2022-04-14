package pipelineinpod

import (
	"context"
	"encoding/base64"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	cprv1alpha1 "github.com/tektoncd/experimental/pipeline-in-pod/pkg/apis/colocatedpipelinerun/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/names"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"knative.dev/pkg/kmeta"
)

const (
	scriptsVolumeName      = "tekton-internal-scripts"
	debugScriptsVolumeName = "tekton-internal-debug-scripts"
	debugInfoVolumeName    = "tekton-internal-debug-info"
	scriptsDir             = "/tekton/scripts"
	debugScriptsDir        = "/tekton/debug/scripts"
	defaultScriptPreamble  = "#!/bin/sh\nset -xe\n"
	debugInfoDir           = "/tekton/debug/info"
	ReleaseAnnotation      = "pipeline.tekton.dev/release"
)

type StepInfo map[string]string

var (
	// Volume definition attached to Pods generated from TaskRuns that have
	// steps that specify a Script.
	scriptsVolume = corev1.Volume{
		Name:         scriptsVolumeName,
		VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
	}
	scriptsVolumeMount = corev1.VolumeMount{
		Name:      scriptsVolumeName,
		MountPath: scriptsDir,
		ReadOnly:  true,
	}
	writeScriptsVolumeMount = corev1.VolumeMount{
		Name:      scriptsVolumeName,
		MountPath: scriptsDir,
		ReadOnly:  false,
	}
	debugScriptsVolume = corev1.Volume{
		Name:         debugScriptsVolumeName,
		VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
	}
	debugScriptsVolumeMount = corev1.VolumeMount{
		Name:      debugScriptsVolumeName,
		MountPath: debugScriptsDir,
	}
	debugInfoVolume = corev1.Volume{
		Name:         debugInfoVolumeName,
		VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
	}
)

var (
	groupVersionKind = schema.GroupVersionKind{
		Group:   v1alpha1.SchemeGroupVersion.Group,
		Version: v1alpha1.SchemeGroupVersion.Version,
		Kind:    "Run",
	}

	// These are injected into all of the source/step containers.
	implicitVolumeMounts = []corev1.VolumeMount{{
		Name:      "tekton-internal-workspace",
		MountPath: pipeline.WorkspaceDir,
	}, {
		Name:      "tekton-internal-home",
		MountPath: pipeline.HomeDir,
	}, {
		Name:      "tekton-internal-results",
		MountPath: pipeline.DefaultResultPath,
	}, {
		Name:      "tekton-internal-steps",
		MountPath: pipeline.StepsDir,
	}}
	implicitVolumes = []corev1.Volume{{
		Name:         "tekton-internal-workspace",
		VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
	}, {
		Name:         "tekton-internal-home",
		VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
	}, {
		Name:         "tekton-internal-results",
		VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
	}, {
		Name:         "tekton-internal-steps",
		VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
	}}
)

type pipelineTaskContainers struct {
	pt         *v1beta1.PipelineTask
	containers []corev1.Container
	sidecars   []corev1.Container
}

func getPod(ctx context.Context, runMeta metav1.ObjectMeta, cpr *cprv1alpha1.ColocatedPipelineRun, tasks []v1beta1.PipelineTask, images pipeline.Images, entrypointCache EntrypointCache) (*corev1.Pod, map[string]StepInfo, error) {
	activeDeadlineSeconds := int64(60 * 60)
	if cpr.Spec.Timeouts != nil && cpr.Spec.Timeouts.Pipeline != nil {
		activeDeadlineSeconds = int64(cpr.Spec.Timeouts.Pipeline.Seconds() * 2)
	}

	var initContainers []corev1.Container
	var volumes []corev1.Volume
	volumes = append(volumes, implicitVolumes...)
	volumeMounts := []corev1.VolumeMount{binROMount}
	volumeMounts = append(volumeMounts, implicitVolumeMounts...)
	scriptsInit, ptcs := convertScripts(images.ShellImage, "", tasks)
	if scriptsInit != nil {
		initContainers = append(initContainers, *scriptsInit)
		volumes = append(volumes, scriptsVolume)
	}
	imagePullSecrets := []corev1.LocalObjectReference{}
	containerMappings := make(map[string]StepInfo)
	for _, ptc := range ptcs {
		stepContainers, err := resolveEntrypoints(ctx, entrypointCache, cpr.Namespace, cpr.Spec.ServiceAccountName, imagePullSecrets, ptc.containers)
		if err != nil {
			return nil, nil, err
		}
		stepInfo := make(map[string]string, len(stepContainers))

		for i, s := range stepContainers {
			name := names.SimpleNameGenerator.RestrictLength(ContainerName(ptc.pt.Name, s.Name, i))
			stepContainers[i].Name = name
			stepInfo[s.Name] = name
		}
		containerMappings[ptc.pt.Name] = stepInfo
		ptc.containers = stepContainers
	}
	// TODO: working dir?
	entrypointInit := corev1.Container{
		Name:         "place-tools",
		Image:        images.EntrypointImage,
		WorkingDir:   "/",
		Command:      []string{"/ko-app/entrypoint", "cp", "/ko-app/entrypoint", entrypointBinary},
		VolumeMounts: []corev1.VolumeMount{binMount},
	}
	// TODO: this func is supposed to handle task timeouts and onerror
	stepContainers, err := orderContainers(make([]string, 0), ptcs, nil)
	if err != nil {
		return nil, nil, err
	}

	initContainers = append([]corev1.Container{entrypointInit}, initContainers...)
	volumes = append(volumes, binVolume, downwardVolume)
	volumes = append(volumes, getStepContainerVolumes(ctx, stepContainers, volumeMounts)...)
	annotations, err := getAnnotations(cpr)
	if err != nil {
		return nil, nil, err
	}
	// TODO: secrets/creds
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: cpr.Namespace,
			Name:      kmeta.ChildName(cpr.Name, "-pod"),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(&v1alpha1.Run{ObjectMeta: metav1.ObjectMeta{Name: runMeta.Name, UID: types.UID(runMeta.UID)}}, groupVersionKind),
			},
			Annotations: annotations,
			Labels:      getLabels(cpr),
		},
		Spec: corev1.PodSpec{
			RestartPolicy:         corev1.RestartPolicyNever,
			InitContainers:        initContainers,
			Containers:            stepContainers,
			ServiceAccountName:    cpr.Spec.ServiceAccountName,
			ActiveDeadlineSeconds: &activeDeadlineSeconds,
			Volumes:               volumes,
		},
	}
	return pod, containerMappings, nil
}

func getStepContainerVolumes(ctx context.Context, stepContainers []corev1.Container, volumeMounts []corev1.VolumeMount) []corev1.Volume {
	var volumes []corev1.Volume
	for i, s := range stepContainers {
		// TODO (maybe) creds-init

		// Add /tekton/run state volumes.
		// Each step should only mount their own volume as RW,
		// all other steps should be mounted RO.
		volumes = append(volumes, runVolume(i))
		for j := 0; j < len(stepContainers); j++ {
			s.VolumeMounts = append(s.VolumeMounts, runMount(j, i != j))
		}

		requestedVolumeMounts := map[string]bool{}
		for _, vm := range s.VolumeMounts {
			requestedVolumeMounts[filepath.Clean(vm.MountPath)] = true
		}
		var toAdd []corev1.VolumeMount
		for _, imp := range volumeMounts {
			if !requestedVolumeMounts[filepath.Clean(imp.MountPath)] {
				toAdd = append(toAdd, imp)
			}
		}
		vms := append(s.VolumeMounts, toAdd...) //nolint
		stepContainers[i].VolumeMounts = vms
	}
	return volumes
}

func convertScripts(shellImageLinux string, shellImageWin string, tasks []v1beta1.PipelineTask) (*corev1.Container, []pipelineTaskContainers) {
	placeScripts := false

	shellImage := shellImageLinux
	shellCommand := "sh"
	shellArg := "-c"
	requiresWindows := checkWindowsRequirement(tasks)
	ptcs := make([]pipelineTaskContainers, len(tasks)) // task might have multiple containers: problem for later

	// Set windows variants for Image, Command and Args
	if requiresWindows {
		shellImage = shellImageWin
		shellCommand = "pwsh"
		shellArg = "-Command"
	}

	placeScriptsInit := corev1.Container{
		Name:         "place-scripts",
		Image:        shellImage,
		Command:      []string{shellCommand},
		Args:         []string{shellArg, ""},
		VolumeMounts: []corev1.VolumeMount{writeScriptsVolumeMount, binMount},
	}

	for i, pt := range tasks {
		steps := pt.TaskSpec.Steps
		sidecars := pt.TaskSpec.Sidecars
		// Place scripts is an init container used for creating scripts in the
		// /tekton/scripts directory which would be later used by the step containers
		// as a Command

		sideCarSteps := []v1beta1.Step{}
		for _, step := range sidecars {
			sidecarStep := v1beta1.Step{
				Container: step.Container,
				Script:    step.Script,
				Timeout:   &metav1.Duration{},
			}
			sideCarSteps = append(sideCarSteps, sidecarStep)
		}

		convertedStepContainers, initContainerArg := convertListOfSteps(steps, &placeScripts, "script")
		placeScriptsInit.Args[1] += initContainerArg

		// Pass empty breakpoint list in "sidecar step to container" converter to not rewrite the scripts and add breakpoints to sidecar
		sidecarContainers, initContainerArg := convertListOfSteps(sideCarSteps, &placeScripts, "sidecar-script")
		placeScriptsInit.Args[1] += initContainerArg

		foo := pt
		ptcs[i] = pipelineTaskContainers{&foo, convertedStepContainers, sidecarContainers}
	}
	if placeScripts {
		return &placeScriptsInit, ptcs
	}
	return nil, ptcs
}

func convertListOfSteps(steps []v1beta1.Step, placeScripts *bool, namePrefix string) ([]corev1.Container, string) {
	containers := []corev1.Container{}
	initContainerArg := ""
	for i, s := range steps {
		if s.Script == "" {
			// Nothing to convert.
			containers = append(containers, s.Container)
			continue
		}

		// Check for a shebang, and add a default if it's not set.
		// The shebang must be the first non-empty line.
		cleaned := strings.TrimSpace(s.Script)
		hasShebang := strings.HasPrefix(cleaned, "#!")
		requiresWindows := strings.HasPrefix(cleaned, "#!win")

		script := s.Script
		if !hasShebang {
			script = defaultScriptPreamble + s.Script
		}

		// At least one step uses a script, so we should return a
		// non-nil init container.
		*placeScripts = true

		// Append to the place-scripts script to place the
		// script file in a known location in the scripts volume.
		scriptFile := filepath.Join(scriptsDir, names.SimpleNameGenerator.RestrictLengthWithRandomSuffix(fmt.Sprintf("%s-%d", namePrefix, i)))
		if requiresWindows {
			command, args, script, scriptFile := extractWindowsScriptComponents(script, scriptFile)
			initContainerArg += fmt.Sprintf(`@"
%s
"@ | Out-File -FilePath %s
`, script, scriptFile)

			steps[i].Command = command
			// Append existing args field to end of derived args
			args = append(args, steps[i].Args...)
			steps[i].Args = args
		} else {
			// Only encode the script for linux scripts
			// The decode-script subcommand of the entrypoint does not work under windows
			script = encodeScript(script)
			heredoc := "_EOF_" // underscores because base64 doesnt include them in its alphabet
			initContainerArg += fmt.Sprintf(`scriptfile="%s"
touch ${scriptfile} && chmod +x ${scriptfile}
cat > ${scriptfile} << '%s'
%s
%s
/tekton/bin/entrypoint decode-script "${scriptfile}"
`, scriptFile, heredoc, script, heredoc)

			// Set the command to execute the correct script in the mounted
			// volume.
			// A previous merge with stepTemplate may have populated
			// Command and Args, even though this is not normally valid, so
			// we'll clear out the Args and overwrite Command.
			steps[i].Command = []string{scriptFile}
		}
		steps[i].VolumeMounts = append(steps[i].VolumeMounts, scriptsVolumeMount)
		containers = append(containers, steps[i].Container)
	}
	return containers, initContainerArg
}

// encodeScript encodes a script field into a format that avoids kubernetes' built-in processing of container args,
// which can mangle dollar signs and unexpectedly replace variable references in the user's script.
func encodeScript(script string) string {
	return base64.StdEncoding.EncodeToString([]byte(script))
}

func checkWindowsRequirement(tasks []v1beta1.PipelineTask) bool {
	for _, pt := range tasks {
		steps := pt.TaskSpec.Steps
		sidecars := pt.TaskSpec.Sidecars
		// Detect windows shebangs
		for _, step := range steps {
			cleaned := strings.TrimSpace(step.Script)
			if strings.HasPrefix(cleaned, "#!win") {
				return true
			}
		}
		// If no step needs windows, then check sidecars to be sure
		for _, sidecar := range sidecars {
			cleaned := strings.TrimSpace(sidecar.Script)
			if strings.HasPrefix(cleaned, "#!win") {
				return true
			}
		}
	}
	return false
}

func extractWindowsScriptComponents(script string, fileName string) ([]string, []string, string, string) {
	// Set the command to execute the correct script in the mounted volume.
	shebangLine := strings.Split(script, "\n")[0]
	splitLine := strings.Split(shebangLine, " ")
	var command, args []string
	if len(splitLine) > 1 {
		strippedCommand := splitLine[1:]
		command = strippedCommand[0:1]
		// Handle legacy powershell limitation
		if strings.HasPrefix(command[0], "powershell") {
			fileName += ".ps1"
		}
		if len(strippedCommand) > 1 {
			args = strippedCommand[1:]
			args = append(args, fileName)
		} else {
			args = []string{fileName}
		}
	} else {
		// If no interpreter is specified then strip the shebang and
		// create a .cmd file
		fileName += ".cmd"
		commandLines := strings.Split(script, "\n")[1:]
		script = strings.Join(commandLines, "\n")
		command = []string{fileName}
		args = []string{}
	}

	return command, args, script, fileName
}

func runMount(i int, ro bool) corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      fmt.Sprintf("%s-%d", runVolumeName, i),
		MountPath: filepath.Join(runDir, strconv.Itoa(i)),
		ReadOnly:  ro,
	}
}

func runVolume(i int) corev1.Volume {
	return corev1.Volume{
		Name:         fmt.Sprintf("%s-%d", runVolumeName, i),
		VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
	}
}

func getAnnotations(cpr *cprv1alpha1.ColocatedPipelineRun) (map[string]string, error) {
	podAnnotations := kmeta.CopyMap(cpr.Annotations)
	// TODO: Add release annotation
	podAnnotations[readyAnnotation] = readyAnnotationValue
	return podAnnotations, nil
}

func getLabels(cpr *cprv1alpha1.ColocatedPipelineRun) map[string]string {
	labels := make(map[string]string, len(cpr.ObjectMeta.Labels)+1)
	for k, v := range cpr.ObjectMeta.Labels {
		labels[k] = v
	}
	labels[cprv1alpha1.ColocatedPipelineRunLabelKey] = cpr.Name
	labels["app.kubernetes.io/managed-by"] = "tekton-pipelines"
	labels["tekton.dev/memberOf"] = "pipelines"
	return labels
}
