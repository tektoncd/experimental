package manager

import (
	"bytes"
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

type testClient struct {
	client.Writer
	list []runtime.Object
}

func (t *testClient) Create(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
	t.list = append(t.list, obj)
	return nil
}

func (t *testClient) Delete(ctx context.Context, obj runtime.Object, opts ...client.DeleteOption) error {
	t.list = append(t.list, obj)
	return nil
}

// testdata
var resources = []*unstructured.Unstructured{
	{
		Object: map[string]interface{}{
			"name":       "test-task",
			"kind":       "Task",
			"apiVersion": "tekton.dev/v1beta1",
		},
	},

	{
		Object: map[string]interface{}{
			"name":       "test-task",
			"kind":       "Task",
			"apiVersion": "tekton.dev/v1beta1",
			"metadata": map[string]interface{}{
				"namespace": "default",
			},
		},
	},

	{
		Object: map[string]interface{}{
			"name":       "test-task",
			"kind":       "Task",
			"apiVersion": "tekton.dev/v1beta1",
			"metadata": map[string]interface{}{
				"namespace": "non-default",
			},
		},
	},

	{
		Object: map[string]interface{}{
			"apiVersion": "tekton.dev/v1beta1",
			"kind":       "Task",
			"metadata": map[string]interface{}{
				"namespace": "default",
				"name":      "echo-hello-world",
			},
			"spec": map[string]interface{}{
				"steps": map[string]interface{}{
					"name":    "echo",
					"image":   "ubuntu",
					"command": []interface{}{string("echo")},
					"args":    []interface{}{string("Hello world")},
				},
			},
		},
	},
}

func TestApplyResource(t *testing.T) {
	tables := []struct {
		name string
		u    *unstructured.Unstructured
		want []runtime.Object
	}{
		{
			name: "ResourceWithoutNamespace",
			u:    resources[0],
			want: []runtime.Object{
				&unstructured.Unstructured{
					Object: map[string]interface{}{
						"name":       "test-task",
						"kind":       "Task",
						"apiVersion": "tekton.dev/v1beta1",
						"metadata": map[string]interface{}{
							"namespace": "default",
						},
					},
				},
			},
		},

		{
			name: "ResourceWithDefaultNamespace",
			u:    resources[1],
			want: []runtime.Object{
				resources[1],
			},
		},

		{
			name: "ResourceWithNonDefaultNamespace",
			u:    resources[2],
			want: []runtime.Object{
				resources[2],
			},
		},

		{
			name: "TektonTaskResourceWithNamespace",
			u:    resources[3],
			want: []runtime.Object{
				resources[3],
			},
		},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			b, err := yaml.Marshal(table.u)
			if err != nil {
				t.Fatalf("fail to marshal the test data: %v", err)
			}

			buf := bytes.NewBuffer(b)
			client := &testClient{}
			if err := CreateResource(context.Background(), client, buf); err != nil {
				t.Fatalf("fail to create resource: %v", err)
			}

			if diff := cmp.Diff(table.want, client.list); diff != "" {
				t.Errorf("Objects mismatch (-want +got): \n %s", diff)
			}
		})
	}

}

func TestDeleteResource(t *testing.T) {
	tables := []struct {
		name string
		u    *unstructured.Unstructured
		want []runtime.Object
	}{
		{
			name: "ResourceWithoutNamespace",
			u:    resources[0],
			want: []runtime.Object{
				&unstructured.Unstructured{
					Object: map[string]interface{}{
						"name":       "test-task",
						"kind":       "Task",
						"apiVersion": "tekton.dev/v1beta1",
						"metadata": map[string]interface{}{
							"namespace": "default",
						},
					},
				},
			},
		},

		{
			name: "ResourceWithDefaultNamespace",
			u:    resources[1],
			want: []runtime.Object{
				resources[1],
			},
		},

		{
			name: "ResourceWithNonDefaultNamespace",
			u:    resources[2],
			want: []runtime.Object{
				resources[2],
			},
		},

		{
			name: "TektonTaskResourceWithNamespace",
			u:    resources[3],
			want: []runtime.Object{
				resources[3],
			},
		},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			b, err := yaml.Marshal(table.u)
			if err != nil {
				t.Fatalf("fail to marshal the test data: %v", err)
			}

			buf := bytes.NewBuffer(b)
			client := &testClient{}
			if err := DeleteResource(context.Background(), client, buf); err != nil {
				t.Fatalf("fail to create resource: %v", err)
			}

			if diff := cmp.Diff(table.want, client.list); diff != "" {
				t.Errorf("Objects mismatch (-want +got): \n %s", diff)
			}
		})
	}
}
