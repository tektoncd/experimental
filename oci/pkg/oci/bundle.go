package oci

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"gopkg.in/yaml.v2"
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
		parsedResources = append(parsedResources, resource...)
	}

	return parsedResources, nil
}

// readPath will read the contents of the file at filePath and use the K8s deserializer to attempt to marshal the text
// into a Tekton struct. This will fail if the resource is unparseable or not a Tekton resource.
func readPath(filePath string) ([]ParsedTektonResource, error) {
	contents, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Try to tease out the type of the file from the extension and load them into a slice (if there are multiple
	// entities in a single file).
	var entities []string
	switch ext := path.Ext(filePath); true {
	case ext == ".yaml" || ext == ".yml":
		entities = strings.Split(string(contents), "---")
	case ext == ".json":
		var partials []interface{}
		err = json.Unmarshal(contents, &partials)
		if err != nil {
			return nil, errors.Wrap(err, "could not marshal contents to json")
		}
		entities = make([]string, 0, len(partials))
		for _, element := range partials {
			rawElement, err := json.Marshal(element)
			if err != nil {
				return nil, errors.Wrapf(err, "could not re-marshal %+v into json", element)
			}
			entities = append(entities, string(rawElement))
		}
	default:
		return nil, fmt.Errorf("can't parse resources of type %s", ext)
	}

	resources := make([]ParsedTektonResource, 0, len(entities))
	for _, entity := range entities {
		resource, err := decodeObject(entity)
		if err != nil {
			// We are not going to bail if we find an unparseable resource, rather, we will just skip it.
			fmt.Printf("skipping %s because %s", filePath, err.Error())
			continue
		}
		resources = append(resources, *resource)
	}
	return resources, nil
}

// decodeObject will attempt to decode a yaml or json string into a single Kubernetes or CRD object and return the
// parsed representation.
func decodeObject(contents string) (*ParsedTektonResource, error) {
	object, kind, err := scheme.Codecs.UniversalDeserializer().Decode([]byte(contents), nil, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "resource is not a valid Kubernetes resource:\n%s", contents)
	}

	if kind.GroupVersion().Identifier() != v1alpha1.SchemeGroupVersion.Identifier() {
		return nil, errors.Wrapf(err, "resource is not a valid Kubernetes resource:\n%s", contents)
	}

	resourceName := getResourceName(object, kind.Kind)
	// Convert the structured data into yaml to get a "clean" copy of the resource.
	rawContents, err := yaml.Marshal(object)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal %+v to yaml", object)
	}

	fmt.Printf("Adding %s:%s to image bundle\n", kind.Kind, resourceName)
	return &ParsedTektonResource{
		Name:     resourceName,
		Kind:     kind,
		Contents: string(rawContents),
		Object:   object,
	}, nil
}

// getResourceName will reflexively read out the ObjectMeta.Name field from the Tekton resource since all known Tekton
// CRDs use the K8s ObjectMeta field.
func getResourceName(object runtime.Object, kind string) string {
	reflection := reflect.Indirect(reflect.ValueOf(object))
	return reflection.FieldByName("ObjectMeta").FieldByName("Name").String()
}
