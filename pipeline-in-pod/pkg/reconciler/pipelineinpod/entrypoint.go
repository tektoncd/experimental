package pipelineinpod

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"gomodules.xyz/jsonpatch/v2"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

const (
	binVolumeName    = "tekton-internal-bin"
	binDir           = "/tekton/bin"
	entrypointBinary = binDir + "/entrypoint"

	runVolumeName = "tekton-internal-run"
	runDir        = "/tekton/run"

	downwardVolumeName     = "tekton-internal-downward"
	downwardMountPoint     = "/tekton/downward"
	terminationPath        = "/tekton/termination"
	downwardMountReadyFile = "ready"
	readyAnnotation        = "tekton.dev/ready"
	readyAnnotationValue   = "READY"

	taskPrefix    = "task-"
	stepPrefix    = "step-"
	sidecarPrefix = "sidecar-"

	breakpointOnFailure = "onFailure"
)

var (
	// TODO(#1605): Generate volumeMount names, to avoid collisions.
	binMount = corev1.VolumeMount{
		Name:      binVolumeName,
		MountPath: binDir,
	}
	binROMount = corev1.VolumeMount{
		Name:      binVolumeName,
		MountPath: binDir,
		ReadOnly:  true,
	}
	binVolume = corev1.Volume{
		Name:         binVolumeName,
		VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
	}

	// TODO(#1605): Signal sidecar readiness by injecting entrypoint,
	// remove dependency on Downward API.
	downwardVolume = corev1.Volume{
		Name: downwardVolumeName,
		VolumeSource: corev1.VolumeSource{
			DownwardAPI: &corev1.DownwardAPIVolumeSource{
				Items: []corev1.DownwardAPIVolumeFile{{
					Path: downwardMountReadyFile,
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: fmt.Sprintf("metadata.annotations['%s']", readyAnnotation),
					},
				}},
			},
		},
	}
	downwardMount = corev1.VolumeMount{
		Name:      downwardVolumeName,
		MountPath: downwardMountPoint,
		// Marking this volume mount readonly is technically redundant,
		// since the volume itself is readonly, but including for completeness.
		ReadOnly: true,
	}
)

