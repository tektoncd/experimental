package pipelinetotaskrun

import (
	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/experimental/pipeline-to-taskrun/test"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/test/diff"
	"testing"
)

func parsePipelineTaskInfo(t *testing.T, name, taskDeclaredParams, providedParamValues, steps, results string) PipelineTaskInfo {
	declaredTask := test.MustParseTask(t, `
spec:
  params:
`+taskDeclaredParams)
	providedP := test.MustParsePipeline(t, `
spec:
  tasks:
  - params:
`+providedParamValues)
	stepsTask := test.MustParseTask(t, `
spec:
  steps:
`+steps)
	resultsTask := test.MustParseTask(t, `
spec:
  results:
`+results)
	return PipelineTaskInfo{
		Name:                name,
		TaskDeclaredParams:  declaredTask.Spec.Params,
		ProvidedParamValues: providedP.Spec.Tasks[0].Params,
		Steps:               stepsTask.Spec.Steps,
		Results:             resultsTask.Spec.Results,
	}
}

func TestNewPipelineTaskInfo(t *testing.T) {
	for _, tc := range []struct {
		Name     string
		Pipeline *v1beta1.Pipeline
		Task     *v1beta1.Task
		Expected PipelineTaskInfo
	}{{
		Name: "complete example",
		Pipeline: test.MustParsePipeline(t, `
spec:
  tasks:
  - name: run-tests
    params:
    - name: package
      value: $(params.package)
`),
		Task: test.MustParseTask(t, `
spec:
  params:
  - name: package
    description: "some package"
  - name: something
    default: ""
  steps:
  - name: clone
    image: some-git-image
    script: |

      #!/usr/bin/env bash
      set -xe
      /ko-app/git-init \
        -url "$(params.package)" \
        -revision "$(params.something)"

  - name: some-other-step-really-long-gonna-get-truncated-so-very-long
    image: someimage
  results:
  - name: commit
    description: "The precise commit SHA"
`),
		Expected: parsePipelineTaskInfo(t, "run-tests", `
  - name: package
    description: "some package"
  - name: something
    default: ""
`, `
    - name: package
      value: $(params.package)
`, `
  - name: clone
    image: some-git-image
    script: |

      #!/usr/bin/env bash
      set -xe
      /ko-app/git-init \
        -url "$(params.package)" \
        -revision "$(params.something)"

  - name: some-other-step-really-long-gonna-get-truncated-so-very-long
    image: someimage
`, `
  - name: commit
    description: "The precise commit SHA"
`),
	}, {
		Name: "no params, results, or script",
		Pipeline: test.MustParsePipeline(t, `
spec:
  tasks:
  - name: run-tests
`),
		Task: test.MustParseTask(t, `
spec:
  params:
  steps:
  - name: clone
    image: ubuntu
`),
		Expected: parsePipelineTaskInfo(t, "run-tests", "", "", `
  - name: clone
    image: ubuntu
`, ""),
	}} {
		t.Run(tc.Name, func(t *testing.T) {
			taskSpecs := map[string]*v1beta1.TaskSpec{
				tc.Pipeline.Spec.Tasks[0].Name: &tc.Task.Spec,
			}
			pti, err := NewPipelineTaskInfo(tc.Pipeline.Spec.Tasks[0], taskSpecs)
			if err != nil {
				t.Fatalf("Didn't expect error but got %v", err)
			}
			if d := cmp.Diff(tc.Expected, pti); d != "" {
				t.Errorf("Didn't get expected object. Diff: %s", diff.PrintWantGot(d))
			}
		})
	}
}

func TestApplyPipelineLevelParams(t *testing.T) {
	run := test.MustParseRun(t, `
spec:
  params:
  - name: git-url
    value: https://github.com/tektoncd/chains
  - name: package
    value: github.com/tektoncd/chains/pkg
  - name: packages
    value: ./pkg/...
  - name: gcs-location
    value: gs://christies-empty-bucket
`)
	var pTasks []v1beta1.PipelineTask
	p := test.MustParsePipeline(t, `
spec:
  tasks:
  - name: grab-source
    params:
    - name: url
      value: $(params.git-url)
  - name: run-tests
    params:
    - name: package
      value: $(params.package)
    - name: packages
      value: $(params.packages) > $(workspaces.source.path)/test-results
  - name: upload-results
    params:
    - name: path
      value: test-results
    - name: location
      value: $(params.gcs-location)
`)
	for _, pt := range p.Spec.Tasks {
		pTasks = append(pTasks, pt)
	}
	expectedP := test.MustParsePipeline(t, `
spec:
  tasks:
  - name: grab-source
    params:
    - name: url
      value: https://github.com/tektoncd/chains
  - name: run-tests
    params:
    - name: package
      value: github.com/tektoncd/chains/pkg 
    - name: packages
      value: ./pkg/... > $(workspaces.source.path)/test-results
  - name: upload-results
    params:
    - name: path
      value: test-results
    - name: location
      value: gs://christies-empty-bucket
`)
	modifiedTasks := applyPipelineLevelParams(pTasks, run.Spec.Params)
	if d := cmp.Diff(expectedP.Spec.Tasks, modifiedTasks); d != "" {
		t.Errorf("Resulting taskrun spec didn't match expectations (-want, +got): %s", d)
	}
}

