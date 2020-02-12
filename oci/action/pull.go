package action

import (
	"errors"
	"fmt"

	"github.com/tektoncd/experimental/oci/pkg"
)

// Pull will perform the `pull` action by retrieving a specific named Tekton resource from the specified OCI image.
func Pull(ref string, kind string, name string) error {
	// Validate the parameters.
	if ref == "" || kind == "" || name == "" {
		return errors.New("must specify an image reference, kind, and resource name")
	}

	imageReference, err := pkg.ValidateImageName(ref)
	if err != nil {
		return err
	}

	contents, err := pkg.PullImage(*imageReference, kind, name)
	fmt.Print(string(contents))

	return nil
}
