package git

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"go.uber.org/zap"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func init() {
	// Because we are using the K8s deserializer, we need to add Tekton's types to it.
	//schemeBuilder := runtime.NewSchemeBuilder(v1alpha1.AddToScheme, v1beta1.AddToScheme)
	//schemeBuilder.AddToScheme(scheme.Scheme)
}

type Repo struct {
	Path        string
	ContextPath string
	head        string
	Log         *zap.SugaredLogger
}

func (r Repo) Head() string {
	if r.head == "" {
		head, _ := rawGit("", "rev-parse", "HEAD")
		r.head = head
	}
	return r.head
}

type (
	TektonResource struct {
		Name     string
		Kind     string
		Versions []TekonResourceVersion
	}

	TekonResourceVersion struct {
		Version     string
		DisplayName string
		Path        string
		Description string
		Tags        []string
	}
)

func (r Repo) ParseTektonResources() ([]TektonResource, error) {
	// TODO(sthaha): may be in parallel
	// TODO(sthaha): replace it by channels and stream and write?
	// TODO(sthaha): get task kind from scheme ?
	kinds := []string{"Task", "Pipeline"}
	resources := []TektonResource{}
	for _, k := range kinds {
		ret, err := r.findResourcesByKind(k)
		if err != nil {
			return []TektonResource{}, err
		}
		resources = append(resources, ret...)
	}
	return resources, nil
}

func ignoreNotExists(err error) error {
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (r Repo) findResourcesByKind(kind string) ([]TektonResource, error) {
	log := r.Log.With("kind", kind)
	log.Info("looking for resources")

	// TODO(sthaha): can we use  GVK to find plural?
	kindPath := filepath.Join(r.Path, r.ContextPath, strings.ToLower(kind))
	resources, err := ioutil.ReadDir(kindPath)
	if err != nil {
		r.Log.Errorf("failed to find %s: %s", kind, err)
		// NOTE: returns empty task list; upto caller to check for error
		return []TektonResource{}, ignoreNotExists(err)
	}

	ret := []TektonResource{}
	for _, res := range resources {
		if !res.IsDir() {
			log.Warnf("ignoring %s  not a directory for %s", res.Name(), kind)
			continue
		}

		tknRes, err := r.parseResource(kind, kindPath, res)
		if err != nil {
			// TODO(sthaha): do something about invalid tasks
			r.Log.Error(err)
			continue
		}
		ret = append(ret, *tknRes)
	}

	r.Log.Info("found ", kind, " len ", len(ret))

	return ret, nil

}

var errInvalidResourceDir = errors.New("invalid resource dir")

func (r Repo) parseResource(kind, kindPath string, res os.FileInfo) (*TektonResource, error) {
	// TODO(sthaha): move this to a different package that can scan a Repo
	r.Log.Info("checking path", kindPath, " resource: ", res.Name())
	// path/<task>/<version>[>
	pattern := filepath.Join(kindPath, res.Name(), "*", res.Name()+".yaml")

	matches, err := filepath.Glob(pattern)
	if err != nil {
		r.Log.Error(err, "failed to find tasks")
		return nil, errInvalidResourceDir
	}

	ret := &TektonResource{
		Name:     res.Name(),
		Kind:     kind,
		Versions: []TekonResourceVersion{},
	}

	for _, m := range matches {
		r.Log.Info("      found file: ", m)

		version, err := r.parseResourceVersion(m, kind)
		if err != nil {
			r.Log.Error(err)
			continue
		}

		ret.Versions = append(ret.Versions, *version)
	}

	return ret, nil
}

// parseResourceVersion will read the contents of the file at filePath and use the K8s deserializer to attempt to marshal the textjj
// into a Tekton struct. This will fail if the resource is unparseable or not a Tekton resource.
func (r Repo) parseResourceVersion(filePath string, kind string) (*TekonResourceVersion, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

	res, err := decodeResource(f, kind)
	if err != nil {
		r.Log.Error(err)
		return nil, err
	}
	if res == nil {
		return nil, nil
	}

	log := r.Log.With("kind", kind, "name", res.GetName())
	apiVersion := res.GetAPIVersion()
	log.Info("current kind: ", kind, apiVersion, res.GroupVersionKind())

	if apiVersion != v1alpha1.SchemeGroupVersion.Identifier() &&
		apiVersion != v1beta1.SchemeGroupVersion.Identifier() {
		log.Infof("Skipping unknown resource %s name: %s", res.GroupVersionKind(), res.GetName())
		return nil, errors.New("invalid resource " + apiVersion)
	}

	labels := res.GetLabels()
	version, ok := labels["app.kubernetes.io/version"]
	if !ok {
		log.Infof("Resource %s name: %s has no version information", res.GroupVersionKind(), res.GetName())
		return nil, fmt.Errorf("resource has no version info %s/%s", res.GroupVersionKind(), res.GetName())
	}

	annotations := res.GetAnnotations()
	displayName, ok := annotations["tekton.dev/displayName"]
	if !ok {
		log.With("action", "ignore").Infof(
			"Resource %s name: %s has no display name", res.GroupVersionKind(), res.GetName())
	}

	tags := annotations["tekton.dev/tags"]

	// first line
	description, found, err := unstructured.NestedString(res.Object, "spec", "description")
	if !found || err != nil {
		log.Infof("Resource %s name: %s has no description", res.GroupVersionKind(), res.GetName())
		return nil, fmt.Errorf("resource has no description %s/%s", res.GroupVersionKind(), res.GetName())
	}

	basePath := filepath.Join(r.Path, r.ContextPath)
	relPath, _ := filepath.Rel(basePath, filePath)

	ret := &TekonResourceVersion{
		Version:     version,
		DisplayName: displayName,
		Tags:        strings.Split(tags, ","),
		Description: description,
		Path:        relPath,
	}
	return ret, nil
}

func ignoreEOF(err error) error {
	if err == io.EOF {
		return nil
	}
	return err
}

// decode consumes the given reader and parses its contents as YAML.
func decodeResource(reader io.Reader, kind string) (*unstructured.Unstructured, error) {
	decoder := yaml.NewYAMLToJSONDecoder(reader)
	var res *unstructured.Unstructured

	for {
		res = &unstructured.Unstructured{}
		if err := decoder.Decode(res); err != nil {
			return nil, ignoreEOF(err)
		}

		if len(res.Object) == 0 {
			continue
		}
		if res.GetKind() == kind {
			break
		}
	}
	return res, nil
}