func TestNamespaceParams(t *testing.T) {
	for _, tc := range []struct {
		Name             string
		PipelineTaskInfo PipelineTaskInfo
		Expected         PipelineTaskInfo
	}{{
		Name: "grab-source and refer to params within params",
		PipelineTaskInfo: parsePipelineTaskInfo(t, "grab-source", `
  - name: url
    description: "git url to clone"
  - name: revision
    description: "git revision to check out"
    default: ""
`, `
    - name: url
      value: https://github.com/tektoncd/chains $(params.url)
`, `
  - name: clone
    image: some-git-image
    script: |
      #!/usr/bin/env bash
      set -xe
      /ko-app/git-init \
        -url "$(params.url)" \
        -revision "$(params.revision)"
  - name: some-other-step-really-long-gonna-get-truncated-so-very-long
    image: someimage
`, `
  - name: commit
    description: "The precise commit SHA that was fetched by this Task"
`),
		Expected: parsePipelineTaskInfo(t, "grab-source", `
  - name: grab-source-url
    description: "git url to clone"
  - name: grab-source-revision
    description: "git revision to check out"
    default: ""
`, `
    - name: grab-source-url
      value: https://github.com/tektoncd/chains $(params.grab-source-url)
`, `
  - name: clone
    image: some-git-image
    script: |
      #!/usr/bin/env bash
      set -xe
      /ko-app/git-init \
        -url "$(params.grab-source-url)" \
        -revision "$(params.grab-source-revision)"
  - name: some-other-step-really-long-gonna-get-truncated-so-very-long
    image: someimage
`, `
  - name: commit
    description: "The precise commit SHA that was fetched by this Task"
`),
	}, {
		Name: "run-tests",
		PipelineTaskInfo: parsePipelineTaskInfo(t, "run-tests", `
  - name: package
    description: "package (and its children) under test"
  - name: packages
    description: "packages to test (default: ./...)"
    default: "./..."
  - name: context
    description: "path to the directory to use as context."
    default: "."
`, `
    - name: package
      value: github.com/tektoncd/chains/pkg 
    - name: packages
      value: ./pkg/... > $(workspaces.source.path)/test-results
`, `
  - name: unit-test
    image: "docker.io/library/golang"
    env:
    - name: CONTEXT
      value: $(params.context)
    script: |
      SRC_PATH="$GOPATH/src/$(params.package)/$(params.context)"
      mkdir -p $SRC_PATH
      cp -R "$(workspaces.source.path)"/"$(params.context)"/* $SRC_PATH
      cd $SRC_PATH
`, ""),
		Expected: parsePipelineTaskInfo(t, "run-tests", `
  - name: run-tests-package
    description: "package (and its children) under test"
  - name: run-tests-packages
    description: "packages to test (default: ./...)"
    default: "./..."
  - name: run-tests-context
    description: "path to the directory to use as context."
    default: "."
`, `
    - name: run-tests-package
      value: "github.com/tektoncd/chains/pkg"
    - name: run-tests-packages
      value: "./pkg/... > $(workspaces.source.path)/test-results"
`, `
  - name: unit-test
    image: "docker.io/library/golang"
    env:
    - name: CONTEXT
      value: $(params.run-tests-context)
    script: |
      SRC_PATH="$GOPATH/src/$(params.run-tests-package)/$(params.run-tests-context)"
      mkdir -p $SRC_PATH
      cp -R "$(workspaces.source.path)"/"$(params.run-tests-context)"/* $SRC_PATH
      cd $SRC_PATH
`, ""),
	}, {
		Name: "upload-results",
		PipelineTaskInfo: parsePipelineTaskInfo(t, "upload-results", `
  - name: path
    description: "The path to files or directories relative to the source workspace that you'd like to upload."
  - name: location
    description: "The address (including \"gs://\") where you'd like to upload files to."
  - name: serviceAccountPath
    description: "The path inside the credentials workspace to the GOOGLE_APPLICATION_CREDENTIALS key file."
    default: "service_account.json"
`, `
    - name: path
      value: test-results
    - name: location
      value: gs://christies-empty-bucket
`, `
  - name: upload
    image: "gcr.io/google.com/cloudsdktool/cloud-sdk:310.0.0"
    script: |
      #!/usr/bin/env bash
      set -xe
      CRED_PATH="$(workspaces.credentials.path)/$(params.serviceAccountPath)"
      SOURCE="$(workspaces.source.path)/$(params.path)"
`, ""),
		Expected: parsePipelineTaskInfo(t, "upload-results", `
  - name: upload-results-path
    description: "The path to files or directories relative to the source workspace that you'd like to upload."
  - name: upload-results-location
    description: "The address (including \"gs://\") where you'd like to upload files to."
  - name: upload-results-serviceAccountPath
    description: "The path inside the credentials workspace to the GOOGLE_APPLICATION_CREDENTIALS key file."
    default: service_account.json
`, `
    - name: upload-results-path
      value: "test-results"
    - name: upload-results-location
      value: "gs://christies-empty-bucket"
`, `
  - name: upload
    image: "gcr.io/google.com/cloudsdktool/cloud-sdk:310.0.0"
    script: |
      #!/usr/bin/env bash
      set -xe
      CRED_PATH="$(workspaces.credentials.path)/$(params.upload-results-serviceAccountPath)"
      SOURCE="$(workspaces.source.path)/$(params.upload-results-path)"
`, ""),
	}} {
		t.Run(tc.Name, func(t *testing.T) {
			updatedPti := tc.PipelineTaskInfo.NamespaceParams()
			if d := cmp.Diff(tc.Expected, updatedPti); d != "" {
				t.Errorf("didn't get expected updated info. Diff: %s", diff.PrintWantGot(d))
			}
		})
	}
}

