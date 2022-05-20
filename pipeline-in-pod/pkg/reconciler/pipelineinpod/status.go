package pipelineinpod

import (
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/go-multierror"
	cprv1alpha1 "github.com/tektoncd/experimental/pipeline-in-pod/pkg/apis/colocatedpipelinerun/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/termination"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
)

const (
	// ReasonExceededNodeResources indicates that the TaskRun's pod has failed to start due
	// to resource constraints on the node
	ReasonExceededNodeResources = "ExceededNodeResources"

	// ReasonCreateContainerConfigError indicates that the TaskRun failed to create a pod due to
	// config error of container
	ReasonCreateContainerConfigError = "CreateContainerConfigError"

	// ReasonPending indicates that the pod is in corev1.Pending, and the reason is not
	// ReasonExceededNodeResources or isPodHitConfigError
	ReasonPending = "Pending"
)

const oomKilled = "OOMKilled"

func updateContainerStatuses(logger *zap.SugaredLogger, cpr *cprv1alpha1.ColocatedPipelineRunStatus, pod *corev1.Pod) error {
	containerNameToStatus := make(map[string]corev1.ContainerStatus, len(pod.Status.ContainerStatuses))
	for _, container := range pod.Status.ContainerStatuses {
		containerNameToStatus[container.Name] = container
	}
	for i, task := range cpr.ChildStatuses {
		for j, step := range task.StepStatuses {
			containerStatus, ok := containerNameToStatus[step.ContainerName]
			if !ok {
				logger.Infof("No container status found for step %s", step.Name)
				continue
			}
			cpr.ChildStatuses[i].StepStatuses[j].ContainerState = containerStatus.State
		}
	}
	return nil
}

// sorts pod containers into the respective tasks
// creates taskrun status by calling maketaskrunstatus
// applies taskrun statuses to the pipelinerun
func MakeColocatedPipelineRunStatus(logger *zap.SugaredLogger, cpr cprv1alpha1.ColocatedPipelineRun, pod *corev1.Pod) (cprv1alpha1.ColocatedPipelineRunStatus, error) {
	logger.Infof("pod status: %s", pod.Status.Phase)
	cprs := &cpr.Status
	if cprs.GetCondition(apis.ConditionSucceeded) == nil || cprs.GetCondition(apis.ConditionSucceeded).Status == corev1.ConditionUnknown {
		markStatusRunning(cprs, v1beta1.PipelineRunReasonRunning.String(), "Not all Tasks in the Pipeline have finished executing")
	}
	sortPodContainerStatuses(pod.Status.ContainerStatuses, pod.Spec.Containers)
	if pod.Status.Phase == corev1.PodSucceeded {
		markStatusSuccess(cprs)
	} else if pod.Status.Phase == corev1.PodFailed {
		failureMsg := getFailureMessage(logger, pod)
		markStatusFailure(cprs, "Reason TODO", failureMsg)
	} else if pod.Status.Phase == corev1.PodRunning {
		markStatusRunning(cprs, "Pod pending", "pod pending")
	} else if pod.Status.Phase == corev1.PodPending {
		switch {
		case IsPodExceedingNodeResources(pod):
			markStatusRunning(cprs, ReasonExceededNodeResources, "TaskRun Pod exceeded available resources")
		case isPodHitConfigError(pod):
			markStatusFailure(cprs, ReasonCreateContainerConfigError, "Failed to create pod due to config error")
		default:
			msg := getWaitingMessage(pod)
			markStatusRunning(cprs, ReasonPending, msg)
			logger.Infof("pod message: %s", msg)
		}
	}

	pending := false
	succeeded := true
	for i, task := range cprs.ChildStatuses {
		taskStatus, taskResults, err := getTaskStatusBasedOnStepStatus(logger, task.StepStatuses)
		if err != nil || taskStatus.Type != apis.ConditionSucceeded {
			return *cprs, fmt.Errorf("error parsing task status %s", taskStatus.Type)
		}
		if taskStatus.Status == corev1.ConditionUnknown {
			logger.Infof("task %s pending: message %s", task.PipelineTaskName, taskStatus.Message)
			pending = true
		} else if taskStatus.Status == corev1.ConditionFalse {
			logger.Infof("task %s failed: message %s", task.PipelineTaskName, taskStatus.Message)
			succeeded = false
		} else {
			logger.Infof("task %s succeeded", task.PipelineTaskName)
		}
		cprs.ChildStatuses[i].TaskRunResults = taskResults
	}

	if pending {
		logger.Infof("Not all Tasks in CPR %s have completed", cpr.Name)
		markStatusRunning(cprs, "Pending", "Pending")
	} else if succeeded {
		logger.Infof("All Tasks in CPR %s have completed", cpr.Name)
		markStatusSuccess(cprs)
		markCompletionTime(cprs)
	} else {
		msg := fmt.Sprintf("At least one Task in CPR %s has failed", cpr.Name)
		logger.Infof(msg)
		markStatusFailure(cprs, "Failed", msg)
		markCompletionTime(cprs)
	}

	return *cprs, nil
}