// createContainers returns the specified steps, modified so that they are
// executed in order by overriding the entrypoint binary.
//
// Containers must have Command specified; if the user didn't specify a
// command, we must have fetched the image's ENTRYPOINT before calling this
// method, using entrypoint_lookup.go.
// Additionally, Step timeouts are added as entrypoint flag.
func createContainers(commonExtraEntrypointArgs []string, ptcs []pipelineTaskContainers, breakpointConfig *v1beta1.TaskRunDebug,
) ([]corev1.Container, []corev1.Volume, error) {
	containers := []corev1.Container{}

	ptNameToPTC := make(map[string]pipelineTaskContainers)
	for _, ptc := range ptcs {
		ptNameToPTC[ptc.pt.Name] = ptc
	}
	var volumes []corev1.Volume

	for _, ptc := range ptcs {
		steps := ptc.containers
		taskSpec := ptc.pt.TaskSpec
		var runAfter []pipelineTaskContainers
		for _, ra := range ptc.pt.RunAfter {
			runAfter = append(runAfter, ptNameToPTC[ra])
		}
		if len(steps) == 0 {
			return nil, nil, errors.New("no steps specified")
		}

		for i, s := range steps {
			var argsForEntrypoint []string
			var waitFiles string
			var waitFileContents bool

			if i == 0 {
				if len(runAfter) == 0 {
					// Wait for "ready" file
					waitFiles = filepath.Join(downwardMountPoint, downwardMountReadyFile)
					waitFileContents = true
					steps[i].VolumeMounts = append(steps[0].VolumeMounts, downwardMount)
				} else {
					// Wait for the last step in any previous tasks
					for j, previous := range runAfter {
						if j != 0 {
							waitFiles = waitFiles + ","
						}
						lastStepInPrevious := len(previous.containers) - 1
						waitFiles = waitFiles + filepath.Join(runDir, previous.pt.Name, strconv.Itoa(lastStepInPrevious), "out")
					}
				}
			} else {
				// wait for previous step in current task
				waitFiles = filepath.Join(runDir, ptc.pt.Name, strconv.Itoa(i-1), "out")
			}
			argsForEntrypoint = []string{
				"-wait_file", waitFiles,
				"-post_file", filepath.Join(runDir, ptc.pt.Name, strconv.Itoa(i), "out"),
				"-termination_path", terminationPath,
				"-step_metadata_dir", filepath.Join(pipeline.StepsDir, steps[i].Name),
			}
			if waitFileContents {
				argsForEntrypoint = append(argsForEntrypoint, "-wait_file_content")
			}

			argsForEntrypoint = append(argsForEntrypoint, commonExtraEntrypointArgs...)
			if taskSpec != nil {
				if taskSpec.Steps != nil && len(taskSpec.Steps) >= i+1 {
					if taskSpec.Steps[i].Timeout != nil {
						argsForEntrypoint = append(argsForEntrypoint, "-timeout", taskSpec.Steps[i].Timeout.Duration.String())
					}
					if taskSpec.Steps[i].OnError != "" {
						argsForEntrypoint = append(argsForEntrypoint, "-on_error", taskSpec.Steps[i].OnError)
					}
				}
				argsForEntrypoint = append(argsForEntrypoint, resultArgument(steps, taskSpec.Results)...)
			}

			if breakpointConfig != nil && len(breakpointConfig.Breakpoint) > 0 {
				breakpoints := breakpointConfig.Breakpoint
				for _, b := range breakpoints {
					// TODO(TEP #0042): Add other breakpoints
					if b == breakpointOnFailure {
						argsForEntrypoint = append(argsForEntrypoint, "-breakpoint_on_failure")
					}
				}
			}

			cmd, args := s.Command, s.Args
			if len(cmd) > 0 {
				argsForEntrypoint = append(argsForEntrypoint, "-entrypoint", cmd[0])
			}
			if len(cmd) > 1 {
				args = append(cmd[1:], args...)
			}
			argsForEntrypoint = append(argsForEntrypoint, "--")

			argsForEntrypoint = append(argsForEntrypoint, args...)

			steps[i].Command = []string{entrypointBinary}
			steps[i].Args = argsForEntrypoint
			steps[i].TerminationMessagePath = terminationPath

			v, vms := getVolumesForStep(ptc, i, runAfter)
			steps[i].VolumeMounts = vms

			volumes = append(volumes, v...)
		}
		containers = append(containers, steps...)
		//v := mountVolumesForTask(&ptc, ptNameToPTC)
	}
	return containers, volumes, nil
}

func getVolumesForStep(ptc pipelineTaskContainers, i int, runAfter []pipelineTaskContainers) ([]corev1.Volume, []corev1.VolumeMount) {
	var volumes []corev1.Volume
	s := ptc.containers[i]

	// TODO (maybe) creds-init

	// Add /tekton/run state volumes.
	// Each step should only mount their own volume as RW,
	// all other steps should be mounted RO.
	volumeMounts := s.VolumeMounts
	volumes = append(volumes, runVolume(ptc.pt.Name, i))
	for j := 0; j < len(ptc.containers); j++ {
		volumeMounts = append(volumeMounts, runMount(ptc.pt.Name, j, i != j))
	}

	// Add /tekton/run volumes for steps this step must wait for
	for _, ra := range runAfter {
		lastStepInPrevious := len(ra.containers) - 1
		volumeMounts = append(volumeMounts, runMount(ra.pt.Name, lastStepInPrevious, true))
	}
	return volumes, volumeMounts
}

func runMount(ptName string, i int, ro bool) corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      fmt.Sprintf("%s-%s-%d", runVolumeName, ptName, i),
		MountPath: filepath.Join(runDir, ptName, strconv.Itoa(i)),
		ReadOnly:  ro,
	}
}

func runVolume(ptName string, i int) corev1.Volume {
	return corev1.Volume{
		Name:         fmt.Sprintf("%s-%s-%d", runVolumeName, ptName, i),
		VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
	}
}

func resultArgument(steps []corev1.Container, results []v1beta1.TaskResult) []string {
	if len(results) == 0 {
		return nil
	}
	return []string{"-results", collectResultsName(results)}
}

func collectResultsName(results []v1beta1.TaskResult) string {
	var resultNames []string
	for _, r := range results {
		resultNames = append(resultNames, r.Name)
	}
	return strings.Join(resultNames, ",")
}

