package action

import (
	"errors"

	"github.com/tektoncd/experimental/oci/pkg"
)

// Push will perform the `push` action by recursively reading all of the
// Tekton specs passed in, bundling it into an image, and pushing the result
// to an OCI-compliant repository.
func Push(ref string, filePaths []string) error {
	// Validate the parameters.
	if ref == "" || len(filePaths) == 0 {
		return errors.New("must specify a valid image name and file paths")
	}

	resources, err := pkg.ReadPaths(filePaths)
	if err != nil {
		return err
	}

	name, err := pkg.ValidateImageName(ref)
	if err != nil {
		return err
	}

	return pkg.PushImage(*name, resources)
}
