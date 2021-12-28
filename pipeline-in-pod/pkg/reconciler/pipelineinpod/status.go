package pipelineinpod

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/termination"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
)

const (
	// ReasonCouldntGetTask indicates that the reason for the failure status is that the
	// Task couldn't be found
	ReasonCouldntGetTask = "CouldntGetTask"

	// ReasonFailedResolution indicated that the reason for failure status is
	// that references within the TaskRun could not be resolved
	ReasonFailedResolution = "TaskRunResolutionFailed"

	// ReasonFailedValidation indicated that the reason for failure status is
	// that taskrun failed runtime validation
	ReasonFailedValidation = "TaskRunValidationFailed"

	// ReasonExceededResourceQuota indicates that the TaskRun failed to create a pod due to
	// a ResourceQuota in the namespace
	ReasonExceededResourceQuota = "ExceededResourceQuota"

	// ReasonExceededNodeResources indicates that the TaskRun's pod has failed to start due
	// to resource constraints on the node
	ReasonExceededNodeResources = "ExceededNodeResources"

	// ReasonCreateContainerConfigError indicates that the TaskRun failed to create a pod due to
	// config error of container
	ReasonCreateContainerConfigError = "CreateContainerConfigError"

	// ReasonPodCreationFailed indicates that the reason for the current condition
	// is that the creation of the pod backing the TaskRun failed
	ReasonPodCreationFailed = "PodCreationFailed"

	// ReasonPending indicates that the pod is in corev1.Pending, and the reason is not
	// ReasonExceededNodeResources or isPodHitConfigError
	ReasonPending = "Pending"

	// timeFormat is RFC3339 with millisecond
	timeFormat = "2006-01-02T15:04:05.000Z07:00"
)

const oomKilled = "OOMKilled"

type taskRunStatusInfo struct {
	pipelineTaskName string
	taskRunName      string
	stepStatusInfos  []stepStatusInfo
}

type stepStatusInfo struct {
	stepName        string
	container       *corev1.Container
	containerStatus *corev1.ContainerStatus
}

// SidecarsReady returns true if all of the Pod's sidecars are Ready or
// Terminated.
func SidecarsReady(podStatus corev1.PodStatus) bool {
	if podStatus.Phase != corev1.PodRunning {
		return false
	}
	for _, s := range podStatus.ContainerStatuses {
		// If the step indicates that it's a step, skip it.
		// An injected sidecar might not have the "sidecar-" prefix, so
		// we can't just look for that prefix, we need to look at any
		// non-step container.
		if IsContainerStep(s.Name) {
			continue
		}
		if s.State.Running != nil && s.Ready {
			continue
		}
		if s.State.Terminated != nil {
			continue
		}
		return false
	}
	return true
}

