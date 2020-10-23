package oci

import (
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
)

// PushImage bundles the Tekton resources and pushes it to an image with the given reference.
func PushImage(ref name.Reference, resources []ParsedTektonResource) error {
	img := empty.Image
	for _, r := range resources {
		log.Printf("[%s]: adding %s:%s\n", ref.Name(), r.Kind, r.Name)
		l, err := tarball.LayerFromReader(strings.NewReader(r.Contents))
		if err != nil {
			return fmt.Errorf("Error creating layer for resource %s/%s: %w", r.Kind, r.Name, err)
		}
		img, err = mutate.Append(img, mutate.Addendum{
			Layer: l,
			Annotations: map[string]string{
				"dev.tekton.image.name":       r.Name,
				"dev.tekton.image.kind":       strings.ToLower(r.Kind.Kind),
				"dev.tekton.image.apiVersion": r.Kind.Version,
			},
		})
		if err != nil {
			return fmt.Errorf("Error appending resource %q: %w", r.Name, err)
		}
	}

	d, err := img.Digest()
	if err != nil {
		return err
	}

	if err := remote.Write(ref, img, remote.WithAuthFromKeychain(authn.DefaultKeychain)); err != nil {
		return err
	}

	log.Println("Pushed", ref.Context().Digest(d.String()))
	return nil
}

// PullImage fetches the image and extracts the Tekton resource with the given kind and name.
func PullImage(ref name.Reference, kind string, name string) ([]byte, error) {
	// TODO: When this is moved into the Tekton controller, authorize this
	// pull as a Service Account in the cluster, and don't rely on the
	// contents of ~/.docker/config.json (which won't exist).
	img, err := remote.Image(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	if err != nil {
		return nil, fmt.Errorf("Error pulling %q: %w", ref, err)
	}

	m, err := img.Manifest()
	if err != nil {
		return nil, err
	}
	ls, err := img.Layers()
	if err != nil {
		return nil, err
	}
	var layer v1.Layer
	for idx, l := range m.Layers {
		// TODO: Check for custom media type.
		if l.Annotations["dev.tekton.image.name"] == getLayerName(kind, name) {
			layer = ls[idx]
			break
		}
	}
	if layer == nil {
		return nil, fmt.Errorf("Resource %s/%s not found", kind, name)
	}
	rc, err := layer.Uncompressed()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return ioutil.ReadAll(rc)
}

func getLayerName(kind string, name string) string {
	return fmt.Sprintf("%s/%s", strings.ToLower(kind), name)
}