var replaceReadyPatchBytes []byte

func init() {
	// https://stackoverflow.com/questions/55573724/create-a-patch-to-add-a-kubernetes-annotation
	readyAnnotationPath := "/metadata/annotations/" + strings.Replace(readyAnnotation, "/", "~1", 1)
	var err error
	replaceReadyPatchBytes, err = json.Marshal([]jsonpatch.JsonPatchOperation{{
		Operation: "replace",
		Path:      readyAnnotationPath,
		Value:     readyAnnotationValue,
	}})
	if err != nil {
		log.Fatalf("failed to marshal replace ready patch bytes: %v", err)
	}
}

// UpdateReady updates the Pod's annotations to signal the first step to start
// by projecting the ready annotation via the Downward API.
func UpdateReady(ctx context.Context, kubeclient kubernetes.Interface, pod corev1.Pod) error {
	// PATCH the Pod's annotations to replace the ready annotation with the
	// "READY" value, to signal the first step to start.
	_, err := kubeclient.CoreV1().Pods(pod.Namespace).Patch(ctx, pod.Name, types.JSONPatchType, replaceReadyPatchBytes, metav1.PatchOptions{})
	return err
}

// StopSidecars updates sidecar containers in the Pod to a nop image, which
// exits successfully immediately.
func StopSidecars(ctx context.Context, nopImage string, kubeclient kubernetes.Interface, namespace, name string) (*corev1.Pod, error) {
	newPod, err := kubeclient.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if k8serrors.IsNotFound(err) {
		// return NotFound as-is, since the K8s error checks don't handle wrapping.
		return nil, err
	} else if err != nil {
		return nil, fmt.Errorf("error getting Pod %q when stopping sidecars: %w", name, err)
	}

	updated := false
	if newPod.Status.Phase == corev1.PodRunning {
		for _, s := range newPod.Status.ContainerStatuses {
			// Stop any running container that isn't a step.
			// An injected sidecar container might not have the
			// "sidecar-" prefix, so we can't just look for that
			// prefix.
			if !IsContainerStep(s.Name) && s.State.Running != nil {
				for j, c := range newPod.Spec.Containers {
					if c.Name == s.Name && c.Image != nopImage {
						updated = true
						newPod.Spec.Containers[j].Image = nopImage
					}
				}
			}
		}
	}
	if updated {
		if newPod, err = kubeclient.CoreV1().Pods(newPod.Namespace).Update(ctx, newPod, metav1.UpdateOptions{}); err != nil {
			return nil, fmt.Errorf("error stopping sidecars of Pod %q: %w", name, err)
		}
	}
	return newPod, nil
}

// IsSidecarStatusRunning determines if any SidecarStatus on a TaskRun
// is still running.
func IsSidecarStatusRunning(tr *v1beta1.TaskRun) bool {
	for _, sidecar := range tr.Status.Sidecars {
		if sidecar.Terminated == nil {
			return true
		}
	}

	return false
}

// IsContainerStep returns true if the container name indicates that it
// represents a step.
func IsContainerStep(name string) bool { return strings.Contains(name, stepPrefix) }

func trimTaskPrefix(name string) string { return strings.TrimPrefix(name, taskPrefix) }

// TrimSidecarPrefix returns the container name, stripped of its sidecar
// prefix.
func TrimSidecarPrefix(name string) string { return strings.TrimPrefix(name, sidecarPrefix) }

func ContainerName(taskName, stepName string, i int) string {
	foo := StepName(stepName, i)
	return "task-" + taskName + "-" + foo
}

// StepName returns the step name after adding "step-" prefix to the actual step name or
// returns "step-unnamed-<step-index>" if not specified
func StepName(name string, i int) string {
	if name != "" {
		return fmt.Sprintf("%s%s", stepPrefix, name)
	}
	return fmt.Sprintf("%sunnamed-%d", stepPrefix, i)
}

func TaskAndStepNameFromContainerName(name string) (string, string, error) {
	foo := trimTaskPrefix(name)
	bar := strings.Split(foo, stepPrefix)
	taskName := strings.TrimSuffix(bar[0], "-")
	stepName := strings.TrimPrefix(bar[1], "-")
	return taskName, stepName, nil
	// TODO: handle unnamed steps
}