// sorts pod containers into the respective tasks
// creates taskrun status by calling maketaskrunstatus
// applies taskrun statuses to the pipelinerun
func MakePipelineRunStatus(logger *zap.SugaredLogger, pr v1beta1.PipelineRun, pod *corev1.Pod) (v1beta1.PipelineRunStatus, error) {
	//complete := areStepsComplete(pod) || pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed
	// Get containers associated with TaskRun
	// Get TaskRun status from containers
	// update PR status with TR statuses
	logger.Infof("pod status: %s", pod.Status.Phase)
	prs := &pr.Status
	if prs.GetCondition(apis.ConditionSucceeded) == nil || prs.GetCondition(apis.ConditionSucceeded).Status == corev1.ConditionUnknown {
		markPRStatusRunning(prs, v1beta1.PipelineRunReasonRunning.String(), "Not all Tasks in the Pipeline have finished executing")
	}
	sortPodContainerStatuses(pod.Status.ContainerStatuses, pod.Spec.Containers)
	if pod.Status.Phase == corev1.PodSucceeded {
		markPRStatusSuccess(prs)
	} else if pod.Status.Phase == corev1.PodFailed {
		failureMsg := getFailureMessage(logger, pod)
		markPRStatusFailure(prs, "Reason TODO", failureMsg)
	} else if pod.Status.Phase == corev1.PodRunning {
		markPRStatusRunning(prs, "Pod pending", "pod pending")
	} else if pod.Status.Phase == corev1.PodPending {
		switch {
		case IsPodExceedingNodeResources(pod):
			markPRStatusRunning(prs, ReasonExceededNodeResources, "TaskRun Pod exceeded available resources")
		case isPodHitConfigError(pod):
			markPRStatusFailure(prs, ReasonCreateContainerConfigError, "Failed to create pod due to config error")
		default:
			msg := getWaitingMessage(pod)
			markPRStatusRunning(prs, ReasonPending, msg)
			logger.Infof("pod message: %s", msg)
		}
	}

	taskNameToTaskStatusInfo, err := getPtContainers(logger, pod)
	if err != nil {
		logger.Errorf("error getting task name from containers for PR %s and pod %s: %s", pr.Name, pod.Name, err)
		return *prs, err
	}
	if len(taskNameToTaskStatusInfo) == 0 {
		logger.Errorf("no tasks!")
	}
	for taskName, statusInfo := range taskNameToTaskStatusInfo {
		logger.Infof("task %s has %d containers", taskName, len(statusInfo.stepStatusInfos))
	}
	pending := false
	succeeded := true
	newTaskRunStatuses := make(map[string]*v1beta1.PipelineRunTaskRunStatus, len(prs.TaskRuns))

	for trName, taskStatusInfo := range taskNameToTaskStatusInfo {
		logger.Infof("processing tr status for task %s; num containers %d", trName, len(taskStatusInfo.stepStatusInfos))

		trs, ok := pr.Status.TaskRuns[trName]
		if !ok {
			trs = &v1beta1.PipelineRunTaskRunStatus{Status: &v1beta1.TaskRunStatus{}}
			logger.Infof("Creating new TR status for task %s", trName)
		}
		newTrs, err := MakeTaskRunStatusFromContainers(logger, *trs.Status, taskStatusInfo.stepStatusInfos, pod.Name, trName)
		if err != nil {
			logger.Errorf("error creating taskrun statuses from containers for PR %s and pod %s: %s", pr.Name, pod.Name, err)
			return *prs, err
		}
		if !IsDone(&newTrs) {
			pending = true
		} else {
			if !IsSuccessful(&newTrs) {
				succeeded = false
			}
		}
		newPrtrs := pipelineTaskStatusFromTaskRun(&newTrs)
		newTaskRunStatuses[trName] = &newPrtrs
	}

	prs.TaskRuns = newTaskRunStatuses
	if pending {
		logger.Infof("Not all TaskRuns in PR %s have completed", pr.Name)
		markPRStatusRunning(prs, "Pending", "Pending")
	} else if succeeded {
		logger.Infof("All TaskRuns in PR %s have completed", pr.Name)
		markPRStatusSuccess(prs)
	} else {
		logger.Infof("At least one TaskRun in PR %s has failed", pr.Name)
		markPRStatusFailure(prs, "failure", "failure")
	}

	return *prs, nil
}

func MakeTaskRunStatusFromContainers(logger *zap.SugaredLogger, trs v1beta1.TaskRunStatus, stepStatusInfos []stepStatusInfo, podName, trName string) (v1beta1.TaskRunStatus, error) {
	var containerStatuses []corev1.ContainerStatus
	allStatusesPresent := true
	for _, ss := range stepStatusInfos {
		if ss.containerStatus == nil {
			allStatusesPresent = false
			logger.Infof("missing container status for TR %s", trName)
		} else {
			containerStatuses = append(containerStatuses, *ss.containerStatus)
		}
	}
	complete := areStepsComplete(containerStatuses)
	logger.Infof("complete: %t, allStatusesPresent: %t", complete, allStatusesPresent)
	if complete && allStatusesPresent {
		updateCompletedTaskRunStatusFromContainers(logger, &trs, containerStatuses)
	} else {
		//updateIncompleteTaskRunStatus(trs, pod)
		markStatusRunning(&trs, ReasonPending, "pending")
	}

	trs.PodName = podName
	trs.Steps = []v1beta1.StepState{}
	trs.Sidecars = []v1beta1.SidecarState{}

	var stepStatuses []corev1.ContainerStatus
	var sidecarStatuses []corev1.ContainerStatus
	for _, s := range containerStatuses {
		if IsContainerStep(s.Name) {
			stepStatuses = append(stepStatuses, s)
		} else if isContainerSidecar(s.Name) {
			sidecarStatuses = append(sidecarStatuses, s)
		}
	}

	var merr *multierror.Error
	if err := setTaskRunStatusBasedOnStepStatus(logger, stepStatuses, &trs, trName); err != nil {
		merr = multierror.Append(merr, err)
	}

	//setTaskRunStatusBasedOnSidecarStatus(sidecarStatuses, trs)

	trs.TaskRunResults = removeDuplicateResults(trs.TaskRunResults)

	return trs, merr.ErrorOrNil()
}

