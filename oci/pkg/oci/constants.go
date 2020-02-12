package oci

const (
	// TektonCatalogMediaType is the root catalog.
	TektonCatalogMediaType = "application/vnd.cdf.tekton.catalog.v1alpha1+json"
	// TektonTaskMediaType is a Tekton task spec.
	TektonTaskMediaType = "application/vnd.cdf.tekton.catalog.task.v1alpha1+yaml"
	// TektonPipelineMediaType is a Tekton pipeline spec.
	TektonPipelineMediaType = "application/vnd.cdf.tekton.catalog.pipeline.v1alpha1+yaml"
	// TektonPipelineResourceMediaType is a Tekton pipeline resource spec.
	TektonPipelineResourceMediaType = "application/vnd.cdf.tekton.catalog.pipelineresource.v1alpha1+yaml"
)

// TektonMediaTypes returns a slice of the various media types supported by our Tekton OCI implementation.
func TektonMediaTypes() []string {
	return []string{
		TektonCatalogMediaType,
		TektonPipelineMediaType,
		TektonTaskMediaType,
		TektonPipelineResourceMediaType,
	}
}
