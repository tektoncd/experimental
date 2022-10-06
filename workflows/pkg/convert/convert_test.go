package convert_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/tektoncd/experimental/workflows/pkg/apis/workflows/v1alpha1"
	"github.com/tektoncd/experimental/workflows/pkg/client/clientset/versioned/scheme"
	"github.com/tektoncd/experimental/workflows/pkg/convert"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	triggersv1beta1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestToPipelineRun(t *testing.T) {
	tests := []struct {
		name     string
		workflow *v1alpha1.Workflow
		want     *pipelinev1beta1.PipelineRun
	}{{
		name: "workflow with pipeline ref",
		workflow: MustParseWorkflow(t, "basic-workflow", "some-namespace", `
spec:
  pipeline:
    git:
      url: https://github.com/tektoncd/pipeline
      revision: main
      pathInRepo: tekton/release-pipeline.yaml
  serviceAccountName: pipelines-release
  timeout: 
    pipeline: 10s
  params:
  - name: imageRegistry
    default: tekton-releases-nightly
    type: string
  workspaces:
  - name: workarea
    emptyDir: {}
  - name: release-secret
    secret:
      secretName: release-secret
`),
		want: MustParsePipelineRun(t, `
metadata:
  generateName: basic-workflow-run-
  namespace: some-namespace
spec:
  pipelineRef:
    resolver: git
    params:
    - name: url
      value: https://github.com/tektoncd/pipeline
    - name: revision
      value: main
    - name: pathInRepo
      value: tekton/release-pipeline.yaml
  params:
  - name: imageRegistry
    value: tekton-releases-nightly
  workspaces:
  - name: workarea
    emptyDir: {}
  - name: release-secret
    secret:
      secretName: release-secret
  serviceAccountName: pipelines-release
  timeouts:
    pipeline: 10s
`),
	}, {
		name: "workflow with pipelineSpec",
		workflow: MustParseWorkflow(t, "basic-workflow", "some-namespace", `
spec:
  pipeline:
    spec:
      tasks:
      - name: task-with-no-params
        taskRef: 
          name: some-task
`),
		want: MustParsePipelineRun(t, `
metadata:
  generateName: basic-workflow-run-
  namespace: some-namespace
spec:
  serviceAccountName: default 
  pipelineSpec:
    tasks:
    - name: task-with-no-params
      taskRef: 
        name: some-task 
`),
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := convert.ToPipelineRun(tt.workflow)
			if err != nil {
				t.Errorf("ToPipelineRun() error = %v", err)
				return
			}
			if diff := cmp.Diff(tt.want, got, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("ToPipelineRun() failed. Diff (-want/+got): %s", diff)
			}
		})
	}
}

func TestToTriggerTemplate(t *testing.T) {
	tests := []struct {
		name string
		w    *v1alpha1.Workflow
		want *triggersv1beta1.TriggerTemplate
	}{{
		name: "single trigger",
		w: MustParseWorkflow(t, "trigger-workflow", "some-namespace", `
spec:
  pipeline:
    git:
      url: https://github.com/tektoncd/pipeline
      revision: main
      pathInRepo: tekton/release-pipeline.yaml
  serviceAccountName: pipelines-release
  timeout: 
    pipeline: 10s
  params:
  - name: imageRegistry
    default: tekton-releases-nightly
    type: string
  workspaces:
  - name: workarea
    emptyDir: {}
  - name: release-secret
    secret:
      secretName: release-secret
`),
		want: MustParseTriggerTemplate(t, `
metadata:
  name: tt-trigger-workflow
  namespace: some-namespace
spec:
  params:
  - name: imageRegistry
    default: tekton-releases-nightly
  resourcetemplates:
  - apiVersion: tekton.dev/v1beta1
    kind: PipelineRun
    metadata:
      generateName: trigger-workflow-run-
      namespace: some-namespace
    spec:
      pipelineRef:
        resolver: git
        params:
        - name: url
          value: https://github.com/tektoncd/pipeline
        - name: revision
          value: main
        - name: pathInRepo
          value: tekton/release-pipeline.yaml
      params:
      - name: imageRegistry
        value: $(tt.params.imageRegistry)
      workspaces:
      - name: workarea
        emptyDir: {}
      - name: release-secret
        secret:
          secretName: release-secret
      serviceAccountName: pipelines-release
      timeouts:
        pipeline: 10s
`),
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := convert.ToTriggerTemplate(tt.w)
			if err != nil {
				t.Errorf("ToTriggerTemplate() error = %v", err)
				return
			}

			if diff := cmp.Diff(tt.want, got, compareResourcetemplates(t)); diff != "" {
				t.Errorf("ToTriggerTemplate() failed. Diff (-want/+got): = %v", diff)
			}
		})
	}
}

