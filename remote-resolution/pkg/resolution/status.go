package resolution

// processing reasons
const (
	// ReasonResolutionInProgress is used to indicate that there are
	// no issues with the parameters of a request and that a
	// resolver is working on the ResourceRequest.
	ReasonResolutionInProgress = "ResolutionInProgress"
)

// happy reasons
const (
	// ReasonResolutionSuccessful is used to indicate that
	// resolution of a resource has completed successfully.
	ReasonResolutionSuccessful = "ResolutionSuccessful"
)

// unhappy reasons
const (
	// ReasonTaskRunResolutionFailed indicates that references within the
	// TaskRun could not be resolved.
	ReasonTaskRunResolutionFailed = "TaskRunResolutionFailed"

	// ReasonCouldntGetTask indicates that a reference to a task did not
	// successfully resolve to a task object. This is distinct from
	// ReasonTaskRunResolutionFailed because it indicates a failure
	// fetching the referenced Task rather than failure to interpret the
	// reference.
	ReasonCouldntGetTask = "CouldntGetTask"

	// ReasonPipelineRunResolutionFailed indicates that references within the
	// PipelineRun could not be resolved.
	ReasonPipelineRunResolutionFailed = "PipelineRunResolutionFailed"

	// ReasonCouldntGetPipeline indicates that a reference to a pipeline did
	// not successfully resolve to a pipeline object.
	ReasonCouldntGetPipeline = "CouldntGetPipeline"

	// ResolutionFailed indicates that some part of the resolution
	// process failed.
	ReasonResolutionFailed = "ResolutionFailed"
)
