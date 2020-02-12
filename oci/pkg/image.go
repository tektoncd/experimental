package pkg

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/containerd/containerd/remotes/docker"
	orascontent "github.com/deislabs/oras/pkg/content"
	orascontext "github.com/deislabs/oras/pkg/context"
	"github.com/deislabs/oras/pkg/oras"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func init() {
	orascontext.GetLogger(context.Background()).Logger.SetLevel(logrus.ErrorLevel)
}

// ImageReference is a generic wrapper around the parts of an OCI image name. Not all of the fields exist for every
// image.
type ImageReference struct {
	FullName string
	Name     string
	Registry string
	Hash     string
	Tag      string
}

// ValidateImageName will return nil if name is a valid, fully qualified image reference. This is based on
// https://github.com/helm/helm/blob/7ffc879f137bd3a69eea53349b01f05e3d1d2385/internal/experimental/registry/reference.go#L48
func ValidateImageName(name string) (*ImageReference, error) {
	if name == "" {
		return nil, fmt.Errorf("%s is not a valid image name", name)
	}

	return getNameParts(name)
}

// getNameParts will attempt to return a { "hostname/image-name", "tag or sha" } list of the provided image name. There maybe less than the full 2 elements if one of the parts isn't included in the original name.
func getNameParts(name string) (*ImageReference, error) {
	ref := ImageReference{
		FullName: name,
	}

	remainingName := name

	// First we split the @ since only the sha form is expected to have this.
	shaParts := strings.Split(name, "@")
	if len(shaParts) > 2 {
		return nil, fmt.Errorf("invalid image name %s, too many @ symbols", name)
	}
	if len(shaParts) == 2 {
		// We have a sha-tagged image name so we can return this split as is.
		ref.Hash = shaParts[1]
		remainingName = shaParts[0]
	}

	// Try splitting on : to see if this is image has a tag.
	nameParts := strings.Split(remainingName, ":")
	if len(nameParts) > 3 {
		return nil, fmt.Errorf("invalid image name %s, too many : symbols", name)
	}
	if len(nameParts) == 3 {
		// There was a tag and also a port on the domain, eg { localhost, 5000/my-image, tag-1 }.
		ref.Tag = nameParts[2]
		remainingName = strings.Join(nameParts[:2], ":")
	}
	if len(nameParts) == 2 {
		// Either there was only a port and no tag, or only a tag.
		if _, err := strconv.Atoi(strings.Split(nameParts[1], "/")[0]); err != nil {
			// We could not parse the beginning part of the second index as a number so it isn't a port. The situation was
			// { my-domain.com/image, tag-1 } instead of { localhost, 5000/my-image }.
			ref.Tag = nameParts[1]
			remainingName = nameParts[0]
		} else {
			ref.Tag = "latest"
			// If there was no tag, remainingName = localhost:5000/my-image
		}
	}
	if len(nameParts) == 1 {
		// We didn't have a tag or port.
		ref.Tag = "latest"
	}

	// Finally, break out the registry url from the image name.
	registryParts := strings.Split(remainingName, "/")
	ref.Registry = registryParts[0]
	ref.Name = registryParts[1]

	return &ref, nil
}

// PushImage will publish the given ImageReference and the provided resources in the proper format to an external OCI
// registry.
func PushImage(name ImageReference, contents []ParsedTektonResource) error {
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

	pushName := getPushName(name)
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
func PullImage(ref ImageReference, kind string, name string) ([]byte, error) {
	resolver := docker.NewResolver(docker.ResolverOptions{})
	memoryStore := orascontent.NewMemoryStore()

	_, _, err := oras.Pull(
		context.Background(),
		resolver,
		ref.FullName,
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

func getPushName(name ImageReference) string {
	return fmt.Sprintf("%s/%s:%s", name.Registry, name.Name, name.Tag)
}
