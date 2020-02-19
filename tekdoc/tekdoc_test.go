package main

import (
	"bytes"
	"github.com/google/go-cmp/cmp"
	"testing"
)

func TestTekdoc(t *testing.T) {
	task, err := read("buildkit-daemonless.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if task.Kind != "Task"{
		t.Error("This is not a task file")
	}

	b := new(bytes.Buffer)
	err = printTask(b,task)
	if err != nil {
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
	if diff := cmp.Diff(v, b.String()); diff != ""{
		t.Error(diff)
	}
}

