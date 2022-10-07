package v1alpha1_test

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/experimental/workflows/test/parse"
)

func TestSetDefaults(t *testing.T) {
	wfWithoutTriggerNames := parse.MustParseWorkflow(t, "trigger-workflow", "some-namespace", `
spec:
  triggers:
  - event:
      type: "push"
    filters:
      gitRef:
        regex: "^main$"
`)
	wfWithoutTriggerNames.SetDefaults(context.Background())
	wfWithTriggerNames := parse.MustParseWorkflow(t, "trigger-workflow", "some-namespace", `
spec:
  triggers:
  - name: "0"
    event:
      type: "push"
    filters:
      gitRef:
        regex: "^main$"
`)
	if d := cmp.Diff(wfWithTriggerNames, wfWithoutTriggerNames); d != "" {
		t.Errorf("wrong triggers: %s", d)
	}
}