func setTaskRunStatusBasedOnStepStatus(logger *zap.SugaredLogger, stepStatuses []corev1.ContainerStatus, trs *v1beta1.TaskRunStatus, trName string) *multierror.Error {
	var merr *multierror.Error

	for _, s := range stepStatuses {
		if s.State.Terminated != nil && len(s.State.Terminated.Message) != 0 {
			msg := s.State.Terminated.Message

			results, err := termination.ParseMessage(logger, msg)
			if err != nil {
				logger.Errorf("termination message could not be parsed as JSON: %v", err)
				merr = multierror.Append(merr, err)
			} else {
				time, err := extractStartedAtTimeFromResults(results)
				if err != nil {
					logger.Errorf("error setting the start time of step %q in taskrun %q: %v", s.Name, trName, err)
					merr = multierror.Append(merr, err)
				}
				exitCode, err := extractExitCodeFromResults(results)
				if err != nil {
					logger.Errorf("error extracting the exit code of step %q in taskrun %q: %v", s.Name, trName, err)
					merr = multierror.Append(merr, err)
				}
				taskResults, pipelineResourceResults, filteredResults := filterResultsAndResources(results)
				if IsSuccessful(trs) {
					trs.TaskRunResults = append(trs.TaskRunResults, taskResults...)
					trs.ResourcesResult = append(trs.ResourcesResult, pipelineResourceResults...)
				}
				msg, err = createMessageFromResults(filteredResults)
				if err != nil {
					logger.Errorf("%v", err)
					err = multierror.Append(merr, err)
				} else {
					s.State.Terminated.Message = msg
				}
				if time != nil {
					s.State.Terminated.StartedAt = *time
				}
				if exitCode != nil {
					s.State.Terminated.ExitCode = *exitCode
				}
			}
		}
		trs.Steps = append(trs.Steps, v1beta1.StepState{
			ContainerState: *s.State.DeepCopy(),
			Name:           trimStepPrefix(s.Name),
			ContainerName:  s.Name,
			ImageID:        s.ImageID,
		})
	}

	return merr

}

func IsSuccessful(trs *v1beta1.TaskRunStatus) bool {
	return trs.GetCondition(apis.ConditionSucceeded).IsTrue()
}
func IsDone(trs *v1beta1.TaskRunStatus) bool {
	return !trs.GetCondition(apis.ConditionSucceeded).IsUnknown()
}

func pipelineTaskStatusFromTaskRun(trs *v1beta1.TaskRunStatus) v1beta1.PipelineRunTaskRunStatus {
	return v1beta1.PipelineRunTaskRunStatus{Status: trs}
}

func setTaskRunStatusBasedOnSidecarStatus(sidecarStatuses []corev1.ContainerStatus, trs *v1beta1.TaskRunStatus) {
	for _, s := range sidecarStatuses {
		trs.Sidecars = append(trs.Sidecars, v1beta1.SidecarState{
			ContainerState: *s.State.DeepCopy(),
			Name:           TrimSidecarPrefix(s.Name),
			ContainerName:  s.Name,
			ImageID:        s.ImageID,
		})
	}
}

func createMessageFromResults(results []v1beta1.PipelineResourceResult) (string, error) {
	if len(results) == 0 {
		return "", nil
	}
	bytes, err := json.Marshal(results)
	if err != nil {
		return "", fmt.Errorf("error marshalling remaining results back into termination message: %w", err)
	}
	return string(bytes), nil
}

