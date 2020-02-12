package oci

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"reflect"

	"github.com/pkg/errors"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
)

func init() {
	// Because we are using the K8s deserializer, we need to add Tekton's types to it.
	schemeBuilder := runtime.NewSchemeBuilder(v1alpha1.AddToScheme)
	schemeBuilder.AddToScheme(scheme.Scheme)
}

// ParsedTektonResource represents a full Tekton task, pipeline, etc that has been read in from the user along with
// metadata about the resource.
type ParsedTektonResource struct {
	Name     string
	Kind     *schema.GroupVersionKind
	Contents string
	Object   runtime.Object
}

// ReadPaths will recursively search each file path for Tekton resources and return the parsed specs or an error.
func ReadPaths(filePaths []string) ([]ParsedTektonResource, error) {
	parsedResources := []ParsedTektonResource{}

	for _, filePath := range filePaths {
		// Check both the existence of the file and if it is a directory.
		info, err := os.Stat(filePath)
		if err != nil {
			return nil, errors.Wrapf(err, "No file or directory found at %s", filePath)
		}

		// If this is a directory, recursively read the subpaths.
		if info.IsDir() {
			files, err := ioutil.ReadDir(filePath)
			if err != nil {
				return nil, errors.Wrapf(err, "Unable to read dir %s", filePath)
			}

			subpaths := make([]string, 0, len(files))
			for _, file := range files {
				subpaths = append(subpaths, path.Join(filePath, file.Name()))
			}

			// Recursively call this function with the sub-paths of this directory.
			resources, err := ReadPaths(subpaths)
			if err != nil {
				return nil, err
			}
			parsedResources = append(parsedResources, resources...)
			continue
		}

		// This path points to a single file. Read it and append the parsed resource.
		resource, err := readPath(filePath)
		if err != nil {
			return nil, err
		}
		parsedResources = append(parsedResources, *resource)
	}

	return parsedResources, nil
}

// readPath will read the contents of the file at filePath and use the K8s deserializer to attempt to marshal the text
// into a Tekton struct. This will fail if the resource is unparseable or not a Tekton resource.
func readPath(filePath string) (*ParsedTektonResource, error) {
	contents, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	object, kind, err := scheme.Codecs.UniversalDeserializer().Decode(contents, nil, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "Resource at %s is not a valid Kubernetes resource:\n%s", filePath, string(contents))
	}

	if kind.GroupVersion().Identifier() != v1alpha1.SchemeGroupVersion.Identifier() {
		return nil, errors.New(fmt.Sprintf("Resource at %s is not a valid Tekton kind:\n%s", filePath, string(contents)))
	}

	resourceName := getResourceName(object, kind.Kind)

	fmt.Printf("Adding %s:%s to image bundle\n", kind.Kind, resourceName)
	return &ParsedTektonResource{
		Name:     resourceName,
		Kind:     kind,
		Contents: string(contents),
		Object:   object,
	}, nil
}

// getResourceName will reflexively read out the ObjectMeta.Name field from the Tekton resource since all known Tekton
// CRDs use the K8s ObjectMeta field.
func getResourceName(object runtime.Object, kind string) string {
	reflection := reflect.Indirect(reflect.ValueOf(object))
	return reflection.FieldByName("ObjectMeta").FieldByName("Name").String()
}
