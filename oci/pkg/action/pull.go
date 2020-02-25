package action

import (
	"errors"
	"fmt"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/tektoncd/experimental/oci/pkg/oci"
)

// Pull retrieves a specific named Tekton resource from the specified OCI image.
func Pull(r string, kind string, n string) error {
	// Validate the parameters.
	if r == "" || kind == "" || n == "" {
		return errors.New("must specify an image reference, kind, and resource name")
	}

	ref, err := name.ParseReference(r)
	if err != nil {
		return err
	}

	contents, err := oci.PullImage(ref, kind, n)
	if err != nil {
		return err
	}
	fmt.Print(string(contents))
	return nil
}
