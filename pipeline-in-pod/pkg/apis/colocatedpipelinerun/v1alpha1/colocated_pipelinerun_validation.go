package v1alpha1

func ValidateColocatedPipelineRun(cpr *ColocatedPipelineRun) error {
	/*
		TODO
		- only one of PipelineRef, PipelineSpec
		- spec.ref not populated
		- spec.Kind = colocatedPipelineRun
		- colocatedpipelinerun timeouts has only timeouts.pipeline
	*/
	return nil
}
