package oci

import (
	"context"
	"fmt"
	"strings"

	"github.com/containerd/containerd/remotes/docker"
	orascontent "github.com/deislabs/oras/pkg/content"
	orascontext "github.com/deislabs/oras/pkg/context"
	"github.com/deislabs/oras/pkg/oras"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func init() {
	orascontext.GetLogger(context.Background()).Logger.SetLevel(logrus.ErrorLevel)
}

// PushImage will publish the given ImageReference and the provided resources in the proper format to an external OCI
// registry.
func PushImage(name name.Reference, contents []ParsedTektonResource) error {
	resolver := docker.NewResolver(docker.ResolverOptions{})
	memoryStore := orascontent.NewMemoryStore()
	descriptors := []v1.Descriptor{}

	for _, resource := range contents {
		descriptor := memoryStore.Add(
			getLayerName(resource.Kind.Kind, resource.Name),
			getLayerMediaType(resource),
			[]byte(resource.Contents),
		)
		descriptors = append(descriptors, descriptor)
	}

	pushName := name.String()
	desc, err := oras.Push(
		context.Background(),
		resolver,
		pushName,
		memoryStore,
		descriptors,
		oras.WithConfigMediaType("application/vnd.cdf.tekton.catalog.v1alpha1+json"),
	)
	if err != nil {
		return errors.Wrap(err, "failed to push to registry")
	}

	fmt.Printf("Pushed %s@%s to remote registry\n", pushName, desc.Digest)
	return nil
}

// PullImage will fetch the image and return the Tekton resource specified by the kind and name.
func PullImage(ref name.Reference, kind string, name string) ([]byte, error) {
	resolver := docker.NewResolver(docker.ResolverOptions{})
	memoryStore := orascontent.NewMemoryStore()

	_, _, err := oras.Pull(
		context.Background(),
		resolver,
		ref.String(),
		memoryStore,
		oras.WithPullEmptyNameAllowed(),
		oras.WithAllowedMediaTypes(TektonMediaTypes()),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch image")
	}

	// Attempt to fetch the contents from the store.
	_, contents, ok := memoryStore.GetByName(getLayerName(kind, name))
	if !ok {
		return nil, errors.Errorf("could not find %s/%s in image contents", kind, name)
	}

	return contents, nil
}

func getLayerName(kind string, name string) string {
	return fmt.Sprintf("%s/%s", strings.ToLower(kind), name)
}

func getLayerMediaType(resource ParsedTektonResource) string {
	return fmt.Sprintf("application/vnd.cdf.tekton.catalog.%s.v1alpha1+yaml", strings.ToLower(resource.Kind.Kind))
}