func filterResultsAndResources(results []v1beta1.PipelineResourceResult) ([]v1beta1.TaskRunResult, []v1beta1.PipelineResourceResult, []v1beta1.PipelineResourceResult) {
	var taskResults []v1beta1.TaskRunResult
	var pipelineResourceResults []v1beta1.PipelineResourceResult
	var filteredResults []v1beta1.PipelineResourceResult
	for _, r := range results {
		switch r.ResultType {
		case v1beta1.TaskRunResultType:
			taskRunResult := v1beta1.TaskRunResult{
				Name:  r.Key,
				Value: r.Value,
			}
			taskResults = append(taskResults, taskRunResult)
			filteredResults = append(filteredResults, r)
		case v1beta1.InternalTektonResultType:
			// Internal messages are ignored because they're not used as external result
			continue
		case v1beta1.PipelineResourceResultType:
			fallthrough
		default:
			pipelineResourceResults = append(pipelineResourceResults, r)
			filteredResults = append(filteredResults, r)
		}
	}

	return taskResults, pipelineResourceResults, filteredResults
}

func removeDuplicateResults(taskRunResult []v1beta1.TaskRunResult) []v1beta1.TaskRunResult {
	if len(taskRunResult) == 0 {
		return nil
	}

	uniq := make([]v1beta1.TaskRunResult, 0)
	latest := make(map[string]v1beta1.TaskRunResult, 0)
	for _, res := range taskRunResult {
		if _, seen := latest[res.Name]; !seen {
			uniq = append(uniq, res)
		}
		latest[res.Name] = res
	}
	for i, res := range uniq {
		uniq[i] = latest[res.Name]
	}
	return uniq
}

func extractStartedAtTimeFromResults(results []v1beta1.PipelineResourceResult) (*metav1.Time, error) {
	for _, result := range results {
		if result.Key == "StartedAt" {
			t, err := time.Parse(timeFormat, result.Value)
			if err != nil {
				return nil, fmt.Errorf("could not parse time value %q in StartedAt field: %w", result.Value, err)
			}
			startedAt := metav1.NewTime(t)
			return &startedAt, nil
		}
	}
	return nil, nil
}

func extractExitCodeFromResults(results []v1beta1.PipelineResourceResult) (*int32, error) {
	for _, result := range results {
		if result.Key == "ExitCode" {
			// We could just pass the string through but this provides extra validation
			i, err := strconv.ParseUint(result.Value, 10, 32)
			if err != nil {
				return nil, fmt.Errorf("could not parse int value %q in ExitCode field: %w", result.Value, err)
			}
			exitCode := int32(i)
			return &exitCode, nil
		}
	}
	return nil, nil
}

func updateCompletedTaskRunStatusFromContainers(logger *zap.SugaredLogger, trs *v1beta1.TaskRunStatus, containerStatuses []corev1.ContainerStatus) {
	if DidPipelineRunFail(containerStatuses) {
		msg := "failed"
		markStatusFailure(trs, v1beta1.TaskRunReasonFailed.String(), msg)
	} else {
		markStatusSuccess(trs)
	}

	// update tr completed time
	trs.CompletionTime = &metav1.Time{Time: time.Now()}
}

func updateIncompleteTaskRunStatus(trs *v1beta1.TaskRunStatus, pod *corev1.Pod) {
	switch pod.Status.Phase {
	case corev1.PodRunning:
		markStatusRunning(trs, v1beta1.TaskRunReasonRunning.String(), "Not all Steps in the Task have finished executing")
	case corev1.PodPending:
		switch {
		case IsPodExceedingNodeResources(pod):
			markStatusRunning(trs, ReasonExceededNodeResources, "TaskRun Pod exceeded available resources")
		case isPodHitConfigError(pod):
			markStatusFailure(trs, ReasonCreateContainerConfigError, "Failed to create pod due to config error")
		default:
			markStatusRunning(trs, ReasonPending, getWaitingMessage(pod))
		}
	}
}

// DidPipelineRunFail check the status of pod to decide if related pipelinerun is failed
func DidPipelineRunFail(containerStatuses []corev1.ContainerStatus) bool {
	f := false
	for _, s := range containerStatuses {
		if IsContainerStep(s.Name) {
			if s.State.Terminated != nil {
				f = f || s.State.Terminated.ExitCode != 0 || isOOMKilled(s)
			}
		}
	}
	return f
}

