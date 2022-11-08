package convert_test

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/tektoncd/experimental/workflows/pkg/apis/workflows/v1alpha1"
	"github.com/tektoncd/experimental/workflows/pkg/convert"
	"github.com/tektoncd/experimental/workflows/test/parse"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	triggersv1beta1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
)

func TestToPipelineRun(t *testing.T) {
	tests := []struct {
		name     string
		workflow *v1alpha1.Workflow
		want     *pipelinev1beta1.PipelineRun
	}{{
		name: "workflow with pipeline ref",
		workflow: parse.MustParseWorkflow(t, "basic-workflow", "some-namespace", `
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
		want: parse.MustParsePipelineRun(t, `
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
		workflow: parse.MustParseWorkflow(t, "basic-workflow", "some-namespace", `
spec:
  pipelineSpec:
    tasks:
      - name: task-with-no-params
        taskRef: 
          name: some-task
`),
		want: parse.MustParsePipelineRun(t, `
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
		w: parse.MustParseWorkflow(t, "trigger-workflow", "some-namespace", `
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
		want: parse.MustParseTriggerTemplate(t, `
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
			got, err := convert.ToTriggerTemplate(tt.w, map[string]string{})
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
		name  string
		w     *v1alpha1.Workflow
		want  []*triggersv1beta1.Trigger
		repos []*v1alpha1.GitRepository
	}{{
		name: "single trigger",
		w: parse.MustParseWorkflow(t, "trigger-workflow", "some-namespace", `
spec:
  triggers:
  - name: on-pr
    event:
      types: ["pull_request"]
      secret:
        secretName: "repo-secret"
        secretKey: "token"
    bindings:
    - name: commit-sha
      value: $(body.pull_request.head.sha)
    - name: url
      value: $(body.repository.clone_url)
  pipelineSpec:
    tasks:
      - name: task-with-no-params
        taskRef:
          name: some-task
`),
		want: []*triggersv1beta1.Trigger{parse.MustParseTrigger(t, `
metadata:
  name: trigger-workflow-on-pr
  namespace: some-namespace
  labels:
    managed-by: tekton-workflows
    workflows.tekton.dev/workflow: trigger-workflow
  ownerReferences:
  - apiVersion: workflows.tekton.dev/v1alpha1
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
	}, {
		name: "with filters",
		w: parse.MustParseWorkflow(t, "trigger-workflow", "some-namespace", `
spec:
  triggers:
  - name: on-pr
    event:
      types: ["pull_request"]
      secret:
        secretName: "repo-secret"
        secretKey: "token"
    filters:
      gitRef:
        regex: '^main$' 
  pipelineSpec:
    tasks:
      - name: task-with-no-params
        taskRef:
          name: some-task
`),
		want: []*triggersv1beta1.Trigger{parse.MustParseTrigger(t, `
metadata:
  name: trigger-workflow-on-pr
  namespace: some-namespace
  labels:
    managed-by: tekton-workflows
    workflows.tekton.dev/workflow: trigger-workflow
  ownerReferences:
  - apiVersion: workflows.tekton.dev/v1alpha1
    kind: Workflow
    name: trigger-workflow
    controller: true
    blockOwnerDeletion: true
spec:
  name: on-pr
  interceptors:
  - name: "gitRef"
    ref:
      name: cel
      kind: ClusterInterceptor
    params:
    - name: "filter"
      value:  "body.ref.split('/')[2].matches('^main$')" 
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
	}, {
		name: "with repos",
		w: parse.MustParseWorkflow(t, "trigger-workflow", "some-namespace", `
spec:
  repos:
  - name: pipelines
  triggers:
  - name: on-pr
    event:
      types: ["pull_request"]
      source:
        repo: pipelines
  params:
  - name: repo-url
    default: $(repos.pipelines.url)
  pipelineSpec:
    tasks:
      - name: task-with-no-params
        taskRef:
          name: some-task
`),
		repos: []*v1alpha1.GitRepository{parse.MustParseRepo(t, "pipelines", "some-namespace", `
spec:
  url: https://tektoncd/pipeline
`)},
		want: []*triggersv1beta1.Trigger{parse.MustParseTrigger(t, `
metadata:
  name: trigger-workflow-on-pr
  namespace: some-namespace
  labels:
    managed-by: tekton-workflows
    workflows.tekton.dev/workflow: trigger-workflow
  ownerReferences:
  - apiVersion: workflows.tekton.dev/v1alpha1
    kind: Workflow
    name: trigger-workflow
    controller: true
    blockOwnerDeletion: true
spec:
  name: on-pr
  interceptors:
  - name: repo
    ref:
      name: cel
      kind: ClusterInterceptor
    params:
    - name: "filter"
      value:  "body.repository.html_url.matches('https://tektoncd/pipeline')" 
  template:
    spec:
      params:
      - name: repo-url
        default: https://tektoncd/pipeline 
      resourcetemplates:
      - apiVersion: tekton.dev/v1beta1
        kind: PipelineRun
        metadata:
          generateName: trigger-workflow-run-
          namespace: some-namespace
        spec:
          serviceAccountName: default
          params:
          - name: repo-url
            value: $(tt.params.repo-url)
          pipelineSpec:
            tasks:
            - name: task-with-no-params
              taskRef: 
                name: some-task
`)},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := convert.ToTriggers(tt.w, tt.repos)
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
