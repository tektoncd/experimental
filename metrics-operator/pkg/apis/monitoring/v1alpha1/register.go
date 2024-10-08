package v1alpha1

import (
	"github.com/tektoncd/experimental/metrics-operator/pkg/apis/monitoring"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// SchemeGroupVersion is group version used to register these objects
var SchemeGroupVersion = schema.GroupVersion{Group: monitoring.GroupName, Version: "v1alpha1"}

// Kind takes an unqualified kind and returns back a Group qualified GroupKind
func Kind(kind string) schema.GroupKind {
	return SchemeGroupVersion.WithKind(kind).GroupKind()
}

// Resource takes an unqualified resource and returns a Group qualified GroupResource
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

var (
	schemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	// AddToScheme adds Build types to the scheme.
	AddToScheme = schemeBuilder.AddToScheme
)

// Adds the list of known types to Scheme.
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&TaskMonitor{},
		&TaskMonitorList{},
		&TaskRunMonitor{},
		&TaskRunMonitorList{},
		&PipelineMonitor{},
		&PipelineMonitorList{},
		&PipelineRunMonitor{},
		&PipelineRunMonitorList{},
	)
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}