func TestToTriggers(t *testing.T) {
	tests := []struct {
		name string
		w    *v1alpha1.Workflow
		want []*triggersv1beta1.Trigger
	}{{
		name: "single trigger",
		w: MustParseWorkflow(t, "trigger-workflow", "some-namespace", `
spec:
  triggers:
  - name: on-pr
    event:
      type: "pull_request"
      secret:
        secretName: "repo-secret"
        secretKey: "token"
    interceptors:
    - name: "only_open_prs"
      ref:
        name: cel
      params:
      - name: "filter"
        value:  "body.action in ['opened', 'synchronize', 'reopened']"  
    bindings:
    - name: commit-sha
      value: $(body.pull_request.head.sha)
    - name: url
      value: $(body.repository.clone_url)
  pipeline:
    spec:
      tasks:
      - name: task-with-no-params
        taskRef:
          name: some-task
`),
		want: []*triggersv1beta1.Trigger{MustParseTrigger(t, `
metadata:
  name: trigger-workflow-on-pr
  namespace: tekton-workflows
  labels:
    managed-by: tekton-workflows
    tekton.dev/workflow: trigger-workflow
  ownerReferences:
  - apiVersion: tekton.dev/v1alpha1
    kind: Workflow
    name: trigger-workflow
    controller: true
    blockOwnerDeletion: true
spec:
  name: on-pr
  bindings:
  - name: commit-sha
    value: $(body.pull_request.head.sha)
  - name: url
    value: $(body.repository.clone_url)
  interceptors:
  - name: "validate-webhook"
    ref:
      name: github
      kind: ClusterInterceptor
    params:
    - name: secretRef
      value:
        secretName: repo-secret
        secretKey: token
    - name: eventTypes
      value: ["pull_request"]
  - name: "only_open_prs"
    ref:
      name: cel
    params:
    - name: "filter"
      value:  "body.action in ['opened', 'synchronize', 'reopened']" 
  template:
    spec:
      resourcetemplates:
      - apiVersion: tekton.dev/v1beta1
        kind: PipelineRun
        metadata:
          generateName: trigger-workflow-run-
          namespace: some-namespace
        spec:
          serviceAccountName: default
          pipelineSpec:
            tasks:
            - name: task-with-no-params
              taskRef: 
                name: some-task
`)},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := convert.ToTriggers(tt.w)
			if err != nil {
				t.Errorf("ToTriggers() error = %v", err)
				return
			}
			if diff := cmp.Diff(tt.want, got, compareResourcetemplates(t), cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("ToTriggers() failed. Diff -want/+got: %s", diff)
			}
		})
	}
}

func mustParseYAML(t *testing.T, yaml string, i runtime.Object) {
	if _, _, err := scheme.Codecs.UniversalDeserializer().Decode([]byte(yaml), nil, i); err != nil {
		t.Fatalf("mustParseYAML (%s): %v", yaml, err)
	}
}

// MustParsePipelineRun takes YAML and parses it into a *v1beta1.PipelineRun
func MustParsePipelineRun(t *testing.T, yaml string) *pipelinev1beta1.PipelineRun {
	var pr pipelinev1beta1.PipelineRun
	yaml = `apiVersion: tekton.dev/v1beta1
kind: PipelineRun
` + yaml
	mustParseYAML(t, yaml, &pr)
	return &pr
}

// MustParsePipelineRun takes YAML and parses it into a *v1beta1.PipelineRun
func MustParseWorkflow(t *testing.T, name, namespace, yaml string) *v1alpha1.Workflow {
	var w v1alpha1.Workflow
	yaml = fmt.Sprintf(`apiVersion: tekton.dev/v1alpha1
kind: Workflow
metadata:
  name: %s
  namespace: %s
`+yaml, name, namespace)
	mustParseYAML(t, yaml, &w)
	return &w
}

// MustParseTriggerTemplate takes YAML and parses it into a *triggersv1beta1.TriggerTemplate
func MustParseTriggerTemplate(t *testing.T, yaml string) *triggersv1beta1.TriggerTemplate {
	var pr triggersv1beta1.TriggerTemplate
	yaml = `apiVersion: triggers.tekton.dev/v1beta1
kind: TriggerTemplate
` + yaml
	mustParseYAML(t, yaml, &pr)
	return &pr
}

// MustParseTrigger takes YAML and parses it into a *triggersv1beta1.Trigger
func MustParseTrigger(t *testing.T, yaml string) *triggersv1beta1.Trigger {
	var pr triggersv1beta1.Trigger
	yaml = `apiVersion: triggers.tekton.dev/v1beta1
kind: Trigger
` + yaml
	mustParseYAML(t, yaml, &pr)
	return &pr
}

func templateToPipelineRun(t *testing.T, rt triggersv1beta1.TriggerResourceTemplate) *pipelinev1beta1.PipelineRun {
	var pr pipelinev1beta1.PipelineRun
	err := json.Unmarshal(rt.Raw, &pr)
	if err != nil {
		t.Fatalf("resourcetemplate json marshal failed: %s", err)
	}
	return &pr
}

// Assumes the resourceTemplates are for creating PipelineRuns
func compareResourcetemplates(t *testing.T) cmp.Option {
	return cmp.Comparer(func(x, y triggersv1beta1.TriggerResourceTemplate) bool {
		// Use cmp.Diff to print out resourcetemplate diffs here since those are hard to parse otherwise
		diff := cmp.Diff(
			templateToPipelineRun(t, x),
			templateToPipelineRun(t, y),
			cmpopts.IgnoreFields(pipelinev1beta1.PipelineRun{}, "ObjectMeta.CreationTimestamp", "Status"))
		if diff != "" {
			t.Errorf("resourcetemplate diff -want/+got: %s", diff)
			return false
		}
		return true
	})
}
