package main

import (
	"bytes"
	"github.com/google/go-cmp/cmp"
	"testing"
)

func TestTekdoc(t *testing.T) {
	task, err := readTask("buildkit-daemonless.yaml")
	if err != nil {
		t.Fatalf("failed to read task: %v", err)
	}
	if task.Kind != "Task" {
		t.Errorf("wanted Task file got %v", task.Kind)
	}

	b := new(bytes.Buffer)
	if err := printTask(b, task); err != nil {
		t.Fatal(err)
	}
	v := `# buildkit-daemonless
## Install the Task
kubectl apply -f https://raw.githubusercontent.com/tektoncd/catalog/master/buildkit-daemonless/buildkit-daemonless.yaml
### Input:-
- DOCKERFILE, The name of the Dockerfile
- BUILDKIT_IMAGE, The name of the BuildKit image
### Resources:-
- source, git
### Output:-
- image, image
`
	if diff := cmp.Diff(v, b.String()); diff != "" {
		t.Errorf("-want, +got: %s", diff)
	}
}