func TestNamespaceSteps(t *testing.T) {
	pti := parsePipelineTaskInfo(t, "grab-source", `
  - name: grab-source-url
    description: "git url to clone"
`, `
    - name: grab-source-url
      value: "https://github.com/tektoncd/chains"
`, `
  - name: clone
    image: some-git-image
    script: |
      echo $(workspaces.foobar.bound)
  - image: ubuntu
`, `
  - name: commit
    description: "The precise commit SHA that was fetched by this Task"
`)
	expected := parsePipelineTaskInfo(t, "grab-source", `
  - name: grab-source-url
    description: "git url to clone"
`, `
    - name: grab-source-url
      value: "https://github.com/tektoncd/chains"
`, `
  - name: grab-source-clone
    image: some-git-image
    script: |
      echo $(workspaces.foobar.bound)
  - image: ubuntu
`, `
  - name: commit
    description: "The precise commit SHA that was fetched by this Task"
`)
	updatedPti := pti.NamespaceSteps()
	if d := cmp.Diff(expected, updatedPti); d != "" {
		t.Errorf("didn't get expected updated info. Diff: %s", diff.PrintWantGot(d))
	}
}

func TestRenameWorkspaces(t *testing.T) {
	pti := parsePipelineTaskInfo(t, "grab-source", `
  - name: grab-source-url
    description: "git url to clone"
`, `
    - name: grab-source-url
      value: "https://github.com/tektoncd/chains $(workspaces.foobar.path)"
`, `
  - name: grab-source-clone
    image: some-git-image
    script: |
      echo $(workspaces.optional.bound)
      echo $(workspaces.foobar.bound)
      echo $(workspaces.foobar.claim)
      echo $(workspaces.foobar.volume)
      cd $(workspaces.foobar.path)
`, `
  - name: commit
    description: "The precise commit SHA that was fetched by this Task"
`)
	expected := parsePipelineTaskInfo(t, "grab-source", `
  - name: grab-source-url
    description: "git url to clone"
`, `
    - name: grab-source-url
      value: "https://github.com/tektoncd/chains $(workspaces.the-ultimate-volume.path)"
`, `
  - name: grab-source-clone
    image: some-git-image
    script: |
      echo $(workspaces.optional.bound)
      echo $(workspaces.the-ultimate-volume.bound)
      echo $(workspaces.the-ultimate-volume.claim)
      echo $(workspaces.the-ultimate-volume.volume)
      cd $(workspaces.the-ultimate-volume.path)
`, `
  - name: commit
    description: "The precise commit SHA that was fetched by this Task"
`)
	newMapping := map[string]string{"foobar": "the-ultimate-volume", "optional": ""}
	updatedPti := pti.RenameWorkspaces(newMapping)
	if d := cmp.Diff(expected, updatedPti); d != "" {
		t.Errorf("didn't get expected updated info. Diff: %s", diff.PrintWantGot(d))
	}
}
