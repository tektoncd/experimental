package v1alpha1_test

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/experimental/workflows/pkg/apis/workflows/v1alpha1"
	"github.com/tektoncd/experimental/workflows/test/parse"
	"knative.dev/pkg/apis"
)

func TestValidateFilters(t *testing.T) {
	tcs := []struct {
		name    string
		wf      *v1alpha1.Workflow
		wantErr *apis.FieldError
	}{{
		name: "gitref filter with push event",
		wf: parse.MustParseWorkflow(t, "trigger-workflow", "some-namespace", `
spec:
  triggers:
  - name: on-push
    event:
      types: ["push"]
    filters:
      gitRef:
        regex: "^main$"
`),
	}, {
		name: "gitref filter with pull_request event",
		wf: parse.MustParseWorkflow(t, "trigger-workflow", "some-namespace", `
spec:
  triggers:
  - name: on-pr
    event:
      types: ["pull_request"]
    filters:
      gitRef:
        regex: "^main$"
`),
	}, {
		name: "gitref filter with other event type",
		wf: parse.MustParseWorkflow(t, "trigger-workflow", "some-namespace", `
spec:
  triggers:
  - name: on-event
    event:
      types: ["some-other-event-type"]
    filters:
      gitRef:
        regex: "^main$"
`),
		wantErr: &apis.FieldError{Message: "gitRef filter can be used only with 'push' and 'pull_request' events but got event some-other-event-type"},
	}}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.wf.Validate(context.Background())
			if d := cmp.Diff(tc.wantErr.Error(), err.Error()); d != "" {
				t.Errorf("wrong error: %s", d)
			}
		})
	}
}