// note: relies on spec.containers and status.containerstatuses being in the same order
func getPtContainers(logger *zap.SugaredLogger, pod *corev1.Pod) (map[string]taskRunStatusInfo, error) {
	ptcs := make(map[string]taskRunStatusInfo, len(pod.Spec.Containers))
	logger.Infof("len spec containers: %d, len status containers: %d", len(pod.Spec.Containers), len(pod.Status.ContainerStatuses))
	for i, c := range pod.Spec.Containers {
		// TODO: sidecars
		taskName, stepName, err := TaskAndStepNameFromContainerName(c.Name)
		if err != nil {
			return nil, err
		}
		newSSI := stepStatusInfo{stepName: stepName, container: &c}
		if len(pod.Spec.Containers) == len(pod.Status.ContainerStatuses) {
			newSSI.containerStatus = &pod.Status.ContainerStatuses[i]
		} else {
			//logger.Infof("len container spec %d, len container status %d", len(pod.Spec.Containers), len(pod.Status.ContainerStatuses))
		}
		elem, ok := ptcs[taskName]
		if ok {
			elem.stepStatusInfos = append(elem.stepStatusInfos, newSSI)
		} else {
			newTaskRunStatusInfo := taskRunStatusInfo{
				pipelineTaskName: taskName, // TODO
				taskRunName:      taskName,
				stepStatusInfos:  []stepStatusInfo{newSSI},
			}
			ptcs[taskName] = newTaskRunStatusInfo
		}
	}
	return ptcs, nil
}

func areStepsComplete(containerStatuses []corev1.ContainerStatus) bool {
	stepsComplete := len(containerStatuses) > 0
	for _, s := range containerStatuses {
		if IsContainerStep(s.Name) {
			if s.State.Terminated == nil {
				stepsComplete = false
			}
		}
	}
	return stepsComplete
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
				if isOOMKilled(s) {
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

// markStatusRunning sets taskrun status to running
func markStatusRunning(trs *v1beta1.TaskRunStatus, reason, message string) {
	trs.SetCondition(&apis.Condition{
		Type:    apis.ConditionSucceeded,
		Status:  corev1.ConditionUnknown,
		Reason:  reason,
		Message: message,
	})
}

func markPRStatusRunning(prs *v1beta1.PipelineRunStatus, reason, message string) {
	prs.SetCondition(&apis.Condition{
		Type:    apis.ConditionSucceeded,
		Status:  corev1.ConditionUnknown,
		Reason:  reason,
		Message: message,
	})
}

// markStatusFailure sets taskrun status to failure with specified reason
func markStatusFailure(trs *v1beta1.TaskRunStatus, reason string, message string) {
	trs.SetCondition(&apis.Condition{
		Type:    apis.ConditionSucceeded,
		Status:  corev1.ConditionFalse,
		Reason:  reason,
		Message: message,
	})
}

// markStatusFailure sets taskrun status to failure with specified reason
func markPRStatusFailure(prs *v1beta1.PipelineRunStatus, reason string, message string) {
	prs.SetCondition(&apis.Condition{
		Type:    apis.ConditionSucceeded,
		Status:  corev1.ConditionFalse,
		Reason:  reason,
		Message: message,
	})
}

// markStatusSuccess sets taskrun status to success
func markStatusSuccess(trs *v1beta1.TaskRunStatus) {
	trs.SetCondition(&apis.Condition{
		Type:    apis.ConditionSucceeded,
		Status:  corev1.ConditionTrue,
		Reason:  v1beta1.TaskRunReasonSuccessful.String(),
		Message: "All Steps have completed executing",
	})
}

// markStatusSuccess sets taskrun status to success
func markPRStatusSuccess(prs *v1beta1.PipelineRunStatus) {
	prs.SetCondition(&apis.Condition{
		Type:    apis.ConditionSucceeded,
		Status:  corev1.ConditionTrue,
		Reason:  v1beta1.TaskRunReasonSuccessful.String(),
		Message: "All Tasks have completed executing",
	})
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

func isOOMKilled(s corev1.ContainerStatus) bool {
	return s.State.Terminated.Reason == oomKilled
}
