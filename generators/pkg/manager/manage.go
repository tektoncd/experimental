package manager

import (
	"context"
	"fmt"
	"io"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/apimachinery/pkg/util/yaml"
)

// Create the Kubernetes client
func GetKubeClient(kubeconfig string) (client.Client, error) {

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("fail to build config from the flags: %w", err)
	}

	return client.New(config, client.Options{})
}

func read(reader io.Reader) ([]*unstructured.Unstructured, error) {
	//read the input file
	r := yaml.NewYAMLToJSONDecoder(reader)
	list := []*unstructured.Unstructured{}
	for {
		//unmarshal the yaml object
		u := new(unstructured.Unstructured)
		if err := r.Decode(u); err != nil {
			if err == io.EOF {
				return list, nil
			}
			return list, fmt.Errorf("failed to unmarshal the reader: %w", err)
		}
		//setup the namespace to default
		if u.GetNamespace() == "" {
			u.SetNamespace("default")
		}
		list = append(list, u)
	}
}

// Create resources from io.Reader on Kubernetes objects using client
func CreateResource(ctx context.Context, cl client.Writer, reader io.Reader) error {
	resources, err := read(reader)
	if err != nil {
		return fmt.Errorf("fail to read the resources: %w", err)
	}
	for _, r := range resources {
		if err := cl.Create(ctx, r); err != nil {
			return fmt.Errorf("failed to create the resource: %w", err)
		}
	}
	return nil
}

// Delete resources from io.Reader on Kubernetes objects using client
func DeleteResource(ctx context.Context, cl client.Writer, reader io.Reader) error {
	resources, err := read(reader)
	if err != nil {
		return fmt.Errorf("fail to read the resources: %w", err)
	}
	for _, r := range resources {
		if err := cl.Delete(ctx, r); err != nil {
			return fmt.Errorf("failed to delete the resource: %w", err)
		}
	}
	return nil
}