func getTaskStatusBasedOnStepStatus(logger *zap.SugaredLogger, stepStatuses []v1beta1.StepState) (apis.Condition, []v1beta1.TaskRunResult, *multierror.Error) {
	var merr *multierror.Error
	var pendingSteps []string
	var failedSteps []string
	outputResults := make([]v1beta1.TaskRunResult, 0)
	for _, s := range stepStatuses {
		if !isComplete(s) {
			pendingSteps = append(pendingSteps, s.Name)
		}
		if isFailure(s) {
			failedSteps = append(failedSteps, s.Name)
		}
		if s.Terminated != nil && len(s.Terminated.Message) != 0 {
			msg := s.Terminated.Message
			results, err := termination.ParseMessage(logger, msg)
			if err != nil {
				logger.Errorf("termination message could not be parsed as JSON: %v", err)
				merr = multierror.Append(merr, err)
			} else {
				taskResults := filterResultsAndResources(results)
				outputResults = append(outputResults, taskResults...)
			}
		}
	}
	var status corev1.ConditionStatus
	var msg string
	if len(pendingSteps) > 0 {
		status = corev1.ConditionUnknown
		msg = fmt.Sprintf("The following steps are pending: %s", pendingSteps)
	} else if len(failedSteps) > 0 {
		status = corev1.ConditionFalse
		msg = fmt.Sprintf("The following steps failed: %s", failedSteps)
	} else {
		status = corev1.ConditionTrue
		msg = "All steps succeeded"
	}

	return apis.Condition{Type: apis.ConditionSucceeded, Status: status, Message: msg}, outputResults, merr
}

func filterResultsAndResources(results []v1beta1.PipelineResourceResult) []v1beta1.TaskRunResult {
	var taskResults []v1beta1.TaskRunResult
	for _, r := range results {
		if r.ResultType == v1beta1.TaskRunResultType && r.Key != "" && r.Value != "" {
			taskResults = append(taskResults, v1beta1.TaskRunResult{
				Name:  r.Key,
				Value: r.Value,
			})
		}
	}

	return taskResults
}

func isComplete(ss v1beta1.StepState) bool {
	return ss.Terminated != nil
}

// returns true if terminated and failed
func isFailure(ss v1beta1.StepState) bool {
	if !isComplete(ss) {
		return false
	}
	return ss.Terminated.ExitCode != 0 || isOOMKilled(ss.ContainerState)
}

func getFailureMessage(logger *zap.SugaredLogger, pod *corev1.Pod) string {
	// First, try to surface an error about the actual build step that failed.
	for _, status := range pod.Status.ContainerStatuses {
		term := status.State.Terminated
		if term != nil {
			msg := status.State.Terminated.Message
			r, _ := termination.ParseMessage(logger, msg)
			for _, result := range r {
				if result.ResultType == v1beta1.InternalTektonResultType && result.Key == "Reason" && result.Value == "TimeoutExceeded" {
					// Newline required at end to prevent yaml parser from breaking the log help text at 80 chars
					return fmt.Sprintf("%q exited because the step exceeded the specified timeout limit; for logs run: kubectl -n %s logs %s -c %s\n",
						status.Name,
						pod.Namespace, pod.Name, status.Name)
				}
			}
			if term.ExitCode != 0 {
				// Newline required at end to prevent yaml parser from breaking the log help text at 80 chars
				return fmt.Sprintf("%q exited with code %d (image: %q); for logs run: kubectl -n %s logs %s -c %s\n",
					status.Name, term.ExitCode, status.ImageID,
					pod.Namespace, pod.Name, status.Name)
			}
		}
	}
	// Next, return the Pod's status message if it has one.
	if pod.Status.Message != "" {
		return pod.Status.Message
	}

	for _, s := range pod.Status.ContainerStatuses {
		if IsContainerStep(s.Name) {
			if s.State.Terminated != nil {
				if isOOMKilled(s.State) {
					return oomKilled
				}
			}
		}
	}

	// Lastly fall back on a generic error message.
	return "build failed for unspecified reasons."
}

// IsPodExceedingNodeResources returns true if the Pod's status indicates there
// are insufficient resources to schedule the Pod.
func IsPodExceedingNodeResources(pod *corev1.Pod) bool {
	for _, podStatus := range pod.Status.Conditions {
		if podStatus.Reason == corev1.PodReasonUnschedulable && strings.Contains(podStatus.Message, "Insufficient") {
			return true
		}
	}
	return false
}

// isPodHitConfigError returns true if the Pod's status undicates there are config error raised
func isPodHitConfigError(pod *corev1.Pod) bool {
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if containerStatus.State.Waiting != nil && containerStatus.State.Waiting.Reason == ReasonCreateContainerConfigError {
			return true
		}
	}
	return false
}

func getWaitingMessage(pod *corev1.Pod) string {
	// First, try to surface reason for pending/unknown about the actual build step.
	for _, status := range pod.Status.ContainerStatuses {
		wait := status.State.Waiting
		if wait != nil && wait.Message != "" {
			return fmt.Sprintf("build step %q is pending with reason %q",
				status.Name, wait.Message)
		}
	}
	// Try to surface underlying reason by inspecting pod's recent status if condition is not true
	for i, podStatus := range pod.Status.Conditions {
		if podStatus.Status != corev1.ConditionTrue {
			return fmt.Sprintf("pod status %q:%q; message: %q",
				pod.Status.Conditions[i].Type,
				pod.Status.Conditions[i].Status,
				pod.Status.Conditions[i].Message)
		}
	}
	// Next, return the Pod's status message if it has one.
	if pod.Status.Message != "" {
		return pod.Status.Message
	}

	// Lastly fall back on a generic pending message.
	return "Pending"
}

// markStatusRunning sets cpr status to running
func markStatusRunning(cprs *cprv1alpha1.ColocatedPipelineRunStatus, reason, message string) {
	cprs.SetCondition(&apis.Condition{
		Type:    apis.ConditionSucceeded,
		Status:  corev1.ConditionUnknown,
		Reason:  reason,
		Message: message,
	})
}

// markStatusFailure sets cpr status to failure with specified reason
func markStatusFailure(cprs *cprv1alpha1.ColocatedPipelineRunStatus, reason string, message string) {
	cprs.SetCondition(&apis.Condition{
		Type:    apis.ConditionSucceeded,
		Status:  corev1.ConditionFalse,
		Reason:  reason,
		Message: message,
	})
}

// markStatusSuccess sets cpr status to success
func markStatusSuccess(cprs *cprv1alpha1.ColocatedPipelineRunStatus) {
	cprs.SetCondition(&apis.Condition{
		Type:    apis.ConditionSucceeded,
		Status:  corev1.ConditionTrue,
		Reason:  v1beta1.TaskRunReasonSuccessful.String(),
		Message: "All Steps have completed executing",
	})
}

func markCompletionTime(cprs *cprv1alpha1.ColocatedPipelineRunStatus) {
	if cprs.CompletionTime == nil {
		cprs.CompletionTime = &metav1.Time{Time: time.Now()}
	}
}

// sortPodContainerStatuses reorders a pod's container statuses so that
// they're in the same order as the step containers from the TaskSpec.
func sortPodContainerStatuses(podContainerStatuses []corev1.ContainerStatus, podSpecContainers []corev1.Container) {
	statuses := map[string]corev1.ContainerStatus{}
	for _, status := range podContainerStatuses {
		statuses[status.Name] = status
	}
	for i, c := range podSpecContainers {
		// prevent out-of-bounds panic on incorrectly formed lists
		if i < len(podContainerStatuses) {
			podContainerStatuses[i] = statuses[c.Name]
		}
	}
}

func isOOMKilled(s corev1.ContainerState) bool {
	return s.Terminated.Reason == oomKilled
}
